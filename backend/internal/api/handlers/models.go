package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ListModels(c *gin.Context) {
	c.JSON(http.StatusOK, []gin.H{
		{"provider": "opencode", "models": []gin.H{
			{"id": "deepseek-v4-pro", "name": "DeepSeek V4 Pro", "tier": "go"},
			{"id": "deepseek-v4-flash", "name": "DeepSeek V4 Flash", "tier": "go"},
		}},
		{"provider": "anthropic", "models": []gin.H{{"id": "claude-sonnet-5", "name": "Claude Sonnet 5"}, {"id": "claude-haiku-5", "name": "Claude Haiku 5"}}},
		{"provider": "openai", "models": []gin.H{{"id": "gpt-4o", "name": "GPT-4o"}, {"id": "gpt-4o-mini", "name": "GPT-4o Mini"}}},
		{"provider": "openrouter", "models": []gin.H{{"id": "openrouter/auto", "name": "OpenRouter Auto"}}},
		{"provider": "lmstudio", "models": []gin.H{{"id": "local-model", "name": "Local (LM Studio)"}}},
	})
}

func SelectModel(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented yet"})
}
