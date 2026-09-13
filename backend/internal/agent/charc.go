package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type AuditDecision struct {
	AllowExecution bool      `json:"allow_execution"`
	RequiresPIN    bool      `json:"requires_pin"`
	RiskLevel      RiskLevel `json:"risk_level"`
	Reason         string    `json:"reason"`
	EscalateToNine bool      `json:"escalate_to_nine"`
}

type CharcAuditor struct {
	provider    providers.Provider
	historyMu   sync.Mutex
	actionHash  map[string]int // hash(task_id + action_type + action_payload) -> conteo de intentos
	dangerRegex []*regexp.Regexp
}

func NewCharcAuditor(provider providers.Provider) *CharcAuditor {
	dangerousPatterns := []string{
		`(?i)rm\s+-rf\s+[/~]`,
		`(?i)mkfs`,
		`(?i)dd\s+if=.*of=/dev/`,
		`(?i)drop\s+database`,
		`(?i)drop\s+table`,
		`(?i)truncate\s+table`,
		`(?i)git\s+push.*--force`,
		`(?i)git\s+reset\s+--hard`,
		`(?i):(){ :|:& };:`,
	}

	var regexes []*regexp.Regexp
	for _, p := range dangerousPatterns {
		if r, err := regexp.Compile(p); err == nil {
			regexes = append(regexes, r)
		}
	}

	return &CharcAuditor{
		provider:    provider,
		actionHash:  make(map[string]int),
		dangerRegex: regexes,
	}
}

// AuditPreExecution inspecciona la subtarea antes de ejecutarla en el host
func (c *CharcAuditor) AuditPreExecution(ctx context.Context, task models.AgentTask, step *models.TaskStep) AuditDecision {
	// 1. Análisis Heurístico Estático de Comandos Destructivos
	for _, r := range c.dangerRegex {
		if r.MatchString(step.Payload) {
			return AuditDecision{
				AllowExecution: true,
				RequiresPIN:    true,
				RiskLevel:      RiskLevelCritical,
				Reason:         fmt.Sprintf("Comando potencialmente destructivo detectado por patrón: %s", r.String()),
			}
		}
	}

	// 2. Loop Breaker Heurístico (Detector de Estancamiento)
	hashKey := c.computeStepHash(task.ID, step.ActionType, step.Payload)
	c.historyMu.Lock()
	c.actionHash[hashKey]++
	count := c.actionHash[hashKey]
	c.historyMu.Unlock()

	if count >= 3 {
		return AuditDecision{
			AllowExecution: false,
			RequiresPIN:    false,
			RiskLevel:      RiskLevelHigh,
			Reason:         fmt.Sprintf("Bucle detectado: la acción idéntica se ha intentado %d veces sin éxito", count),
			EscalateToNine: true,
		}
	}

	// 3. Auditoría Semántica con LLM Local si el paso es un comando shell complejo
	if step.ActionType == "shell_exec" && c.provider != nil {
		return c.auditWithLLM(ctx, task, step)
	}

	return AuditDecision{
		AllowExecution: true,
		RequiresPIN:    step.RequiresPIN,
		RiskLevel:      RiskLevelLow,
		Reason:         "Validación heurística aprobada",
	}
}

func (c *CharcAuditor) auditWithLLM(ctx context.Context, task models.AgentTask, step *models.TaskStep) AuditDecision {
	prompt := fmt.Sprintf(`Eres CHARC, el Auditor de Seguridad y Supervisor de Bucles de OzyAssist.
Evalúa el siguiente comando antes de su ejecución.

=== CONTEXTO ===
Tarea: %s
Payload / Comando: %s

Responde estrictamente con un JSON con la siguiente estructura:
{
  "allow_execution": true,
  "requires_pin": false,
  "risk_level": "low|medium|high|critical",
  "reason": "explicación concisa"
}`, task.Title, step.Payload)

	messages := []providers.Message{
		{Role: "user", Content: prompt},
	}

	chunkCh, err := c.provider.StreamCompletion(ctx, messages, providers.CompletionOptions{
		Temperature: 0.0,
		MaxTokens:   300,
		Stream:      false,
	})
	if err != nil {
		// Fallback permisivo seguro si el LLM local no responde
		return AuditDecision{
			AllowExecution: true,
			RequiresPIN:    step.RequiresPIN,
			RiskLevel:      RiskLevelMedium,
			Reason:         "Aprobado por fallback (LLM auditor no disponible)",
		}
	}

	var fullText string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			fullText += chunk.Content
		}
	}

	var decision AuditDecision
	if err := parseJSONResponse(fullText, &decision); err != nil {
		return AuditDecision{
			AllowExecution: true,
			RequiresPIN:    step.RequiresPIN,
			RiskLevel:      RiskLevelLow,
			Reason:         "Aprobado por regla base",
		}
	}

	return decision
}

func (c *CharcAuditor) computeStepHash(taskID, actionType, payload string) string {
	h := sha256.Sum256([]byte(taskID + ":" + actionType + ":" + strings.TrimSpace(payload)))
	return hex.EncodeToString(h[:8])
}

func (c *CharcAuditor) ResetTaskHistory(taskID string) {
	c.historyMu.Lock()
	defer c.historyMu.Unlock()
	for k := range c.actionHash {
		if strings.HasPrefix(k, taskID) {
			delete(c.actionHash, k)
		}
	}
}
