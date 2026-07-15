package providers

type LMStudioProvider struct {
	*OpenAIProvider
}

func NewLMStudio(baseURL string) *LMStudioProvider {
	p := NewOpenAI("not-needed")
	p.cfg.baseURL = baseURL
	p.cfg.model = "local-model"
	return &LMStudioProvider{p}
}

func (p *LMStudioProvider) Name() string       { return "lmstudio" }
func (p *LMStudioProvider) SupportsTools() bool { return false }
func (p *LMStudioProvider) Models() []string {
	return []string{"local-model"}
}
