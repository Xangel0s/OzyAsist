package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ozyassist/backend/internal/providers"
)

type PlanStep struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	ActionType  string `json:"actionType"` // file_read, file_write, command_exec, browser_action, llm_reason
	Target      string `json:"target"`
	Input       string `json:"input,omitempty"`
	DependsOn   []int  `json:"dependsOn,omitempty"` // IDs de pasos que deben completarse antes
}

type Plan struct {
	Goal  string     `json:"goal"`
	Steps []PlanStep `json:"steps"`
}

// Planner descompone un goal en pasos ejecutables usando el LLM.
type Planner struct {
	provider providers.Provider
}

func NewPlanner(provider providers.Provider) *Planner {
	return &Planner{provider: provider}
}

func (p *Planner) CreatePlan(ctx context.Context, goal string, contextInfo string) (*Plan, error) {
	systemPrompt := `Eres un planificador de tareas de IA. Descompones un objetivo en pasos ejecutables.

Responde ÚNICAMENTE con un JSON válido con esta estructura:
{
  "steps": [
    {
      "id": 1,
      "description": "Descripción clara del paso",
      "actionType": "file_read|file_write|command_exec|browser_action|llm_reason",
      "target": "ruta del archivo o comando",
      "input": "contenido a escribir o parámetros adicionales (opcional)",
      "dependsOn": []
    }
  ]
}

Reglas:
- "file_read": para leer archivos existentes
- "file_write": para crear o modificar archivos (incluir contenido en "input")
- "command_exec": para ejecutar comandos del sistema
- "llm_reason": para pasos que requieren razonamiento sin ejecutar herramientas
- Los pasos deben ser atómicos y ejecutables secuencialmente
- Si un paso depende de otro, incluir su ID en "dependsOn"
- Máximo 10 pasos por plan`

	history := []providers.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Contexto:\n%s\n\nObjetivo: %s", contextInfo, goal)},
	}

	var fullContent string
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			d := time.Duration(2<<attempt) * time.Second
			time.Sleep(d)
		}
		content, err := p.callLLM(ctx, history)
		if err == nil {
			fullContent = content
			break
		}
		lastErr = err
	}
	if fullContent == "" {
		return nil, fmt.Errorf("planner LLM call (3 attempts): %w", lastErr)
	}

	// Extract JSON from response (handle markdown code fences)
	jsonStr := fullContent
	if start := indexOf(jsonStr, "{"); start >= 0 {
		jsonStr = jsonStr[start:]
	}
	if end := lastIndexOf(jsonStr, "}"); end >= 0 {
		jsonStr = jsonStr[:end+1]
	}

	var plan struct {
		Steps []PlanStep `json:"steps"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("parse plan JSON: %w\nResponse: %s", err, fullContent)
	}

	for i := range plan.Steps {
		plan.Steps[i].ID = i + 1
	}

	return &Plan{Goal: goal, Steps: plan.Steps}, nil
}

func (p *Planner) callLLM(ctx context.Context, history []providers.Message) (string, error) {
	chunkCh, err := p.provider.StreamCompletion(ctx, history, providers.CompletionOptions{Stream: false})
	if err != nil {
		return "", err
	}

	var fullContent string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			fullContent += chunk.Content
		}
	}

	if fullContent == "" {
		return "", fmt.Errorf("planner returned empty response")
	}
	return fullContent, nil
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func lastIndexOf(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
