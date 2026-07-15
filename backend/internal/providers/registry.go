package providers

import (
	"fmt"
	"log"
	"os"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

var global = &Registry{providers: make(map[string]Provider)}

func Register(name string, p Provider) {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.providers[name] = p
	log.Printf("Provider registrado: %s", name)
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

func InitProviders() {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		Register("openai", NewOpenAI(key))
	}
	if key := os.Getenv("OPENCODE_API_KEY"); key != "" {
		Register("opencode", NewOpenCode(key))
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		Register("anthropic", NewAnthropic(key))
	}
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		Register("openrouter", NewOpenRouter(key))
	}
	if url := os.Getenv("LMSTUDIO_URL"); url != "" {
		Register("lmstudio", NewLMStudio(url))
	}
	log.Printf("Providers disponibles: %v", Available())
}
