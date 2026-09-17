package providers

// KiloCodeProvider implementa el proveedor de KiloCode AI Gateway.
// KiloCode expone un gateway unificado compatible con OpenAI que enruta
// a más de 500 modelos: Anthropic, OpenAI, Google, Mistral, DeepSeek, etc.
// Endpoint: https://api.kilo.ai/api/gateway
// Autenticación: Bearer <JWT-token>
type KiloCodeProvider struct {
	*OpenAIProvider
}

// NewKiloCode crea un proveedor KiloCode con el JWT token provisto.
func NewKiloCode(apiKey string) *KiloCodeProvider {
	p := NewOpenAI(apiKey)
	p.cfg.providerName = "kilocode"
	p.cfg.baseURL = "https://api.kilo.ai/api/gateway"
	p.cfg.model = "kilo/anthropic/claude-sonnet-4-5"
	return &KiloCodeProvider{p}
}

// Name retorna el identificador canónico del proveedor.
func (p *KiloCodeProvider) Name() string { return "kilocode" }

// SupportsTools indica que el gateway de KiloCode soporta tool calling
// (pasa la llamada al modelo subyacente que tenga esa capacidad).
func (p *KiloCodeProvider) SupportsTools() bool { return true }

// Models retorna una selección de modelos disponibles en el gateway de KiloCode.
// Se usa la notación "kilo/<proveedor>/<modelo>" del gateway unificado.
func (p *KiloCodeProvider) Models() []string {
	return []string{
		"kilo/anthropic/claude-sonnet-4-5",
		"kilo/anthropic/claude-opus-4-5",
		"kilo/openai/gpt-4o",
		"kilo/openai/gpt-4o-mini",
		"kilo/openai/gpt-5",
		"kilo/google/gemini-2-5-pro",
		"kilo/deepseek/deepseek-chat",
		"kilo/mistral/mistral-large-latest",
	}
}
