package memory

import (
	"fmt"
	"log"
	"time"

	"github.com/ozyassist/backend/internal/db"
)

type Caps2Entry struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Source    string `json:"source"`    // "chat", "import", "project", "system_index"
	SourceID  string `json:"sourceId"`  // chat_id, project_id, etc.
	UserID    string `json:"userId"`
	ProjectID string `json:"projectId,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type Caps2Store struct {
	embedCli *EmbeddingClient
	ram      *RAMClient
}

var caps2 *Caps2Store

func GetCaps2() *Caps2Store {
	if caps2 == nil {
		caps2 = &Caps2Store{
			embedCli: GetEmbeddingClient(),
			ram:      GetVectorClient(),
		}
	}
	return caps2
}

// Store chunklea, embebe y guarda en RAM y SQLite.
func (c *Caps2Store) Store(entry Caps2Entry) error {
	chunks := ChunkText(entry.Content)

	for _, chunk := range chunks {
		pointID := fmt.Sprintf("%s-%d", entry.ID, chunk.Index)
		payload := map[string]interface{}{
			"content":   chunk.Content,
			"source":    entry.Source,
			"sourceId":  entry.SourceID,
			"userId":    entry.UserID,
			"projectId": entry.ProjectID,
			"timestamp": entry.Timestamp,
			"chunkIdx":  chunk.Index,
		}

		// 1. Intentar generar embedding con Ollama / local
		vector, err := c.embedCli.Embed(chunk.Content)
		if err != nil {
			// Si Ollama no está activo, no bloqueamos: guardamos texto en SQLite
			log.Printf("[Memoria] Ollama local no disponible para embedding (%v). Continuando con persistencia en SQLite.", err)
			vector = nil
		}

		// 2. Si tenemos vector, guardar en RAM y persistir en SQLite
		if len(vector) > 0 {
			_ = c.ram.Insert(pointID, vector, payload)
			if entry.Source != "system_index" {
				_ = c.ram.SaveToSQLite(pointID, entry.UserID, entry.ProjectID, entry.Source, entry.SourceID, chunk.Content, vector)
			}
		}

		// 3. Registrar también en memory_entries para búsqueda léxica FTS5
		if db.DB != nil && entry.Source != "system_index" {
			_, _ = db.DB.Exec(`INSERT OR IGNORE INTO memory_entries (id, user_id, project_id, source, source_id, topic, content)
			                   VALUES (?, ?, ?, ?, ?, ?, ?)`,
				pointID, entry.UserID, entry.ProjectID, entry.Source, entry.SourceID, "chat", chunk.Content)
		}
	}
	return nil
}

// Search busca usando Similitud Semántica en RAM + Búsqueda Léxica FTS5 en SQLite.
func (c *Caps2Store) Search(query string, limit int, projectID, userID string) ([]string, error) {
	if limit <= 0 {
		limit = 5
	}

	seen := make(map[string]bool)
	var results []string

	// 1. Búsqueda semántica en RAM (si Ollama está disponible)
	vector, err := c.embedCli.Embed(query)
	if err == nil && len(vector) > 0 {
		filter := map[string]interface{}{}
		var andConditions []map[string]interface{}
		if projectID != "" {
			andConditions = append(andConditions, map[string]interface{}{"projectId": projectID})
		}
		if userID != "" {
			andConditions = append(andConditions, map[string]interface{}{"userId": userID})
		}

		if len(andConditions) == 1 {
			for k, v := range andConditions[0] {
				filter[k] = v
			}
		} else if len(andConditions) > 1 {
			filter["$and"] = andConditions
		}

		ramResults, _ := c.ram.Search(vector, limit, filter)
		for _, r := range ramResults {
			if content, ok := r.Payload["content"].(string); ok && content != "" && !seen[content] {
				seen[content] = true
				results = append(results, content)
			}
		}
	}

	// 2. Si faltan resultados o Ollama no estaba encendido, consultar SQLite FTS5 (BM25)
	if len(results) < limit && db.DB != nil {
		ftsQuery := sanitizeFTSQuery(query)
		if ftsQuery != "" {
			q := `SELECT m.content FROM memory_entries m
			      JOIN memory_entries_fts f ON m.rowid = f.rowid
			      WHERE memory_entries_fts MATCH ?
			      ORDER BY rank LIMIT ?`
			rows, err := db.DB.Query(q, ftsQuery, limit-len(results))
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var text string
					if err := rows.Scan(&text); err == nil && !seen[text] {
						seen[text] = true
						results = append(results, text)
					}
				}
			}
		}
	}

	return results, nil
}

// StoreChatMessage guarda automáticamente un mensaje de chat en memoria episódica.
func StoreChatMessage(userID, projectID, chatID, role, content string) {
	if content == "" {
		return
	}
	entry := Caps2Entry{
		ID:        fmt.Sprintf("msg-%s-%d", chatID, time.Now().UnixNano()),
		Content:   fmt.Sprintf("[%s] %s", role, content),
		Source:    "chat",
		SourceID:  chatID,
		UserID:    userID,
		ProjectID: projectID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := GetCaps2().Store(entry); err != nil {
		log.Printf("StoreChatMessage: %v", err)
	}
}
