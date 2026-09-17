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

	// Registrar host local (LM Studio / Ollama)
	RegisterLocalHostURL(GetLocalHostURL())

	log.Printf("Providers disponibles iniciales: %v", Available())
}

