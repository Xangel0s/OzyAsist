package providers

type GroqProvider struct {
	*OpenAIProvider
}

func NewGroq(apiKey string) *GroqProvider {
	p := NewOpenAI(apiKey)
	p.cfg.providerName = "groq"
	p.cfg.baseURL = "https://api.groq.com/openai/v1"
	p.cfg.model = "llama-3.3-70b-versatile"
	return &GroqProvider{p}
}

func (p *GroqProvider) Name() string { return "groq" }
func (p *GroqProvider) Models() []string {
	return []string{
		"llama-3.3-70b-versatile",
		"llama-3.1-8b-instant",
		"mixtral-8x7b-32768",
		"gemma2-9b-it",
	}
}
