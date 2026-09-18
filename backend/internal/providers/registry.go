package providers

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	keys      map[string]string
}

var global = &Registry{
	providers: make(map[string]Provider),
	keys:      make(map[string]string),
}

func Register(name string, p Provider) {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.providers[name] = p
	log.Printf("Provider registrado: %s", name)
}

func Unregister(name string) {
	global.mu.Lock()
	defer global.mu.Unlock()
	delete(global.providers, name)
	delete(global.keys, name)
	log.Printf("Provider removido: %s", name)
}

func Get(name string) (Provider, error) {
	global.mu.RLock()
	defer global.mu.RUnlock()
	p, ok := global.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider no encontrado: %s", name)
	}
	return p, nil
}

// GetDefaultOrFirstProvider busca el primer proveedor configurado con clave válida
func GetDefaultOrFirstProvider() Provider {
	priorityOrder := []string{"cohere", "groq", "openai", "openrouter", "anthropic", "deepseek", "opencode", "lmstudio", "ollama"}
	for _, name := range priorityOrder {
		if p, err := Get(name); err == nil && GetProviderKey(name) != "" {
			return p
		}
	}
	for _, name := range priorityOrder {
		if p, err := Get(name); err == nil {
			return p
		}
	}
	return nil
}

func Available() []string {
	global.mu.RLock()
	defer global.mu.RUnlock()
	names := make([]string, 0, len(global.providers))
	for n := range global.providers {
		names = append(names, n)
	}
	return names
}

type ProviderInfo struct {
	Name   string   `json:"provider"`
	Models []string `json:"models"`
}

func ListAvailable() []ProviderInfo {
	global.mu.RLock()
	defer global.mu.RUnlock()
	result := make([]ProviderInfo, 0, len(global.providers))
	for name, p := range global.providers {
		result = append(result, ProviderInfo{
			Name:   name,
			Models: p.Models(),
		})
	}
	return result
}

func RegisterProviderKey(name, key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		Unregister(name)
		return
	}

	global.mu.Lock()
	global.keys[name] = key
	global.mu.Unlock()

	// Check if key is an OpenCode key (e.g. sk-GsH...)
	isOpenCodeKey := strings.HasPrefix(key, "sk-GsH") || strings.Contains(key, "opencode")

	switch name {
	case "opencode":
		Register("opencode", NewOpenCode(key))
	case "deepseek":
		Register("deepseek", NewOpenCode(key))
	case "openai":
		if isOpenCodeKey {
			Register("opencode", NewOpenCode(key))
		} else {
			Register("openai", NewOpenAI(key))
		}
	case "openrouter":
		Register("openrouter", NewOpenRouter(key))
	case "groq":
		Register("groq", NewGroq(key))
	case "cohere":
		Register("cohere", NewCohere(key))
	case "anthropic":
		Register("anthropic", NewAnthropic(key))
	case "mistral":
		Register("mistral", NewMistral(key))
	case "kilocode", "kilo":
		Register("kilocode", NewKiloCode(key))
	}
}

func GetProviderKey(name string) string {
	global.mu.RLock()
	key, ok := global.keys[name]
	global.mu.RUnlock()
	if ok && strings.TrimSpace(key) != "" {
		return strings.TrimSpace(key)
	}

	switch name {
	case "groq":
		return strings.TrimSpace(os.Getenv("GROQ_API_KEY"))
	case "cohere":
		return strings.TrimSpace(os.Getenv("COHERE_API_KEY"))
	case "openrouter":
		return strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	case "openai":
		return strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	case "anthropic":
		return strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	case "deepseek":
		if k := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")); k != "" {
			return k
		}
		return strings.TrimSpace(os.Getenv("OPENCODE_API_KEY"))
	case "opencode":
		if k := strings.TrimSpace(os.Getenv("OPENCODE_API_KEY")); k != "" {
			return k
		}
		return strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	case "mistral":
		return strings.TrimSpace(os.Getenv("MISTRAL_API_KEY"))
	case "kilocode", "kilo":
		return strings.TrimSpace(os.Getenv("KILOCODE_API_KEY"))
	}
	return ""
}

var (
	localHostURL   string
	localHostURLMu sync.RWMutex
)

// RegisterLocalHostURL registers or updates a local host endpoint (LM Studio, Ollama, vLLM, LocalAI)
func RegisterLocalHostURL(url string) {
	localHostURLMu.Lock()
	defer localHostURLMu.Unlock()
	url = strings.TrimSpace(url)
	localHostURL = url
	if url == "" {
		Unregister("lmstudio")
		Unregister("ollama")
		return
	}
	normalizedURL := strings.TrimRight(url, "/")
	if !strings.HasSuffix(normalizedURL, "/v1") {
		normalizedURL = normalizedURL + "/v1"
	}
	Register("lmstudio", NewLMStudio(normalizedURL))
	Register("ollama", NewOllama(normalizedURL))
}

// GetLocalHostURL returns the active local host endpoint (defaulting to env or http://localhost:11434)
func GetLocalHostURL() string {
	localHostURLMu.RLock()
	defer localHostURLMu.RUnlock()
	if localHostURL != "" {
		return localHostURL
	}
	if url := strings.TrimSpace(os.Getenv("LMSTUDIO_URL")); url != "" {
		return url
	}
	if url := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")); url != "" {
		return url
	}
	return "http://localhost:11434"
}

