package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type OzyProviderTestRequest struct {
	Provider string `json:"provider"` // "openrouter", "ollama", "deepseek", "opencode", "anthropic"
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
}

type OzyProviderTestResponse struct {
	Success   bool     `json:"success"`
	LatencyMS int64    `json:"latency_ms"`
	Message   string   `json:"message"`
	Models    []string `json:"models,omitempty"`
}

type ProviderHandler struct{}

func NewProviderHandler() *ProviderHandler {
	return &ProviderHandler{}
}

func (h *ProviderHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req OzyProviderTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	resp := ExecuteProviderTest(r.Context(), req)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func TestProviderConnection(c *gin.Context) {
	var req OzyProviderTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	resp := ExecuteProviderTest(c.Request.Context(), req)
	c.JSON(http.StatusOK, resp)
}

func ExecuteProviderTest(ctx context.Context, req OzyProviderTestRequest) OzyProviderTestResponse {
	start := time.Now()
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var success bool
	var msg string
	var availableModels []string

	client := &http.Client{Timeout: 5 * time.Second}

	switch strings.ToLower(req.Provider) {
	case "ollama":
		url := req.BaseURL
		if url == "" {
			url = "http://localhost:11434"
		}
		httpReq, _ := http.NewRequestWithContext(testCtx, "GET", url+"/api/tags", nil)
		resp, err := client.Do(httpReq)
		if err != nil {
			msg = fmt.Sprintf("Ollama local no responde en %s: %v", url, err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var tags struct {
					Models []struct {
						Name string `json:"name"`
					} `json:"models"`
				}
				_ = json.NewDecoder(resp.Body).Decode(&tags)
				for _, m := range tags.Models {
					availableModels = append(availableModels, m.Name)
				}
				success = true
				msg = fmt.Sprintf("Conectado con Ollama local en ZBook (%d modelos detectados)", len(availableModels))
			} else {
				msg = fmt.Sprintf("Ollama devolvió código HTTP %d", resp.StatusCode)
			}
		}

	case "openrouter":
		httpReq, _ := http.NewRequestWithContext(testCtx, "GET", "https://openrouter.ai/api/v1/auth/key", nil)
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
		resp, err := client.Do(httpReq)
		if err != nil {
			msg = fmt.Sprintf("Error de red conectando con OpenRouter: %v", err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				success = true
				msg = "Autenticación correcta con OpenRouter API (Créditos activos)"
			} else {
				msg = fmt.Sprintf("OpenRouter rechazó la API Key (HTTP %d)", resp.StatusCode)
			}
		}

	case "deepseek":
		httpReq, _ := http.NewRequestWithContext(testCtx, "GET", "https://api.deepseek.com/models", nil)
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
		resp, err := client.Do(httpReq)
		if err != nil {
			msg = fmt.Sprintf("Error de conexión con DeepSeek: %v", err)
		} else {
			defer resp.Body.Close()
			success = resp.StatusCode == http.StatusOK
			if success {
				msg = "Conexión exitosa con DeepSeek API"
			} else {
				msg = fmt.Sprintf("DeepSeek devolvió HTTP %d (Verifica tu clave)", resp.StatusCode)
			}
		}

	default:
		msg = "Proveedor no soportado para test en vivo"
	}

	latency := time.Since(start).Milliseconds()

	return OzyProviderTestResponse{
		Success:   success,
		LatencyMS: latency,
		Message:   msg,
		Models:    availableModels,
	}
}
