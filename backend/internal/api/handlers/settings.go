package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/providers"
)

func GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"providers": providers.ListAvailable(),
	})
}

func UpdateSettings(c *gin.Context) {
	var req struct {
		OpencodeKey  string `json:"opencode_key"`
		OpenAIKey    string `json:"openai_key"`
		OpenRouterKey string `json:"openrouter_key"`
		AnthropicKey string `json:"anthropic_key"`
		DeepseekKey  string `json:"deepseek_key"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
		return
	}

	if req.OpencodeKey != "" {
		providers.RegisterProviderKey("opencode", req.OpencodeKey)
	}
	if req.DeepseekKey != "" {
		providers.RegisterProviderKey("opencode", req.DeepseekKey)
	}
	if req.OpenAIKey != "" {
		providers.RegisterProviderKey("openai", req.OpenAIKey)
	}
	if req.OpenRouterKey != "" {
		providers.RegisterProviderKey("openrouter", req.OpenRouterKey)
	}
	if req.AnthropicKey != "" {
		providers.RegisterProviderKey("anthropic", req.AnthropicKey)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "actualizado",
		"providers": providers.ListAvailable(),
	})
}