func InitProviders() {
	LoadEnvFiles()

	if key := os.Getenv("COHERE_API_KEY"); key != "" {
		RegisterProviderKey("cohere", key)
	}
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		RegisterProviderKey("groq", key)
	}
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		RegisterProviderKey("openrouter", key)
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		RegisterProviderKey("openai", key)
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		RegisterProviderKey("anthropic", key)
	}
	if key := os.Getenv("DEEPSEEK_API_KEY"); key != "" {
		RegisterProviderKey("deepseek", key)
	}
	if key := os.Getenv("OPENCODE_API_KEY"); key != "" {
		RegisterProviderKey("opencode", key)
	}
	if key := os.Getenv("MISTRAL_API_KEY"); key != "" {
		RegisterProviderKey("mistral", key)
	}
	if key := os.Getenv("KILOCODE_API_KEY"); key != "" {
		RegisterProviderKey("kilocode", key)
	}

	// Registrar host local (LM Studio / Ollama)
	RegisterLocalHostURL(GetLocalHostURL())

	log.Printf("Providers disponibles iniciales: %v", Available())
}

// ProviderCatalogItem describe un proveedor soportado en el sistema para selectores de interfaz
type ProviderCatalogItem struct {
	ID           string   `json:"id"`
	DisplayName  string   `json:"display_name"`
	DefaultModel string   `json:"default_model"`
	Description  string   `json:"description"`
	IsLocal      bool     `json:"is_local"`
}

// GetSupportedCatalog devuelve la lista oficial de proveedores soportados en OzyAssist
func GetSupportedCatalog() []ProviderCatalogItem {
	return []ProviderCatalogItem{
		{
			ID:           "cohere",
			DisplayName:  "Cohere",
			DefaultModel: "command-r-plus-08-2024",
			Description:  "Command R+ / R7B - Multilingüe y agentic reasoning",
			IsLocal:      false,
		},
		{
			ID:           "groq",
			DisplayName:  "Groq",
			DefaultModel: "llama-3.3-70b-versatile",
			Description:  "Llama 3.3 70B - Ultra rápida inferencia (~800 tok/s)",
			IsLocal:      false,
		},
		{
			ID:           "mistral",
			DisplayName:  "Mistral AI",
			DefaultModel: "mistral-large-latest",
			Description:  "Mistral Large / Codestral - Código y razonamiento",
			IsLocal:      false,
		},
		{
			ID:           "openai",
			DisplayName:  "OpenAI",
			DefaultModel: "gpt-4o",
			Description:  "GPT-4o / GPT-4o-mini - Razonamiento multimodal",
			IsLocal:      false,
		},
		{
			ID:           "deepseek",
			DisplayName:  "DeepSeek / OpenCode",
			DefaultModel: "deepseek-chat",
			Description:  "DeepSeek-V3 / R1 - Especializado en programación",
			IsLocal:      false,
		},
		{
			ID:           "kilocode",
			DisplayName:  "KiloCode Gateway",
			DefaultModel: "anthropic/claude-3.7-sonnet",
			Description:  "Pasarela universal +500 modelos (Claude, GPT, Gemini)",
			IsLocal:      false,
		},
		{
			ID:           "lmstudio",
			DisplayName:  "LM Studio (Local)",
			DefaultModel: "local-model",
			Description:  "Servidor local http://localhost:1234/v1 (Sin API Key)",
			IsLocal:      true,
		},
		{
			ID:           "ollama",
			DisplayName:  "Ollama (Local)",
			DefaultModel: "llama3",
			Description:  "Servidor local http://localhost:11434/v1 (Sin API Key)",
			IsLocal:      true,
		},
	}
}

// CreateProvider crea una instancia del proveedor dado su identificador y clave (puede ser vacía)
func CreateProvider(name, key string) Provider {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "cohere":
		return NewCohere(key)
	case "groq":
		return NewGroq(key)
	case "mistral":
		return NewMistral(key)
	case "openai":
		if strings.HasPrefix(key, "sk-GsH") || strings.Contains(key, "opencode") {
			return NewOpenCode(key)
		}
		return NewOpenAI(key)
	case "deepseek", "opencode":
		return NewOpenCode(key)
	case "kilocode", "kilo":
		return NewKiloCode(key)
	case "openrouter":
		return NewOpenRouter(key)
	case "anthropic":
		return NewAnthropic(key)
	case "lmstudio":
		url := GetLocalHostURL()
		if !strings.HasSuffix(url, "/v1") {
			url = strings.TrimRight(url, "/") + "/v1"
		}
		return NewLMStudio(url)
	case "ollama":
		url := GetLocalHostURL()
		if !strings.HasSuffix(url, "/v1") {
			url = strings.TrimRight(url, "/") + "/v1"
		}
		return NewOllama(url)
	default:
		return nil
	}
}

// EnsureProvider obtiene el proveedor registrado o crea una instancia segura con la clave guardada o vacía.
func EnsureProvider(name string) Provider {
	name = strings.ToLower(strings.TrimSpace(name))
	if p, err := Get(name); err == nil && p != nil {
		return p
	}
	key := GetProviderKey(name)
	p := CreateProvider(name, key)
	if p != nil {
		Register(name, p)
	}
	return p
}

