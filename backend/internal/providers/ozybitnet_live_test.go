package providers

import (
	"context"
	"testing"
	"time"
)

func TestOzyBitNetStreamCompletion(t *testing.T) {
	p := NewOzyBitNet(GetOzyBitNetURL())

	talkAlive, _ := p.HealthCheck()
	if !talkAlive {
		t.Skip("OzyTalk server (port 8766) not running, skipping live completion test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	messages := []Message{
		{Role: "user", Content: "Di 'OzyAssist conectado con OzyBitNet' en pocas palabras."},
	}
	opts := CompletionOptions{
		Model:       "ozytalk-1.58b",
		MaxTokens:   30,
		Temperature: 0.7,
		Stream:      true,
	}

	stream, err := p.StreamCompletion(ctx, messages, opts)
	if err != nil {
		t.Fatalf("StreamCompletion fallo: %v", err)
	}

	var fullResponse string
	for chunk := range stream {
		if chunk.Type == "text" {
			fullResponse += chunk.Content
		}
	}

	t.Logf("Respuesta generada por OzyBitNet a traves de Go: %s", fullResponse)
	if len(fullResponse) == 0 {
		t.Errorf("expected non-empty response from OzyBitNet")
	}
}
