package providers

type OpenCodeProvider struct {
	*OpenAIProvider
}

func NewOpenCode(apiKey string) *OpenCodeProvider {
	p := NewOpenAI(apiKey)
	p.cfg.baseURL = "https://opencode.ai/zen/go/v1"
	p.cfg.model = "deepseek-v4-pro"
	return &OpenCodeProvider{p}
}

func (p *OpenCodeProvider) Name() string    { return "opencode" }
func (p *OpenCodeProvider) Models() []string {
	return []string{
		"deepseek-v4-pro",
		"deepseek-v4-flash",
	}
}
