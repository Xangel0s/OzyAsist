package providers

type OpenRouterProvider struct {
	*OpenAIProvider
}

func NewOpenRouter(apiKey string) *OpenRouterProvider {
	p := NewOpenAI(apiKey)
	p.cfg.providerName = "openrouter"
	p.cfg.baseURL = "https://openrouter.ai/api/v1"
	p.cfg.model = "deepseek/deepseek-chat"
	return &OpenRouterProvider{p}
}

func (p *OpenRouterProvider) Name() string    { return "openrouter" }
func (p *OpenRouterProvider) Models() []string {
	return []string{
		"deepseek/deepseek-chat",
		"deepseek/deepseek-r1",
		"meta-llama/llama-3.3-70b-instruct",
		"anthropic/claude-3.5-sonnet",
		"qwen/qwen-2.5-coder-32b-instruct",
		"mistralai/codestral-2501",
	}
}
