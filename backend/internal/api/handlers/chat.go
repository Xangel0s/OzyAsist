package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
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
	if provider == "" {
		provider = "openai"
	}
	model := req.Model
	if model == "" {
		model = "gpt-4o"
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

func SendMessage(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "usa WebSocket /ws para streaming"})
}
