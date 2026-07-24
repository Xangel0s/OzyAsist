package providers

import (
	"context"
	"testing"
)

func TestOpenCodeProviderInitialization(t *testing.T) {
	provider := NewOpenCode("test-key")
	if provider.Name() != "opencode" {
		t.Errorf("Expected provider name 'opencode', got '%s'", provider.Name())
	}

	models := provider.Models()
	if len(models) == 0 {
		t.Errorf("Expected non-empty models for opencode provider")
	}

	foundDeepseek := false
	for _, m := range models {
		if m == "deepseek-v4-pro" {
			foundDeepseek = true
			break
		}
	}
	if !foundDeepseek {
		t.Errorf("Expected deepseek-v4-pro in opencode models, got %v", models)
	}
}

func TestDynamicProviderRegistration(t *testing.T) {
	RegisterProviderKey("opencode", "sk-test-opencode-key")

	p, err := Get("opencode")
	if err != nil {
		t.Fatalf("Failed to retrieve dynamically registered opencode provider: %v", err)
	}

	if p.Name() != "opencode" {
		t.Errorf("Expected name 'opencode', got '%s'", p.Name())
	}
}

func TestStreamCompletionEmptyKeyHandling(t *testing.T) {
	provider := NewOpenCode("")
	ctx := context.Background()
	ch, err := provider.StreamCompletion(ctx, []Message{{Role: "user", Content: "Hola"}}, CompletionOptions{})
	if err == nil {
		for chunk := range ch {
			if chunk.Type == "error" {
				return
			}
		}
	}
}
