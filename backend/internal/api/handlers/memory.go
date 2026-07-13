package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/memory"
)

func ImportMemory(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	userID := db.DefaultUserID()

	chunks := memory.ChunkText(req.Content)
	feedback := fmt.Sprintf("Importados %d fragmentos de memoria.", len(chunks))

	for i, chunk := range chunks {
		entry := models.MemoryEntry{
			ID:        uuid.NewString(),
			UserID:    userID,
			Source:    "import",
			Topic:     fmt.Sprintf("Memory import chunk %d", i+1),
			Content:   chunk.Content,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		if err := db.CreateMemoryEntry(&entry); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		memEntry := memory.Caps2Entry{
			ID:        entry.ID,
			Content:   chunk.Content,
			Source:    "import",
			SourceID:  entry.ID,
			UserID:    userID,
			Timestamp: entry.CreatedAt,
		}
		if err := memory.GetCaps2().Store(memEntry); err != nil {
			// non-fatal: Qdrant may not be running
			feedback += " (solo guardado local, sin vectores)"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"chunks":   len(chunks),
		"feedback": feedback,
	})
}

func SearchMemory(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query param 'q' is required"})
		return
	}

	userID := db.DefaultUserID()

	results, err := db.FTS5SearchMemory(query, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]gin.H, 0, len(results))
	for _, r := range results {
		response = append(response, gin.H{
			"id":    r.ID,
			"text":  r.Snippet,
			"score": r.Score,
			"topic": r.Title,
		})
	}

	// Also try vector search as enrichment (non-critical)
	if vecResults, err := memory.GetCaps2().Search(query, 3, "", userID); err == nil {
		for _, v := range vecResults {
			response = append(response, gin.H{
				"id":    "vec-" + uuid.NewString(),
				"text":  v,
				"score": 0.0,
				"topic": "semantic match",
			})
		}
	}

	if response == nil {
		response = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"results": response})
}

func UpdateMessageFeedback(c *gin.Context) {
	messageID := c.Param("messageId")
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messageId is required"})
		return
	}

	var req struct {
		Feedback string `json:"feedback"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if req.Feedback != "" && req.Feedback != "like" && req.Feedback != "dislike" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "feedback must be empty, 'like', or 'dislike'"})
		return
	}

	if err := db.UpdateMessageFeedback(messageID, req.Feedback); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
