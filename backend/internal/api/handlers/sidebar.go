package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/providers"
)

func Observe(c *gin.Context) {
	var req struct {
		Context   string `json:"context"`   // Text description of what user is seeing
		ChatID    string `json:"chatId"`    // Optional: active chat context
		ProjectID string `json:"projectId"` // Optional: active project context
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Context == "" && req.ChatID == "" && req.ProjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one context source is required"})
		return
	}

	prov, err := providers.Get("opencode")
	if err != nil {
		prov, _ = providers.Get("openai")
	}
	if prov == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no provider available"})
		return
	}

	// Build observation context
	var observation string
	if req.Context != "" {
		observation = "User's current context: " + req.Context + "\n\n"
	}
	if req.ChatID != "" {
		if chat, err := db.GetChat(req.ChatID); err == nil {
			observation += fmt.Sprintf("Active chat: %s (mode: %s)\n", chat.Name, chat.Mode)
			if msgs, err := db.GetMessages(req.ChatID); err == nil && len(msgs) > 0 {
				last := msgs[len(msgs)-1]
				observation += fmt.Sprintf("Last message: [%s] %s\n", last.Role, last.Content)
			}
		}
	}
	if req.ProjectID != "" {
		if project, err := db.GetProject(req.ProjectID); err == nil {
			observation += fmt.Sprintf("Active project: %s\nRoot: %s\n", project.Name, project.RootPath)
		}
	}

	prompt := fmt.Sprintf(`Eres un asistente lateral de IA que observa lo que el usuario está haciendo y ofrece ayuda contextual.

Contexto actual:
%s

Basado en este contexto, provee:
1. Un resumen breve de lo que el usuario está haciendo
2. 2-3 sugerencias de acciones útiles que puede tomar
3. Cualquier información relevante del contexto

Responde en español, máximo 3 párrafos.`, observation)

	chunkCh, err := prov.StreamCompletion(context.Background(), []providers.Message{
		{Role: "user", Content: prompt},
	}, providers.CompletionOptions{Stream: false})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			response += chunk.Content
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis": response,
		"context":  observation,
	})
}

func SidebarCommand(c *gin.Context) {
	var req struct {
		Command   string `json:"command"`
		ChatID    string `json:"chatId"`
		ProjectID string `json:"projectId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Command == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "command is required"})
		return
	}

	prov, err := providers.Get("opencode")
	if err != nil {
		prov, _ = providers.Get("openai")
	}
	if prov == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no provider available"})
		return
	}

	var contextStr string
	if req.ChatID != "" {
		if msgs, err := db.GetMessages(req.ChatID); err == nil && len(msgs) > 0 {
			last := msgs[len(msgs)-1]
			contextStr = fmt.Sprintf("Último mensaje del chat: [%s] %s\n", last.Role, last.Content)
		}
	}

	prompt := fmt.Sprintf(`Contexto: %s
Comando del usuario: %s

Responde al comando del usuario de manera útil y directa. Si es una pregunta sobre el contexto, respóndela. Si es una acción, explica cómo ejecutarla.`, contextStr, req.Command)

	chunkCh, err := prov.StreamCompletion(context.Background(), []providers.Message{
		{Role: "user", Content: prompt},
	}, providers.CompletionOptions{Stream: false})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			response += chunk.Content
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"response": response,
	})
}
