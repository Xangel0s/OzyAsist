package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"providers": gin.H{
			"anthropic": gin.H{"api_key": "", "enabled": false},
			"openai":    gin.H{"api_key": "", "enabled": false},
			"openrouter": gin.H{"api_key": "", "enabled": false},
			"lmstudio":  gin.H{"base_url": "http://localhost:1234", "enabled": false},
		},
		"embeddings": gin.H{
			"provider": "lmstudio",
			"model":    "nomic-embed-text-v1.5",
		},
		"qdrant": gin.H{
			"url": "http://localhost:6333",
		},
	})
}

func UpdateSettings(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
}
