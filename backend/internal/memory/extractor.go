package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ozyassist/backend/internal/providers"
)

type ExtractedFact struct {
	Category   MemoryCategory `json:"category"`
	Content    string         `json:"content"`
	Confidence float64        `json:"confidence"`
}

type FactExtractor struct {
	provider providers.Provider
	store    *Store
}

func NewFactExtractor(provider providers.Provider, store *Store) *FactExtractor {
	return &FactExtractor{
		provider: provider,
		store:    store,
	}
}

// ExtractAndPersistAsync procesa el mensaje en background
func (fe *FactExtractor) ExtractAndPersistAsync(ctx context.Context, userID, userMessage, assistantResponse string) {
	go func() {
		_ = fe.ExtractAndPersist(context.Background(), userID, userMessage, assistantResponse)
	}()
}

// ExtractAndPersist analiza el mensaje sincrónicamente
func (fe *FactExtractor) ExtractAndPersist(ctx context.Context, userID, userMessage, assistantResponse string) error {
	if len(strings.TrimSpace(userMessage)) < 8 {
		return nil
	}

	prompt := fmt.Sprintf(`Eres el Extractor de Memoria Continua de OzyAssist.
Analiza la siguiente interacción y extrae ÚNICAMENTE hechos permanentes, preferencias técnicas, reglas o configuraciones del usuario.
Si el mensaje no contiene hechos persistentes útiles (ej. saludos o preguntas genéricas), devuelve un arreglo vacío [].

Categorías válidas:
- 'preferencia' (ej: "Prefiere TypeScript estricto con interfaces")
- 'stack' (ej: "El backend está desarrollado en Go 1.22 y SQLite WAL")
- 'hardware' (ej: "Trabaja en una laptop ZBook Workstation")
- 'regla' (ej: "Nunca hacer push a main sin tests pasando")
- 'contexto' (ej: "El puerto de desarrollo principal es el 8080")

=== MENSAJE DEL USUARIO ===
%s

=== RESPUESTA DEL ASISTENTE ===
%s

Responde estrictamente con un JSON:
[
  {"category": "stack", "content": "hecho conciso y atómico", "confidence": 0.95}
]`, userMessage, assistantResponse)

	text, err := fe.queryLLM(ctx, prompt, 0.0, 400)
	if err != nil {
		return err
	}

	var facts []ExtractedFact
	cleanText := strings.TrimSpace(text)
	if start := strings.Index(cleanText, "["); start != -1 {
		if end := strings.LastIndex(cleanText, "]"); end != -1 && end > start {
			_ = json.Unmarshal([]byte(cleanText[start:end+1]), &facts)
		}
	}

	for _, fact := range facts {
		if fact.Content != "" && fact.Category != "" {
			_ = fe.store.UpsertFact(ctx, userID, fact.Category, fact.Content, fact.Confidence)
		}
	}

	return nil
}

func (fe *FactExtractor) queryLLM(ctx context.Context, prompt string, temp float64, maxTokens int) (string, error) {
	if fe.provider == nil {
		return "", fmt.Errorf("llm provider no configurado")
	}

	messages := []providers.Message{
		{Role: "user", Content: prompt},
	}

	chunkCh, err := fe.provider.StreamCompletion(ctx, messages, providers.CompletionOptions{
		Temperature: temp,
		MaxTokens:   maxTokens,
		Stream:      false,
	})
	if err != nil {
		return "", err
	}

	var fullText string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			fullText += chunk.Content
		}
	}
	return strings.TrimSpace(fullText), nil
}
