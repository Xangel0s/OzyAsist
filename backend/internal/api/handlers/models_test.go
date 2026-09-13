package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/providers"
)

func TestGetAvailableModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/models/available", GetAvailableModels)

	t.Run("Empty when no providers or ollama active", func(t *testing.T) {
		os.Unsetenv("OPENROUTER_API_KEY")
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("DEEPSEEK_API_KEY")
		os.Unsetenv("OPENCODE_API_KEY")
		providers.RegisterLocalHostURL("http://127.0.0.1:59999") // unreachable dummy port

		req, _ := http.NewRequest(http.MethodGet, "/api/models/available", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.Code)
		}

		var models []AvailableModel
		if err := json.Unmarshal(resp.Body.Bytes(), &models); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(models) != 0 {
			t.Fatalf("expected 0 models when unconfigured/unreachable, got %d", len(models))
		}
	})

	t.Run("Returns models when mock Ollama is active", func(t *testing.T) {
		mockOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/tags" {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"models": []map[string]any{
						{"name": "qwen2.5-coder:7b"},
					},
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer mockOllama.Close()

		providers.RegisterLocalHostURL(mockOllama.URL)
		defer providers.RegisterLocalHostURL("http://127.0.0.1:59999")

		req, _ := http.NewRequest(http.MethodGet, "/api/models/available", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.Code)
		}

		var models []AvailableModel
		if err := json.Unmarshal(resp.Body.Bytes(), &models); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(models) == 0 {
			t.Fatalf("expected models when mock Ollama is active, got 0")
		}

		// First model should be marked as default
		if !models[0].IsDefault || models[0].Provider != "ollama" {
			t.Errorf("expected first model to be default ollama, got %v", models[0])
		}
	})
}
