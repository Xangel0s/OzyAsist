package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

const defaultBaseURL = "http://localhost:1234/v1"

var embedOnce sync.Once
var embedClient *EmbeddingClient

func GetEmbeddingClient() *EmbeddingClient {
	embedOnce.Do(func() {
		baseURL := os.Getenv("LMSTUDIO_URL")
		if baseURL == "" {
			baseURL = defaultBaseURL
		}
		embedClient = &EmbeddingClient{
			baseURL: baseURL,
			client:  &http.Client{},
		}
	})
	return embedClient
}

type EmbeddingClient struct {
	baseURL string
	client  *http.Client
}

type embedRequest struct {
	Model string  `json:"model"`
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
	body := embedRequest{
		Model: "text-embedding-nomic-embed-text-v1.5",
		Input: texts,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	resp, err := ec.client.Post(ec.baseURL+"/embeddings", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embed response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embed API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result embedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse embed response: %w", err)
	}

	embeddings := make([][]float64, len(result.Data))
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}
	return embeddings, nil
}

func (ec *EmbeddingClient) Dimension() int {
	return 768 // nomic-embed-text-v1.5
}
