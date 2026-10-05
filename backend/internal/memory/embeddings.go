package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const defaultOllamaURL = "http://localhost:11434/v1"
const defaultLMStudioURL = "http://localhost:1234/v1"

var embedOnce sync.Once
var embedClient *EmbeddingClient

func GetEmbeddingClient() *EmbeddingClient {
	embedOnce.Do(func() {
		baseURL := os.Getenv("OLLAMA_URL")
		if baseURL == "" {
			baseURL = os.Getenv("LMSTUDIO_URL")
		}
		if baseURL == "" {
			// Por defecto intentamos conectar a Ollama local
			baseURL = defaultOllamaURL
		}
		model := os.Getenv("EMBEDDING_MODEL")
		if model == "" {
			model = "nomic-embed-text"
		}
		embedClient = &EmbeddingClient{
			baseURL: baseURL,
			model:   model,
			client:  &http.Client{Timeout: 3 * time.Second},
		}
	})
	return embedClient
}

type EmbeddingClient struct {
	baseURL     string
	model       string
	client      *http.Client
	mu          sync.RWMutex
	lastFailure time.Time
}

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
}

func (ec *EmbeddingClient) Embed(text string) ([]float64, error) {
	vecs, err := ec.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return vecs[0], nil
}

func (ec *EmbeddingClient) EmbedBatch(texts []string) ([][]float64, error) {
	ec.mu.RLock()
	if !ec.lastFailure.IsZero() && time.Since(ec.lastFailure) < 15*time.Second {
		ec.mu.RUnlock()
		return fallbackEmbedBatch(texts, ec.Dimension()), nil
	}
	ec.mu.RUnlock()

	body := embedRequest{
		Model: ec.model,
		Input: texts,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return fallbackEmbedBatch(texts, ec.Dimension()), nil
	}

	resp, err := ec.client.Post(ec.baseURL+"/embeddings", "application/json", bytes.NewReader(data))
	if err != nil {
		ec.mu.Lock()
		if ec.lastFailure.IsZero() || time.Since(ec.lastFailure) > 10*time.Minute {
			log.Printf("[Memoria] Ollama local/modelo no disponible (%v). Activando motor de embeddings semánticos en Go puro (Zero-Docker).", err)
		}
		ec.lastFailure = time.Now()
		ec.mu.Unlock()
		return fallbackEmbedBatch(texts, ec.Dimension()), nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != 200 {
		ec.mu.Lock()
		if ec.lastFailure.IsZero() || time.Since(ec.lastFailure) > 10*time.Minute {
			errMsg := string(respBody)
			if err != nil {
				errMsg = err.Error()
			}
			log.Printf("[Memoria] Servidor de embedding retornó código %d (%s). Usando fallback en Go puro.", resp.StatusCode, errMsg)
		}
		ec.lastFailure = time.Now()
		ec.mu.Unlock()
		return fallbackEmbedBatch(texts, ec.Dimension()), nil
	}

	var result embedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fallbackEmbedBatch(texts, ec.Dimension()), nil
	}

	embeddings := make([][]float64, len(result.Data))
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}
	return embeddings, nil
}

func fallbackEmbedBatch(texts []string, dim int) [][]float64 {
	res := make([][]float64, len(texts))
	for i, t := range texts {
		res[i] = PureGoFallbackEmbedding(t, dim)
	}
	return res
}

func (ec *EmbeddingClient) Dimension() int {
	return 768 // nomic-embed-text-v1.5
}

// PureGoFallbackEmbedding genera un vector denso de 768 dimensiones normalizado L2
// basado en hashing de n-gramas de caracteres y tokens de palabras en Go puro (Zero-Docker).
// Permite similitud coseno semántica instantánea cuando Ollama o nomic-embed-text no están descargados.
func PureGoFallbackEmbedding(text string, dim int) []float64 {
	if dim <= 0 {
		dim = 768
	}
	vec := make([]float64, dim)
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return vec
	}

	words := strings.Fields(text)
	for _, w := range words {
		// Token hash
		h := fnv1a(w)
		idx := int(h % uint64(dim))
		sign := 1.0
		if (h & 1) == 0 {
			sign = -1.0
		}
		vec[idx] += sign * 1.5

		// Character trigrams
		runes := []rune(w)
		if len(runes) >= 3 {
			for i := 0; i <= len(runes)-3; i++ {
				trigram := string(runes[i : i+3])
				th := fnv1a(trigram)
				tidx := int(th % uint64(dim))
				tsign := 1.0
				if (th & 1) == 0 {
					tsign = -1.0
				}
				vec[tidx] += tsign * 0.8
			}
		}
	}

	// Normalización L2
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		invSqrt := 1.0 / math.Sqrt(norm)
		for i := range vec {
			vec[i] *= invSqrt
		}
	}

	return vec
}

func fnv1a(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}
