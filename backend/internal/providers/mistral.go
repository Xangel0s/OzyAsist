package providers

// MistralProvider implementa el proveedor de Mistral AI.
// Mistral expone una API 100% compatible con OpenAI (/chat/completions + SSE streaming),
// por lo que reutilizamos OpenAIProvider apuntando al endpoint oficial de Mistral.
type MistralProvider struct {
	*OpenAIProvider
}

// NewMistral crea un proveedor Mistral con la API Key provista.
func NewMistral(apiKey string) *MistralProvider {
	p := NewOpenAI(apiKey)
	p.cfg.providerName = "mistral"
	p.cfg.baseURL = "https://api.mistral.ai/v1"
	p.cfg.model = "mistral-large-latest"
	return &MistralProvider{p}
}

// Name retorna el identificador canónico del proveedor.
func (p *MistralProvider) Name() string { return "mistral" }

// SupportsTools indica que Mistral soporta tool calling nativo (OpenAI-compatible).
func (p *MistralProvider) SupportsTools() bool { return true }

// Models retorna los modelos recomendados de Mistral AI.
func (p *MistralProvider) Models() []string {
	return []string{
		"mistral-large-latest",
		"mistral-small-latest",
		"codestral-latest",
		"open-mixtral-8x22b",
		"open-mistral-nemo",
		"open-codestral-mamba",
	}
}
