package memory

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// Caps2 almacena memoria episódica: cada chunk de conversación se embebe y guarda en Qdrant.
// También provee búsqueda semántica para inyectar contexto relevante en el prompt.

type Caps2Entry struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Source    string `json:"source"`    // "chat", "import", "project"
	SourceID  string `json:"sourceId"`  // chat_id, project_id, etc.
	UserID    string `json:"userId"`
	ProjectID string `json:"projectId,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type Caps2Store struct {
	embedCli *EmbeddingClient
	qdrant   *QdrantClient
}

var caps2 *Caps2Store

func GetCaps2() *Caps2Store {
	if caps2 == nil {
		caps2 = &Caps2Store{
			embedCli: GetEmbeddingClient(),
			qdrant:   GetQdrantClient(),
		}
	}
	return caps2
}

// Store chunklea, embebe y guarda en Qdrant.
func (c *Caps2Store) Store(entry Caps2Entry) error {
	chunks := ChunkText(entry.Content)

	for _, chunk := range chunks {
		vector, err := c.embedCli.Embed(chunk.Content)
		if err != nil {
			log.Printf("Caps2 embed error (will skip): %v", err)
			return fmt.Errorf("embed: %w", err)
		}

		payload := map[string]interface{}{
			"content":   chunk.Content,
			"source":    entry.Source,
			"sourceId":  entry.SourceID,
			"userId":    entry.UserID,
			"projectId": entry.ProjectID,
			"timestamp": entry.Timestamp,
			"chunkIdx":  chunk.Index,
		}

		pointID := uuid.NewString()
		if err := c.qdrant.Insert(pointID, vector, payload); err != nil {
			log.Printf("Caps2 Qdrant insert error (will skip): %v", err)
			return fmt.Errorf("qdrant insert: %w", err)
		}
	}
	return nil
}

// Search busca los top-k chunks más relevantes por similitud semántica.
func (c *Caps2Store) Search(query string, limit int, projectID, userID string) ([]string, error) {
	vector, err := c.embedCli.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	filter := map[string]interface{}{}
	if projectID != "" {
		filter["must"] = []map[string]interface{}{
			{"key": "projectId", "match": map[string]interface{}{"value": projectID}},
		}
	}
	if userID != "" {
		filter["must"] = []map[string]interface{}{
			{"key": "userId", "match": map[string]interface{}{"value": userID}},
		}
	}

	results, err := c.qdrant.Search(vector, limit, filter)
	if err != nil {
		return nil, fmt.Errorf("qdrant search: %w", err)
	}

	var texts []string
	for _, r := range results {
		if content, ok := r.Payload["content"].(string); ok {
			texts = append(texts, content)
		}
	}
	return texts, nil
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
