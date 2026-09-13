package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/taskengine"
)

type AgentRole string

const (
	RoleOzy   AgentRole = "ozy"   // Ejecutor principal
	RoleCharc AgentRole = "charc" // Auditor de seguridad y bucles
	RoleNine  AgentRole = "nine"  // Estratega de razonamiento
)

type TriadMessage struct {
	FromRole AgentRole   `json:"from_role"`
	ToRole   AgentRole   `json:"to_role"`
	TaskID   string      `json:"task_id"`
	StepID   string      `json:"step_id"`
	Type     string      `json:"type"` // 'audit_req', 'audit_resp', 'escalate_nine', 'replan_resp'
	Payload  interface{} `json:"payload"`
}

type CognitiveTriad struct {
	Charc   *CharcAuditor
	Nine    *NineStrategist
	msgChan chan TriadMessage
}

func NewCognitiveTriad(localAuditorProvider, reasoningProvider providers.Provider) *CognitiveTriad {
	return &CognitiveTriad{
		Charc:   NewCharcAuditor(localAuditorProvider),
		Nine:    NewNineStrategist(reasoningProvider),
		msgChan: make(chan TriadMessage, 100),
	}
}

// SuperviseStep intercepta y valida el paso antes y durante la ejecución
func (t *CognitiveTriad) SuperviseStep(
	ctx context.Context,
	task models.AgentTask,
	step *models.TaskStep,
	recentLogs string,
) (*AuditDecision, *StrategicDiagnosis, error) {
	// 1. Auditoría Previa por CHARC
	decision := t.Charc.AuditPreExecution(ctx, task, step)
	log.Printf("[triad:charc] Paso %s evaluado. Permite: %v, Requiere PIN: %v, Riesgo: %s",
		step.ID, decision.AllowExecution, decision.RequiresPIN, decision.RiskLevel)

	// Si no requiere escalamiento, devuelve la decisión de Charc
	if !decision.EscalateToNine {
		return &decision, nil, nil
	}

	// 2. Escalamiento a NINE si CHARC detecta bloqueo o bucle crítico
	log.Printf("[triad:nine] Escalando bloqueo del paso %s a Nine para razonamiento profundo...", step.ID)
	diagnosis, err := t.Nine.FormulateAlternativeStrategy(ctx, task, step, decision.Reason, recentLogs)
	if err != nil {
		return &decision, nil, fmt.Errorf("error obteniendo estrategia de Nine: %w", err)
	}

	return &decision, diagnosis, nil
}

// Supervise implementa la interfaz taskengine.Supervisor
func (t *CognitiveTriad) Supervise(
	ctx context.Context,
	task models.AgentTask,
	step *models.TaskStep,
	recentLogs string,
) (*taskengine.TriadAuditResult, error) {
	decision, diagnosis, err := t.SuperviseStep(ctx, task, step, recentLogs)
	if err != nil {
		return nil, err
	}
	res := &taskengine.TriadAuditResult{
		AllowExecution: decision.AllowExecution,
		RequiresPIN:    decision.RequiresPIN,
		RiskLevel:      string(decision.RiskLevel),
		Reason:         decision.Reason,
		EscalateToNine: decision.EscalateToNine,
	}
	if diagnosis != nil && len(diagnosis.AlternativePlan) > 0 {
		res.AlternativePayload = diagnosis.AlternativePlan[0].Payload
	}
	return res, nil
}
