package agent

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

//go:embed mcts_planner.py
var embeddedMCTSPlannerScript string

func init() {
	AgentTools = append(AgentTools,
		providers.ToolDef{
			Name: "nine_strategic_plan",
			Description: "Invoca al estratega cognitivo NINE para razonamiento profundo, pensamiento extendido y formulación de planes estratégicos multi-etapa con soporte de scripts Python para tareas analíticas o complejas.",
			InputSchema: mustJSON(`{
				"type": "object",
				"properties": {
					"objective": {
						"type": "string",
						"description": "Meta u objetivo complejo a analizar y planificar"
					},
					"contextData": {
						"type": "string",
						"description": "Datos de contexto, archivos involucrados o estado actual del sistema"
					},
					"executeScratchpad": {
						"type": "boolean",
						"description": "Si es true, ejecuta automáticamente el script Python analítico de NINE en la mesa de trabajo para recolectar datos reales"
					}
				},
				"required": ["objective"]
			}`),
		},
		providers.ToolDef{
			Name: "nine_mcts_solve",
			Description: "Ejecuta Monte Carlo Tree Search (MCTS con UCB1/PUCT) en Python 3.11 para resolver trayectorias óptimas de acciones o sintetizar rutas de recuperación ante errores de herramientas.",
			InputSchema: mustJSON(`{
				"type": "object",
				"properties": {
					"goal": {
						"type": "string",
						"description": "Meta u objetivo a resolver mediante exploración de árbol MCTS"
					},
					"contextData": {
						"type": "string",
						"description": "Contexto adicional del sistema o archivos involucrados"
					},
					"failedTool": {
						"type": "string",
						"description": "Herramienta que falló si se busca recuperación de error"
					},
					"errorMessage": {
						"type": "string",
						"description": "Mensaje de error observado en la herramienta previa"
					},
					"iterations": {
						"type": "integer",
						"description": "Número de simulaciones MCTS a explorar (por defecto 60)"
					}
				},
				"required": ["goal"]
			}`),
		},
	)
}

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

type StrategicPlanStep struct {
	Order           int    `json:"order"`
	Action          string `json:"action"`
	Description     string `json:"description"`
	ExpectedOutcome string `json:"expected_outcome"`
}

type StrategicPlan struct {
	Title            string              `json:"title"`
	Summary          string              `json:"summary"`
	Reasoning        string              `json:"reasoning"`
	PythonScratchpad string              `json:"python_scratchpad,omitempty"`
	Steps            []StrategicPlanStep `json:"steps"`
	RequiredTools    []string            `json:"required_tools"`
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

// FormulateStrategicPlan genera un plan de pensamiento profundo con criterio superior y opcionalmente código Python analítico
func (n *NineStrategist) FormulateStrategicPlan(
	ctx context.Context,
	objective string,
	contextData string,
) (*StrategicPlan, error) {
	prov := n.reasoningProvider
	if prov == nil {
		prov = providers.GetDefaultOrFirstProvider()
	}
	if prov == nil {
		return nil, fmt.Errorf("no hay ningún proveedor LLM disponible para NINE")
	}

	prompt := fmt.Sprintf(`Eres NINE, el Estratega Cognitivo y Motor de Razonamiento Superior de OzyAssist.
Tu misión es aplicar criterio técnico de alto nivel, descomponer problemas complejos y formular un plan de ejecución impecable.

Tienes a tu disposición:
1. Herramientas del sistema OzyAssist (Win32 nativo, cámara en Rust, explorador de archivos, red, ventanas).
2. Un entorno nativo Python 3.11 en la mesa de trabajo (~/.ozy/workspace) con pandas, openpyxl, pillow, opencv, pymupdf, requests, etc.
3. El ejecutor central OZY que llevará a cabo tus instrucciones.

=== OBJETIVO A PLANIFICAR ===
%s

=== CONTEXTO ADICIONAL ===
%s

=== INSTRUCCIONES ===
1. Analiza los requisitos, las dependencias y los riesgos de alucinación o datos incompletos.
2. Si el problema involucra inspeccionar datos, parsear documentos, procesar números o contrastar información, DISEÑA un script Python (python_scratchpad) que se ejecutará en la mesa de trabajo para obtener datos 100%% reales.
3. Desglosa los pasos que OZY debe seguir de forma ordenada y concisa.
4. Devuelve ESTRICTAMENTE un JSON con la siguiente estructura:
{
  "title": "Título técnico del plan",
  "summary": "Resumen ejecutivo del plan (1-2 oraciones)",
  "reasoning": "Pensamiento analítico detallado: por qué este enfoque es óptimo, qué riesgos se mitigan y qué criterio técnico se aplica",
  "python_scratchpad": "# Código Python para ejecutar en ~/.ozy/workspace si aplica, o vacío si no se requiere",
  "required_tools": ["os_python_exec", "os_explore", "..."],
  "steps": [
    {
      "order": 1,
      "action": "Nombre de herramienta o acción",
      "description": "Detalle técnico de qué hacer en este paso",
      "expected_outcome": "Resultado esperado verificable"
    }
  ]
}`, objective, contextData)

	messages := []providers.Message{
		{Role: "user", Content: prompt},
	}

	chunkCh, err := prov.StreamCompletion(ctx, messages, providers.CompletionOptions{
		Temperature: 0.3,
		MaxTokens:   3500,
		Stream:      false,
	})
	if err != nil {
		return nil, fmt.Errorf("fallo consultando a NINE: %w", err)
	}

	var fullText string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			fullText += chunk.Content
		}
	}

	var plan StrategicPlan
	if err := parseJSONResponse(fullText, &plan); err != nil {
		return nil, fmt.Errorf("error parseando plan de NINE: %w (Respuesta: %s)", err, fullText)
	}

	return &plan, nil
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

