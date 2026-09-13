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
		OpencodeKey   string `json:"opencode_key"`
		OpenAIKey     string `json:"openai_key"`
		OpenRouterKey string `json:"openrouter_key"`
		AnthropicKey  string `json:"anthropic_key"`
		DeepseekKey   string `json:"deepseek_key"`
		LocalHostURL  string `json:"local_host_url"`
		OllamaURL     string `json:"ollama_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
		return
	}

	providers.RegisterProviderKey("opencode", req.OpencodeKey)
	if req.DeepseekKey != "" {
		providers.RegisterProviderKey("deepseek", req.DeepseekKey)
	}
	providers.RegisterProviderKey("openai", req.OpenAIKey)
	providers.RegisterProviderKey("openrouter", req.OpenRouterKey)
	providers.RegisterProviderKey("anthropic", req.AnthropicKey)

	host := req.LocalHostURL
	if host == "" {
		host = req.OllamaURL
	}
	if host != "" {
		providers.RegisterLocalHostURL(host)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "actualizado",
		"providers": providers.ListAvailable(),
	})
}
