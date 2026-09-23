package providers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type LlamaCppProvider struct {
	*OpenAIProvider
	baseURL string
	models  []string
}

// NewLlamaCpp initializes a native llama-server provider with dynamic model detection
func NewLlamaCpp(baseURL string) *LlamaCppProvider {
	p := NewOpenAI("not-needed")

	// Ensure trailing /v1
	normalized := strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(normalized, "/v1") {
		normalized = normalized + "/v1"
	}
	p.cfg.baseURL = normalized
	p.cfg.model = "OzyAssist-7B"

	lp := &LlamaCppProvider{
		OpenAIProvider: p,
		baseURL:        normalized,
	}

	lp.refreshModels()
	if len(lp.models) > 0 {
		p.cfg.model = lp.models[0]
	}

	return lp
}

func (p *LlamaCppProvider) Name() string       { return "llamacpp" }
func (p *LlamaCppProvider) SupportsTools() bool { return true }

func (p *LlamaCppProvider) Models() []string {
	p.refreshModels()
	if len(p.models) == 0 {
		return []string{"OzyAssist-7B"}
	}
	return p.models
}

func (p *LlamaCppProvider) SetModel(modelName string) {
	p.cfg.model = modelName
}

func (p *LlamaCppProvider) refreshModels() {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(p.baseURL + "/models")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && len(result.Data) > 0 {
		var names []string
		for _, m := range result.Data {
			if m.ID != "" {
				names = append(names, m.ID)
			}
		}
		if len(names) > 0 {
			p.models = names
		}
	}
}
