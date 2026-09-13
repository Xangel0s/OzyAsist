package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/mcp"
)

func ListMCPConnectors(c *gin.Context) {
	userID := db.DefaultUserID()
	connectors, err := db.ListMCPConnectors(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if connectors == nil {
		connectors = []models.MCPConnector{}
	}
	c.JSON(http.StatusOK, connectors)
}

func CreateMCPConnector(c *gin.Context) {
	var req struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Env     []string `json:"env"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	connector := &models.MCPConnector{
		ID:        uuid.NewString(),
		UserID:    db.DefaultUserID(),
		Name:      req.Name,
		Command:   req.Command,
		Status:    "disconnected",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	connector.SetArgs(req.Args)
	connector.SetEnv(req.Env)

	if err := db.CreateMCPConnector(connector); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Try to start it immediately
	config := mcp.ServerConfig{
		ID:      connector.ID,
		Name:    connector.Name,
		Command: connector.Command,
		Args:    connector.GetArgs(),
		Env:     connector.GetEnv(),
	}
	
	if err := mcp.DefaultRegistry.AddServer(context.Background(), config); err != nil {
		db.UpdateMCPConnectorStatus(connector.ID, "error")
		connector.Status = "error"
		// Don't fail the request, just return it as error status
	} else {
		db.UpdateMCPConnectorStatus(connector.ID, "connected")
		connector.Status = "connected"
	}

	c.JSON(http.StatusCreated, connector)
}

func DeleteMCPConnector(c *gin.Context) {
	id := c.Param("id")
	
	// Stop server if running
	mcp.DefaultRegistry.RemoveServer(id)

	if err := db.DeleteMCPConnector(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func ReconnectMCPConnector(c *gin.Context) {
	id := c.Param("id")
	connector, err := db.GetMCPConnector(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Stop old if exists
	mcp.DefaultRegistry.RemoveServer(id)

	config := mcp.ServerConfig{
		ID:      connector.ID,
		Name:    connector.Name,
		Command: connector.Command,
		Args:    connector.GetArgs(),
		Env:     connector.GetEnv(),
	}

	if err := mcp.DefaultRegistry.AddServer(context.Background(), config); err != nil {
		db.UpdateMCPConnectorStatus(id, "error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	db.UpdateMCPConnectorStatus(id, "connected")
	c.JSON(http.StatusOK, gin.H{"status": "connected"})
}
