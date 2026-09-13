package providers

import (
	"context"
	"testing"
	"time"
)

func TestCohereCompletion(t *testing.T) {
	key := GetProviderKey("cohere")
	if key == "" {
		t.Skip("COHERE_API_KEY no configurada, saltando prueba de integración")
	}
	prov := NewCohere(key)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msgs := []Message{
		{Role: "user", Content: "Di solo 'Hola desde Cohere' y nada mas"},
	}

	ch, err := prov.StreamCompletion(ctx, msgs, CompletionOptions{
		Model: "command-r-plus-08-2024",
	})
	if err != nil {
		t.Fatalf("StreamCompletion error: %v", err)
	}

	var fullResp string
	for chunk := range ch {
		if chunk.Type == "text" {
			fullResp += chunk.Content
		}
	}

	t.Logf("Respuesta Cohere: %s", fullResp)
	if fullResp == "" {
		t.Fatalf("Respuesta vacia")
	}
}