func execNineStrategicPlan(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Objective         string `json:"objective"`
		ContextData       string `json:"contextData"`
		ExecuteScratchpad bool   `json:"executeScratchpad"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil || strings.TrimSpace(params.Objective) == "" {
		return "❌ Parámetros inválidos para nine_strategic_plan: se requiere 'objective'.", false
	}

	prov := providers.GetDefaultOrFirstProvider()
	if prov == nil {
		for _, name := range providers.Available() {
			if p, err := providers.Get(name); err == nil && p != nil {
				prov = p
				break
			}
		}
	}
	strategist := NewNineStrategist(prov)
	plan, err := strategist.FormulateStrategicPlan(ctx, params.Objective, params.ContextData)
	if err != nil {
		return fmt.Sprintf("⚠️ NINE no pudo generar el plan completo: %v", err), false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🧠 === PLAN ESTRATÉGICO DE NINE: %s ===\n", strings.ToUpper(plan.Title)))
	sb.WriteString(fmt.Sprintf("📋 Resumen: %s\n\n", plan.Summary))

	if plan.Reasoning != "" {
		sb.WriteString("🤔 Criterio y Pensamiento Profundo:\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", plan.Reasoning))
	}

	if len(plan.Steps) > 0 {
		sb.WriteString("📌 Secuencia de Pasos Estratégicos:\n")
		for _, st := range plan.Steps {
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n     -> Resultado Esperado: %s\n",
				st.Order, st.Action, st.Description, st.ExpectedOutcome))
		}
		sb.WriteString("\n")
	}

	if strings.TrimSpace(plan.PythonScratchpad) != "" {
		sb.WriteString("🐍 Código Analítico Python en Mesa de Trabajo:\n")
		sb.WriteString("```python\n")
		sb.WriteString(plan.PythonScratchpad)
		sb.WriteString("\n```\n")

		// Ejecutar opcionalmente el scratchpad
		if params.ExecuteScratchpad {
			sb.WriteString("\n⚙️ Ejecutando Scratchpad de NINE en ~/.ozy/workspace/...\n")
			pyOut, ok := ExecutePythonInWorkspace(ctx, plan.PythonScratchpad, "nine_scratchpad.py", 45)
			if ok {
				sb.WriteString("✅ Salida del Scratchpad:\n" + pyOut + "\n")
			} else {
				sb.WriteString("⚠️ Advertencia en Scratchpad:\n" + pyOut + "\n")
			}
		}
	}

	return sb.String(), true
}

// MCTSTrajectoryStep representa un nodo/paso en la trayectoria óptima calculada por MCTS.
type MCTSTrajectoryStep struct {
	Step        int            `json:"step"`
	ActionName  string         `json:"action_name"`
	Tool        string         `json:"tool"`
	Args        map[string]any `json:"args"`
	Rationale   string         `json:"rationale"`
	Visits      int            `json:"visits"`
	Confidence  float64        `json:"confidence"`
}

// MCTSResult es el resultado de la simulación de árbol de Monte Carlo.
type MCTSResult struct {
	Status            string               `json:"status"`
	Goal              string               `json:"goal"`
	TotalSimulations  int                  `json:"total_simulations"`
	DurationMs        float64              `json:"duration_ms"`
	OptimalTrajectory []MCTSTrajectoryStep `json:"optimal_trajectory"`
	RecoveryAdvice    string               `json:"recovery_advice"`
}

