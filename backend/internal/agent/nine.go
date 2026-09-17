package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

type ReplanAction struct {
	ActionType       string  `json:"action_type"`
	Payload          string  `json:"payload"`
	VerificationRule *string `json:"verification_rule,omitempty"`
	Reason           string  `json:"reason"`
}

type StrategicDiagnosis struct {
	RootCause       string         `json:"root_cause"`
	AlternativePlan []ReplanAction `json:"alternative_plan"`
	StrategicAdvice string         `json:"strategic_advice"`
	CanProceed      bool           `json:"can_proceed"`
}

type NineStrategist struct {
	reasoningProvider providers.Provider
}

func NewNineStrategist(reasoningProvider providers.Provider) *NineStrategist {
	return &NineStrategist{
		reasoningProvider: reasoningProvider,
	}
}

// FormulateAlternativeStrategy analiza el bloqueo y sintetiza una ruta de resolución
func (n *NineStrategist) FormulateAlternativeStrategy(
	ctx context.Context,
	task models.AgentTask,
	failedStep *models.TaskStep,
	blockageReason string,
	recentLogs string,
) (*StrategicDiagnosis, error) {
	if n.reasoningProvider == nil {
		return nil, fmt.Errorf("proveedor de razonamiento para Nine no configurado")
	}

	prompt := fmt.Sprintf(`Eres NINE, el Estratega de Razonamiento Profundo de OzyAssist.
El ejecutor OZY ha entrado en un bloqueo o bucle repetitivo y el auditor CHARC ha detenido la ejecución.

=== TAREA GLOBAL ===
ID: %s
Título: %s
Prompt Original: %s

=== SUBTAREA BLOQUEADA ===
Paso: %d
Acción: %s
Payload: %s
Motivo del Bloqueo: %s

=== ÚLTIMA SALIDA / REGISTROS ===
%s

=== INSTRUCCIONES ===
1. Deduce la causa raíz arquitectónica o sintáctica del problema.
2. Propón una ruta alternativa concreta descomponiéndola en pasos de recuperación.
3. Devuelve estrictamente un JSON válido con esta estructura:
{
  "root_cause": "explicación profunda del origen del fallo",
  "can_proceed": true,
  "strategic_advice": "consejo clave para OZY",
  "alternative_plan": [
    {
      "action_type": "shell_exec",
      "payload": "comando alternativo o de corrección",
      "reason": "por qué este paso desbloquea el flujo"
    }
  ]
}`, task.ID, task.Title, task.Prompt, failedStep.StepOrder, failedStep.ActionType, failedStep.Payload, blockageReason, recentLogs)

	messages := []providers.Message{
		{Role: "user", Content: prompt},
	}

	chunkCh, err := n.reasoningProvider.StreamCompletion(ctx, messages, providers.CompletionOptions{
		Temperature: 0.6,
		MaxTokens:   3000,
		Stream:      false,
	})
	if err != nil {
		return nil, fmt.Errorf("fallo consultando el motor de razonamiento de Nine: %w", err)
	}

	var fullText string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			fullText += chunk.Content
		}
	}

	var diagnosis StrategicDiagnosis
	if err := parseJSONResponse(fullText, &diagnosis); err != nil {
		return nil, fmt.Errorf("error deserializando estrategia de Nine: %w (Respuesta: %s)", err, fullText)
	}

	return &diagnosis, nil
}

func parseJSONResponse(raw string, dest interface{}) error {
	cleaned := strings.TrimSpace(raw)
	if idx := strings.Index(cleaned, "{"); idx != -1 {
		if endIdx := strings.LastIndex(cleaned, "}"); endIdx != -1 && endIdx > idx {
			cleaned = cleaned[idx : endIdx+1]
		}
	}
	return json.Unmarshal([]byte(cleaned), dest)
}

// FormulateNineIntervention permite a Nine intervenir en el mismo LLM cuando el loop interactivo detecta un bloqueo.
func FormulateNineIntervention(ctx context.Context, provider providers.Provider, userMessage, toolName, toolInput, lastError string) string {
	if provider == nil {
		return "⚠️ [NINE]: Bucle detectado. Cambia de estrategia y busca una ruta alternativa."
	}

	prompt := fmt.Sprintf(`Eres NINE, el Estratega de Razonamiento Superior de OzyAssist.
El ejecutor Ozy ha entrado en un bucle repetitivo intentando la herramienta '%s' con input '%s'.
Error o resultado repetido:
%s

Objetivo original del usuario:
%s

INSTRUCCIONES PARA NINE:
1. Explica en 1 oración la causa raíz por la que Ozy está trabado.
2. Da exactamente 2 pasos alternativos claros que Ozy debe seguir para cumplir el objetivo del usuario sin repetir la misma herramienta fallida.
Sé conciso, directo y técnico.`, toolName, toolInput, lastError, userMessage)

	messages := []providers.Message{
		{Role: "user", Content: prompt},
	}

	chunkCh, err := provider.StreamCompletion(ctx, messages, providers.CompletionOptions{
		Temperature: 0.3,
		MaxTokens:   400,
		Stream:      false,
	})
	if err != nil {
		return "⚠️ [NINE]: Bucle detectado. Abandona este enfoque e intenta resolverlo con un script o comando diferente."
	}

	var advice strings.Builder
	advice.WriteString("🧠 [INTERVENCIÓN ESTRATÉGICA DE NINE]:\n")
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			advice.WriteString(chunk.Content)
		}
	}
	return advice.String()
}
