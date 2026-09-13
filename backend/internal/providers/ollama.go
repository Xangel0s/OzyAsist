package providers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type OllamaProvider struct {
	*OpenAIProvider
	baseURL string
	models  []string
}

// NewOllama initializes a true Ollama provider with dynamic model fetching
func NewOllama(baseURL string) *OllamaProvider {
	p := NewOpenAI("not-needed")
	
	// OpenAI compatibility layer in Ollama
	p.cfg.baseURL = baseURL

	op := &OllamaProvider{
		OpenAIProvider: p,
		baseURL:        strings.TrimSuffix(strings.TrimSuffix(baseURL, "/v1"), "/"),
	}

	// Fetch models dynamically from Ollama native API
	op.refreshModels()
	
	if len(op.models) > 0 {
		p.cfg.model = op.models[0]
	}

	return op
}

func (p *OllamaProvider) Name() string       { return "ollama" }
func (p *OllamaProvider) SupportsTools() bool { return true }

func (p *OllamaProvider) Models() []string {
	p.refreshModels()
	if len(p.models) == 0 {
		return []string{"ozyassist:7b"} // Fallback default
	}
	return p.models
}

// SetModel dynamically sets the selected model
func (p *OllamaProvider) SetModel(modelName string) {
	p.cfg.model = modelName
}

func (p *OllamaProvider) refreshModels() {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(p.baseURL + "/api/tags")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		var names []string
		for _, m := range result.Models {
			names = append(names, m.Name)
		}
		p.models = names
	}
}