// EnsureMCTSPlannerScript despliega el script mcts_planner.py en ~/.ozy/workspace si no existe.
func EnsureMCTSPlannerScript() (string, error) {
	wsDir := GetWorkspaceDir()
	scriptPath := filepath.Join(wsDir, "mcts_planner.py")

	// Si no existe o tiene tamaño 0, escribirlo desde el recurso embebido
	if fi, err := os.Stat(scriptPath); err != nil || fi.Size() == 0 {
		if strings.TrimSpace(embeddedMCTSPlannerScript) == "" {
			return "", fmt.Errorf("el script mcts_planner.py embebido está vacío")
		}
		if err := os.WriteFile(scriptPath, []byte(embeddedMCTSPlannerScript), 0644); err != nil {
			return "", fmt.Errorf("error escribiendo mcts_planner.py en mesa de trabajo: %w", err)
		}
	}
	return scriptPath, nil
}

// SolveWithMCTS ejecuta el planificador Monte Carlo Tree Search en Python y retorna la trayectoria óptima.
func SolveWithMCTS(ctx context.Context, goal, contextData, failedTool, errorMessage string, iterations int) (*MCTSResult, error) {
	scriptPath, err := EnsureMCTSPlannerScript()
	if err != nil {
		return nil, err
	}

	if iterations <= 0 {
		iterations = 60
	}

	pyExe := FindPythonExecutable()
	args := []string{
		scriptPath,
		"--goal", goal,
		"--iterations", strconv.Itoa(iterations),
	}
	if contextData != "" {
		args = append(args, "--context", contextData)
	}
	if failedTool != "" {
		args = append(args, "--failed_tool", failedTool)
	}
	if errorMessage != "" {
		args = append(args, "--error_msg", errorMessage)
	}

	cmd := exec.CommandContext(ctx, pyExe, args...)
	cmd.Dir = GetWorkspaceDir()
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error ejecutando mcts_planner.py (%v): %s", err, string(outBytes))
	}

	var res MCTSResult
	rawStr := strings.TrimSpace(string(outBytes))
	if err := parseJSONResponse(rawStr, &res); err != nil {
		return nil, fmt.Errorf("error deserializando salida MCTS: %w (salida: %s)", err, rawStr)
	}

	return &res, nil
}

func execNineMCTSSolve(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Goal         string `json:"goal"`
		ContextData  string `json:"contextData"`
		FailedTool   string `json:"failedTool"`
		ErrorMessage string `json:"errorMessage"`
		Iterations   int    `json:"iterations"`
	}

	if err := json.Unmarshal(tc.Input, &params); err != nil || strings.TrimSpace(params.Goal) == "" {
		return "❌ Parámetros inválidos para nine_mcts_solve: se requiere 'goal'.", false
	}

	res, err := SolveWithMCTS(ctx, params.Goal, params.ContextData, params.FailedTool, params.ErrorMessage, params.Iterations)
	if err != nil {
		return fmt.Sprintf("⚠️ Error en simulador MCTS: %v", err), false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🌳 === NINE MCTS PLANNER (UCB1) ===\n"))
	sb.WriteString(fmt.Sprintf("🎯 Objetivo: %s\n", res.Goal))
	sb.WriteString(fmt.Sprintf("⚡ Simulaciones: %d en %.2f ms\n\n", res.TotalSimulations, res.DurationMs))

	if res.RecoveryAdvice != "" {
		sb.WriteString("🚨 RUTA DE RECUPERACIÓN ÓPTIMA:\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", res.RecoveryAdvice))
	}

	if len(res.OptimalTrajectory) > 0 {
		sb.WriteString("📌 TRAYECTORIA ÓPTIMA RECOMENDADA:\n")
		for _, step := range res.OptimalTrajectory {
			argsJSON, _ := json.Marshal(step.Args)
			sb.WriteString(fmt.Sprintf("  Paso %d: [%s] (Herramienta: `%s`)\n", step.Step, step.ActionName, step.Tool))
			sb.WriteString(fmt.Sprintf("    • Justificación: %s\n", step.Rationale))
			if len(step.Args) > 0 {
				sb.WriteString(fmt.Sprintf("    • Argumentos sugeridos: `%s`\n", string(argsJSON)))
			}
			sb.WriteString(fmt.Sprintf("    • Visitas UCB1: %d | Confianza Q: %.3f\n", step.Visits, step.Confidence))
		}
	}

	return sb.String(), true
}

