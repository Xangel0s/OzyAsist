package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/providers"
)

func AnalyzeMemory(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	prov := pickFirstProvider()
	if prov == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no provider available"})
		return
	}

	preview := req.Content
	if len(preview) > 2000 {
		preview = preview[:2000] + "..."
	}

	prompt := `Analiza el siguiente contenido de memoria/conocimiento que un usuario ha subido. 
En español, proporciona un breve feedback que incluya:
1. Patrones o temas principales que detectas
2. Tipo de contenido (técnico, creativo, procedimental, etc.)
3. Una sugerencia de cómo Ozy (el asistente) podría usar esta información para ayudar mejor al usuario

Responde en 2-3 párrafos concisos, en español. No uses markdown.

Contenido del usuario:
---
` + preview

	chunkCh, err := prov.StreamCompletion(context.Background(), []providers.Message{
		{Role: "user", Content: prompt},
	}, providers.CompletionOptions{Stream: false})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM call failed: " + err.Error()})
		return
	}

	var feedback string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			feedback += chunk.Content
		}
	}
	if feedback == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM returned empty feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"feedback": feedback})
}
