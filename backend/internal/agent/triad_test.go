package agent

import (
	"context"
	"testing"

	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

type mockTriadProvider struct {
	response string
}

func (m *mockTriadProvider) StreamCompletion(ctx context.Context, messages []providers.Message, opts providers.CompletionOptions) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk, 2)
	ch <- providers.StreamChunk{Type: "text", Content: m.response}
	ch <- providers.StreamChunk{Type: "done"}
	close(ch)
	return ch, nil
}

func (m *mockTriadProvider) Name() string           { return "mock-triad" }
func (m *mockTriadProvider) SupportsTools() bool    { return false }
func (m *mockTriadProvider) Models() []string       { return []string{"mock-model"} }

func TestCharc_DangerousPatternDetection(t *testing.T) {
	auditor := NewCharcAuditor(nil)
	task := models.AgentTask{ID: "task_1", Title: "Limpieza"}
	step := &models.TaskStep{
		ID:         "step_1",
		ActionType: "shell_exec",
		Payload:    "rm -rf /",
	}

	decision := auditor.AuditPreExecution(context.Background(), task, step)
	if !decision.RequiresPIN || decision.RiskLevel != RiskLevelCritical {
		t.Fatalf("Esperaba requerimiento de PIN por comando crítico, obtuvo: %+v", decision)
	}
}

func TestCharc_LoopDetectionAndEscalation(t *testing.T) {
	auditor := NewCharcAuditor(nil)
	task := models.AgentTask{ID: "task_loop", Title: "Compilación repetitiva"}
	step := &models.TaskStep{
		ID:         "step_loop_1",
		ActionType: "shell_exec",
		Payload:    "go build ./...",
	}

	// Primer y segundo intento
	_ = auditor.AuditPreExecution(context.Background(), task, step)
	_ = auditor.AuditPreExecution(context.Background(), task, step)

	// Tercer intento idéntico: debe disparar EscalateToNine
	decision := auditor.AuditPreExecution(context.Background(), task, step)
	if !decision.EscalateToNine || decision.AllowExecution {
		t.Fatalf("Esperaba corte de bucle y escalado a Nine en el 3er intento, obtuvo: %+v", decision)
	}
}

func TestNine_FormulateStrategy(t *testing.T) {
	mockResp := `{
		"root_cause": "Falta importación del paquete fmt",
		"can_proceed": true,
		"strategic_advice": "Añadir import faltante antes de compilar",
		"alternative_plan": [
			{"action_type": "shell_exec", "payload": "goimports -w .", "reason": "corrige imports"}
		]
	}`

	strategist := NewNineStrategist(&mockTriadProvider{response: mockResp})
	task := models.AgentTask{ID: "task_2", Title: "Arreglar error"}
	step := &models.TaskStep{ID: "step_2", StepOrder: 1, ActionType: "shell_exec", Payload: "go build"}

	diag, err := strategist.FormulateAlternativeStrategy(context.Background(), task, step, "Error undefined: Println", "undefined: fmt.Println")
	if err != nil {
		t.Fatalf("Fallo en Nine: %v", err)
	}

	if len(diag.AlternativePlan) != 1 || diag.AlternativePlan[0].ActionType != "shell_exec" {
		t.Fatalf("Estrategia generada inválida: %+v", diag)
	}
}

func TestTriad_SuperviseStep(t *testing.T) {
	mockResp := `{
		"root_cause": "Loop detected",
		"can_proceed": true,
		"strategic_advice": "Skip loop",
		"alternative_plan": [
			{"action_type": "shell_exec", "payload": "echo unblocked", "reason": "unblock"}
		]
	}`

	triad := NewCognitiveTriad(nil, &mockTriadProvider{response: mockResp})
	task := models.AgentTask{ID: "task_triad", Title: "Triad loop test"}
	step := &models.TaskStep{ID: "step_triad", StepOrder: 1, ActionType: "shell_exec", Payload: "stuck command"}

	// Trigger 3 executions to cause loop detection
	_ = triad.Charc.AuditPreExecution(context.Background(), task, step)
	_ = triad.Charc.AuditPreExecution(context.Background(), task, step)

	decision, diagnosis, err := triad.SuperviseStep(context.Background(), task, step, "error log")
	if err != nil {
		t.Fatalf("Error supervisando paso: %v", err)
	}

	if !decision.EscalateToNine {
		t.Fatalf("Esperaba EscalateToNine=true")
	}

	if diagnosis == nil || len(diagnosis.AlternativePlan) == 0 {
		t.Fatalf("Esperaba diagnóstico y plan alternativo de Nine")
	}
}
