package providers

type OpenRouterProvider struct {
	*OpenAIProvider
}

func NewOpenRouter(apiKey string) *OpenRouterProvider {
	p := NewOpenAI(apiKey)
	p.cfg.baseURL = "https://openrouter.ai/api/v1"
	p.cfg.model = "openrouter/auto"
	return &OpenRouterProvider{p}
}

func (p *OpenRouterProvider) Name() string    { return "openrouter" }
func (p *OpenRouterProvider) Models() []string {
	return []string{
		"openrouter/auto",
		"qwen/qwen-2.5-coder-32b-instruct",
		"deepseek/deepseek-chat",
		"google/gemini-2.0-flash-001",
	}
}
