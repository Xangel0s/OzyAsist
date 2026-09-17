package memory

import (
	"encoding/json"
	"log"
	"math"
	"sort"
	"sync"

	"github.com/ozyassist/backend/internal/db"
)

type VectorPoint struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector"`
	Payload  map[string]interface{} `json:"payload"`
}

type ScoredPoint struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

var ramOnce sync.Once
var ramClient *RAMClient

func GetVectorClient() *RAMClient {
	ramOnce.Do(func() {
		ramClient = &RAMClient{
			points: make(map[string]VectorPoint),
		}
		// Cargar recuerdos persistidos de SQLite a la RAM al iniciar
		if err := ramClient.LoadFromSQLite(); err != nil {
			log.Printf("[RAM VectorStore] Aviso cargando de SQLite: %v", err)
		}
	})
	return ramClient
}

type RAMClient struct {
	mu     sync.RWMutex
	points map[string]VectorPoint
}

// LoadFromSQLite carga todos los vectores guardados en la tabla episodic_vectors a la RAM.
func (r *RAMClient) LoadFromSQLite() error {
	if db.DB == nil {
		return nil
	}

	rows, err := db.DB.Query("SELECT id, user_id, project_id, source, source_id, content, vector_json FROM episodic_vectors")
	if err != nil {
		return err
	}
	defer rows.Close()

	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for rows.Next() {
		var id, userID, projectID, source, sourceID, content, vectorJSON string
		if err := rows.Scan(&id, &userID, &projectID, &source, &sourceID, &content, &vectorJSON); err != nil {
			continue
		}

		var vec []float64
		if err := json.Unmarshal([]byte(vectorJSON), &vec); err != nil {
			continue
		}

		payload := map[string]interface{}{
			"content":   content,
			"source":    source,
			"sourceId":  sourceID,
			"userId":    userID,
			"projectId": projectID,
		}

		r.points[id] = VectorPoint{
			ID:      id,
			Vector:  vec,
			Payload: payload,
		}
		count++
	}

	if count > 0 {
		log.Printf("[RAM VectorStore] %d recuerdos cargados exitosamente de SQLite a la memoria RAM.", count)
	}
	return nil
}

// SaveToSQLite persiste un vector en SQLite para que perdure entre reinicios.
func (r *RAMClient) SaveToSQLite(id, userID, projectID, source, sourceID, content string, vector []float64) error {
	if db.DB == nil {
		return nil
	}

	vecBytes, err := json.Marshal(vector)
	if err != nil {
		return err
	}

	query := `INSERT OR REPLACE INTO episodic_vectors (id, user_id, project_id, source, source_id, content, vector_json)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = db.DB.Exec(query, id, userID, projectID, source, sourceID, content, string(vecBytes))
	return err
}

func (r *RAMClient) Insert(id string, vector []float64, payload map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.points[id] = VectorPoint{
		ID:      id,
		Vector:  vector,
		Payload: payload,
	}
	return nil
}

func (r *RAMClient) InsertBatch(points []VectorPoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range points {
		r.points[p.ID] = p
	}
	return nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (r *RAMClient) Search(vector []float64, limit int, filter map[string]interface{}) ([]ScoredPoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ScoredPoint

	for _, p := range r.points {
		match := true
		if andConditions, ok := filter["$and"].([]map[string]interface{}); ok {
			for _, cond := range andConditions {
				for k, v := range cond {
					if p.Payload[k] != v {
						match = false
						break
					}
				}
				if !match {
					break
				}
			}
		} else {
			for k, v := range filter {
				if p.Payload[k] != v {
					match = false
					break
				}
			}
		}

		if match {
			score := cosineSimilarity(vector, p.Vector)
			results = append(results, ScoredPoint{
				ID:      p.ID,
				Score:   score,
				Payload: p.Payload,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}
