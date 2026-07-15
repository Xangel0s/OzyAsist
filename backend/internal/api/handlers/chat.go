package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

func ListChats(c *gin.Context) {
	chats, err := db.ListChats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, chats)
}

func CreateChat(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		ProjectID string `json:"project_id"`
		Mode     string `json:"mode"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
		return
	}

	mode := req.Mode
	if mode != "chat" && mode != "code" {
		mode = "chat"
	}
	provider := req.Provider
	model := req.Model

	// Validate provider exists, fallback to first available
	if _, err := providers.Get(provider); err != nil {
		available := providers.Available()
		if len(available) > 0 {
			provider = available[0]
			if p, err := providers.Get(provider); err == nil && model == "" {
				if len(p.Models()) > 0 {
					model = p.Models()[0]
				}
			}
		} else {
			provider = "openrouter"
		}
	}
	if model == "" {
		if p, err := providers.Get(provider); err == nil && len(p.Models()) > 0 {
			model = p.Models()[0]
		} else {
			model = "openrouter/auto"
		}
	}

	chat := &models.Chat{
		ID:        uuid.NewString(),
		UserID:    db.DefaultUserID(),
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Mode:      mode,
		Provider:  provider,
		Model:     model,
		CreatedAt: time.Now(),
	}

	if err := db.CreateChat(chat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, chat)
}

func GetChat(c *gin.Context) {
	chat, err := db.GetChat(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chat no encontrado"})
		return
	}

	msgs, err := db.GetMessages(chat.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chat":     chat,
		"messages": msgs,
	})
}

func DeleteChat(c *gin.Context) {
	if err := db.DeleteChat(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "eliminado"})
}

func UpdateChat(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
		return
	}

	chat, err := db.GetChat(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chat no encontrado"})
		return
	}

	if req.Name != "" {
		chat.Name = req.Name
	}
	if req.Provider != "" {
		chat.Provider = req.Provider
	}
	if req.Model != "" {
		chat.Model = req.Model
	}

	if err := db.UpdateChat(chat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chat)
}

func SendMessage(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "usa WebSocket /ws para streaming"})
}
