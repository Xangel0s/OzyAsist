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
		"deepseek/deepseek-chat",
		"deepseek/deepseek-r1",
		"qwen/qwen-2.5-coder-32b-instruct",
		"anthropic/claude-3.5-sonnet",
		"google/gemini-2.0-flash-001",
		"meta-llama/llama-3.3-70b-instruct",
		"mistralai/codestral-2501",
	}
}
