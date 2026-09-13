package agent

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/taskengine"
)

type Healer struct {
	provider providers.Provider
	verifier *Verifier
}

func NewHealer(provider providers.Provider, verifier *Verifier) *Healer {
	return &Healer{
		provider: provider,
		verifier: verifier,
	}
}

type HealAttemptResult = taskengine.HealResult

// AttemptRecovery analiza el error de un paso y genera un parche o comando correctivo
func (h *Healer) AttemptRecovery(
	ctx context.Context,
	task models.AgentTask,
	step *models.TaskStep,
	stepError string,
	output string,
) (*taskengine.HealResult, error) {
	if step.RetryCount >= step.MaxRetries {
		return &taskengine.HealResult{
			Success:       false,
			FailureReason: fmt.Sprintf("límite máximo de reintentos (%d) alcanzado", step.MaxRetries),
		}, nil
	}

	step.RetryCount++
	log.Printf("[self-heal] Intentando auto-reparación paso %s (Intento %d/%d) para tarea %s",
		step.ID, step.RetryCount, step.MaxRetries, task.ID)

	prompt := fmt.Sprintf(`Eres el módulo de Auto-Reparación (Self-Healing) de OzyAssist.
La siguiente subtarea ha fallado durante su ejecución o verificación heurística.

=== CONTEXTO DE LA TAREA ===
Título: %s
Prompt Global: %s

=== SUBTAREA FALLIDA ===
Orden: %d
Tipo de Acción: %s
Payload ejecutado: %s
Error detectado: %s
Salida capturada:
%s

=== INSTRUCCIONES ===
1. Analiza el traceback o causa raíz del fallo.
2. Devuelve una solución concreta y concisa.
3. Si la acción fue 'shell_exec', devuelve el comando corregido.
4. Si fue 'file_patch', devuelve el parche diff en formato unificado.
Formato de respuesta: Devuelve únicamente un bloque con la acción corregida.`,
		task.Title, task.Prompt, step.StepOrder, step.ActionType, step.Payload, stepError, output)

	if h.provider == nil {
		return &taskengine.HealResult{
			Success:       false,
			FailureReason: "no hay LLM provider disponible para generar parche de auto-reparación",
		}, nil
	}

	messages := []providers.Message{
		{Role: "user", Content: prompt},
	}

	chunkCh, err := h.provider.StreamCompletion(ctx, messages, providers.CompletionOptions{
		Temperature: 0.1,
		MaxTokens:   1500,
		Stream:      false,
	})
	if err != nil {
		return nil, fmt.Errorf("fallo al consultar LLM para self-healing: %w", err)
	}

	var fullContent string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			fullContent += chunk.Content
		}
	}

	patch := strings.TrimSpace(fullContent)
	if patch == "" {
		return &taskengine.HealResult{
			Success:       false,
			FailureReason: "el modelo devolvió una propuesta de corrección vacía",
		}, nil
	}

	return &taskengine.HealResult{
		Success:      true,
		PatchApplied: patch,
		NewOutput:    "Parche de corrección generado por auto-reparación",
	}, nil
}
