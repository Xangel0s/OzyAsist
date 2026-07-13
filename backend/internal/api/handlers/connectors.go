package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
)

func ListConnectors(c *gin.Context) {
	userID := db.DefaultUserID()
	connList, err := db.ListConnectors(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if connList == nil {
		connList = []models.Connector{}
	}
	c.JSON(http.StatusOK, connList)
}

func CreateConnector(c *gin.Context) {
	var req struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		Endpoint   string `json:"endpoint"`
		AuthConfig string `json:"authConfig"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	connector := &models.Connector{
		ID:             uuid.NewString(),
		UserID:         db.DefaultUserID(),
		Name:           req.Name,
		Type:           req.Type,
		Endpoint:       req.Endpoint,
		AuthConfigJSON: req.AuthConfig,
		CreatedAt:      time.Now(),
	}

	if connector.Type == "" {
		connector.Type = "custom"
	}

	if err := db.CreateConnector(connector); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, connector)
}

func DeleteConnector(c *gin.Context) {
	id := c.Param("id")
	if err := db.DeleteConnector(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
