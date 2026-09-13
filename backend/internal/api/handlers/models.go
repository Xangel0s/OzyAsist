package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/providers"
)

type AvailableModel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"` // "ollama", "openrouter", "deepseek", "anthropic", "openai", "hybrid"
	Group     string `json:"group"`    // "Ollama Local (ZBook)", "OpenRouter Cloud", "Recomendado", etc.
	IsLocal   bool   `json:"is_local"`
	IsDefault bool   `json:"is_default,omitempty"`
}

// GetAvailableModels introspects real configured LLM providers and local Ollama instances.
// It connects live to OpenRouter, Ollama, DeepSeek, Anthropic, and OpenAI to retrieve their real model catalogs.
// If no functional models or API keys are detected, it returns an empty slice [].
func GetAvailableModels(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 4*time.Second)
	defer cancel()

	var models []AvailableModel
	client := &http.Client{Timeout: 3 * time.Second}

	// 1. Detección de Host Local (LM Studio, Ollama, LocalAI, vLLM)
	localHostURL := providers.GetLocalHostURL()
	if localHostURL != "" {
		normalizedURL := strings.TrimRight(localHostURL, "/")
		hostClean := strings.TrimPrefix(strings.TrimPrefix(normalizedURL, "http://"), "https://")

		// 1.A. Introspección OpenAI-compatible / LM Studio (/v1/models o /api/v1/models)
		modelsEndpoints := []string{
			normalizedURL + "/v1/models",
			normalizedURL + "/api/v1/models",
			normalizedURL + "/models",
		}
		foundLMStudio := false

		for _, endpoint := range modelsEndpoints {
			req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
			if err == nil {
				resp, err := client.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					var oaResp struct {
						Data []struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"data"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&oaResp); err == nil && len(oaResp.Data) > 0 {
						for _, m := range oaResp.Data {
							name := m.Name
							if name == "" {
								name = m.ID
							}
							models = append(models, AvailableModel{
								ID:       "lmstudio/" + m.ID,
								Name:     name + " (Local)",
								Provider: "lmstudio",
								Group:    fmt.Sprintf("LM Studio Local (%s)", hostClean),
								IsLocal:  true,
							})
						}
						foundLMStudio = true
					}
					resp.Body.Close()
					if foundLMStudio {
						break
					}
				}
			}
		}

		// 1.B. Introspección Ollama (/api/tags) si no es LM Studio o si tiene Ollama corriendo
		if !foundLMStudio {
			req, err := http.NewRequestWithContext(ctx, "GET", normalizedURL+"/api/tags", nil)
			if err == nil {
				resp, err := client.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					var tags struct {
						Models []struct {
							Name string `json:"name"`
							Size int64  `json:"size"`
						} `json:"models"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&tags); err == nil && len(tags.Models) > 0 {
						for _, m := range tags.Models {
							cleanName := strings.Split(m.Name, ":")[0]
							models = append(models, AvailableModel{
								ID:       "ollama/" + m.Name,
								Name:     cleanName + " (Local)",
								Provider: "ollama",
								Group:    fmt.Sprintf("Ollama Local (%s)", hostClean),
								IsLocal:  true,
							})
						}
					}
					resp.Body.Close()
				}
			}
		}
	}

	// 2. Detección y catálogo real en vivo de OpenRouter API
	openRouterKey := providers.GetProviderKey("openrouter")
	if openRouterKey != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/models", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+openRouterKey)
			req.Header.Set("HTTP-Referer", "http://localhost:1420")
			req.Header.Set("X-Title", "OzyAssist")
			resp, err := client.Do(req)
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					var orResp struct {
						Data []struct {
							ID          string `json:"id"`
							Name        string `json:"name"`
							Description string `json:"description"`
							ContextLen  int    `json:"context_length"`
						} `json:"data"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&orResp); err == nil && len(orResp.Data) > 0 {
						// Prioritize top popular models first
						var topModels []AvailableModel
						var otherModels []AvailableModel

						popularKeywords := []string{"claude-3.7", "claude-3.5", "deepseek-r1", "deepseek-chat", "gpt-4o", "gemini-2.0", "qwen-2.5", "llama-3.3"}

						for _, m := range orResp.Data {
							displayName := m.Name
							if displayName == "" {
								displayName = m.ID
							}
							item := AvailableModel{
								ID:       "openrouter/" + m.ID,
								Name:     displayName,
								Provider: "openrouter",
								Group:    "OpenRouter Cloud",
								IsLocal:  false,
							}

							isPopular := false
							for _, kw := range popularKeywords {
								if strings.Contains(strings.ToLower(m.ID), kw) {
									isPopular = true
									break
								}
							}

							if isPopular {
								topModels = append(topModels, item)
							} else {
								otherModels = append(otherModels, item)
							}
						}

						models = append(models, topModels...)
						// Limit other models to top 30 to avoid overwhelming the frontend
						if len(otherModels) > 30 {
							models = append(models, otherModels[:30]...)
						} else {
							models = append(models, otherModels...)
						}
					}
				}
				resp.Body.Close()
			}
		}
	}

	// 3. Detección y catálogo real de DeepSeek Directo
	deepseekKey := providers.GetProviderKey("deepseek")
	if deepseekKey == "" {
		deepseekKey = providers.GetProviderKey("opencode")
	}
	if deepseekKey != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.deepseek.com/models", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+deepseekKey)
			resp, err := client.Do(req)
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					var dsResp struct {
						Data []struct {
							ID string `json:"id"`
						} `json:"data"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&dsResp); err == nil && len(dsResp.Data) > 0 {
						for _, m := range dsResp.Data {
							displayName := m.ID
							if m.ID == "deepseek-reasoner" {
								displayName = "DeepSeek R1 (Reasoner)"
							} else if m.ID == "deepseek-chat" {
								displayName = "DeepSeek V3 (Chat)"
							}
							models = append(models, AvailableModel{
								ID:       "deepseek/" + m.ID,
								Name:     displayName,
								Provider: "deepseek",
								Group:    "DeepSeek API",
								IsLocal:  false,
							})
						}
					}
				}
				resp.Body.Close()
			}
		}
	}

	// 4. Detección y catálogo real de Anthropic Directo
	anthropicKey := providers.GetProviderKey("anthropic")
	if anthropicKey != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/models", nil)
		if err == nil {
			req.Header.Set("x-api-key", anthropicKey)
			req.Header.Set("anthropic-version", "2023-06-01")
			resp, err := client.Do(req)
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					var antResp struct {
						Data []struct {
							ID          string `json:"id"`
							DisplayName string `json:"display_name"`
						} `json:"data"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&antResp); err == nil && len(antResp.Data) > 0 {
						for _, m := range antResp.Data {
							name := m.DisplayName
							if name == "" {
								name = m.ID
							}
							models = append(models, AvailableModel{
								ID:       "anthropic/" + m.ID,
								Name:     name,
								Provider: "anthropic",
								Group:    "Anthropic Cloud",
								IsLocal:  false,
							})
						}
					}
				}
				resp.Body.Close()
			}
		}
	}

	// 5. Detección y catálogo real de OpenAI Directo
	openaiKey := providers.GetProviderKey("openai")
	if openaiKey != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+openaiKey)
			resp, err := client.Do(req)
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					var oaResp struct {
						Data []struct {
							ID string `json:"id"`
						} `json:"data"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&oaResp); err == nil && len(oaResp.Data) > 0 {
						for _, m := range oaResp.Data {
							// Filter relevant chat models
							if strings.HasPrefix(m.ID, "gpt-") || strings.HasPrefix(m.ID, "o1") || strings.HasPrefix(m.ID, "o3") || strings.HasPrefix(m.ID, "o4") {
								models = append(models, AvailableModel{
									ID:       "openai/" + m.ID,
									Name:     m.ID,
									Provider: "openai",
									Group:    "OpenAI Cloud",
									IsLocal:  false,
								})
							}
						}
					}
				}
				resp.Body.Close()
			}
		}
	}

	// 6. Asignar IsDefault al primer modelo disponible, dando prioridad a ozyassist
	if len(models) > 0 {
		defaultIdx := 0
		for i, m := range models {
			if strings.Contains(strings.ToLower(m.Name), "ozyassist") {
				defaultIdx = i
				break
			}
		}
		models[defaultIdx].IsDefault = true
	} else {
		// Dejar explícitamente vacío [] si no hay proveedores configurados y funcionales
		models = []AvailableModel{}
	}

	c.JSON(http.StatusOK, models)
}

func ListModels(c *gin.Context) {
	GetAvailableModels(c)
}

func SelectModel(c *gin.Context) {
	var req struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cuerpo inválido"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "provider": req.Provider, "model": req.Model})
}
