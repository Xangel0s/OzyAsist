package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/providers"
)

func ListModels(c *gin.Context) {
	available := providers.ListAvailable()
	c.JSON(http.StatusOK, available)
}

func SelectModel(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
}
