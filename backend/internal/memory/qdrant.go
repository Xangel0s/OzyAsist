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

const defaultQdrantURL = "http://localhost:6333"

var qdrantOnce sync.Once
var qdrantClient *QdrantClient

func GetQdrantClient() *QdrantClient {
	qdrantOnce.Do(func() {
		baseURL := os.Getenv("QDRANT_URL")
		if baseURL == "" {
			baseURL = defaultQdrantURL
		}
		qdrantClient = &QdrantClient{
			baseURL: baseURL,
			client:  &http.Client{},
		}
		_ = qdrantClient.ensureCollection()
	})
	return qdrantClient
}

type QdrantClient struct {
	baseURL string
	client  *http.Client
}

type qdrantPoint struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector"`
	Payload  map[string]interface{} `json:"payload"`
}

type qdrantUpsertRequest struct {
	Batch qdrantBatch `json:"batch"`
}

type qdrantBatch struct {
	IDs      []string                 `json:"ids"`
	Vectors  [][]float64              `json:"vectors"`
	Payloads []map[string]interface{} `json:"payloads,omitempty"`
}

type qdrantSearchRequest struct {
	Vector      []float64               `json:"vector"`
	Limit       int                     `json:"limit"`
	Filter      *map[string]interface{} `json:"filter,omitempty"`
	WithPayload bool                    `json:"with_payload"`
}

type qdrantScoredPoint struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

type qdrantSearchResponse struct {
	Result []qdrantScoredPoint `json:"result"`
}

func (qc *QdrantClient) ensureCollection() error {
	// PUT /collections/ozy_memory — no-op if exists
	body := map[string]interface{}{
		"name": "ozy_memory",
		"vectors": map[string]interface{}{
			"size":     768,
			"distance": "Cosine",
		},
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("PUT", qc.baseURL+"/collections/ozy_memory", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := qc.client.Do(req)
	if err != nil {
		return fmt.Errorf("Qdrant create collection: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func (qc *QdrantClient) Insert(id string, vector []float64, payload map[string]interface{}) error {
	return qc.InsertBatch([]qdrantPoint{{
		ID:      id,
		Vector:  vector,
		Payload: payload,
	}})
}

func (qc *QdrantClient) InsertBatch(points []qdrantPoint) error {
	req := qdrantUpsertRequest{
		Batch: qdrantBatch{
			IDs:      make([]string, 0, len(points)),
			Vectors:  make([][]float64, 0, len(points)),
			Payloads: make([]map[string]interface{}, 0, len(points)),
		},
	}
	for _, p := range points {
		req.Batch.IDs = append(req.Batch.IDs, p.ID)
		req.Batch.Vectors = append(req.Batch.Vectors, p.Vector)
		req.Batch.Payloads = append(req.Batch.Payloads, p.Payload)
	}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal upsert: %w", err)
	}

	httpReq, err := http.NewRequest("PUT", qc.baseURL+"/collections/ozy_memory/points?wait=true", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create upsert request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := qc.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Qdrant upsert: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Qdrant upsert error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (qc *QdrantClient) Search(vector []float64, limit int, filter map[string]interface{}) ([]qdrantScoredPoint, error) {
	body := qdrantSearchRequest{
		Vector:      vector,
		Limit:       limit,
		WithPayload: true,
	}
	if len(filter) > 0 {
		body.Filter = &filter
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal search: %w", err)
	}

	resp, err := qc.client.Post(qc.baseURL+"/collections/ozy_memory/points/search", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("Qdrant search: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read search response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Qdrant search error %d: %s", resp.StatusCode, string(respBody))
	}

	var result qdrantSearchResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}
	return result.Result, nil
}
