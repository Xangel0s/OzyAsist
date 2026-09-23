package providers

import (
	"os"
	"strings"
)

type LMStudioProvider struct {
	*OpenAIProvider
}

func NewLMStudio(baseURL string) *LMStudioProvider {
	p := NewOpenAI("not-needed")
	p.cfg.baseURL = baseURL
	model := strings.TrimSpace(os.Getenv("DEFAULT_MODEL"))
	if model == "" {
		model = "qwen2.5-coder-7b-instruct"
	}
	p.cfg.model = model
	return &LMStudioProvider{p}
}

func (p *LMStudioProvider) Name() string       { return "lmstudio" }
func (p *LMStudioProvider) SupportsTools() bool { return true }
func (p *LMStudioProvider) Models() []string {
	if p.cfg.model != "" && p.cfg.model != "local-model" {
		return []string{p.cfg.model, "local-model"}
	}
	return []string{"qwen2.5-coder-7b-instruct", "local-model"}
}
