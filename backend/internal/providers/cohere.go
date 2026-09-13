package providers

type CohereProvider struct {
	*OpenAIProvider
}

func NewCohere(apiKey string) *CohereProvider {
	p := NewOpenAI(apiKey)
	p.cfg.providerName = "cohere"
	p.cfg.baseURL = "https://api.cohere.com/compatibility/v1"
	p.cfg.model = "command-r-plus-08-2024"
	return &CohereProvider{p}
}

func (p *CohereProvider) Name() string { return "cohere" }
func (p *CohereProvider) Models() []string {
	return []string{
		"command-r-plus-08-2024",
		"command-r-08-2024",
		"command-r7b-12-2024",
		"command-a-03-2025",
		"c4ai-aya-expanse-32b",
	}
}
