package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ozyassist/backend/internal/providers"
)

type mockNineProvider struct {
	response string
}

func (m *mockNineProvider) Name() string { return "mock_nine" }
func (m *mockNineProvider) SupportsTools() bool { return false }
func (m *mockNineProvider) Models() []string { return []string{"mock-model"} }
func (m *mockNineProvider) StreamCompletion(ctx context.Context, messages []providers.Message, opts providers.CompletionOptions) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- providers.StreamChunk{
			Type:    "text",
			Content: m.response,
		}
	}()
	return ch, nil
}

func TestNine_FormulateStrategicPlan(t *testing.T) {
	mockJSON := `{
		"title": "Auditoría de Archivos Grandes y Reporte en Excel",
		"summary": "Analizar almacenamiento con Python y compilar informe estructurado.",
		"reasoning": "El volumen de datos requiere un análisis en memoria mediante pandas para prevenir saturación de disco.",
		"python_scratchpad": "import os\nprint('Espacio libre verificado')",
		"required_tools": ["os_python_exec", "os_create_excel"],
		"steps": [
			{
				"order": 1,
				"action": "os_python_exec",
				"description": "Escanear carpeta y recopilar estadísticas de archivos pesados",
				"expected_outcome": "Dataset en memoria consolidado"
			},
			{
				"order": 2,
				"action": "os_create_excel",
				"description": "Generar reporte formal para el usuario",
				"expected_outcome": "Archivo XLSX generado"
			}
		]
	}`

	mockProv := &mockNineProvider{response: mockJSON}
	providers.Register("mock_nine", mockProv)
	strategist := NewNineStrategist(mockProv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	plan, err := strategist.FormulateStrategicPlan(ctx, "Audita archivos pesados", "Directorio C:/Users/User")
	if err != nil {
		t.Fatalf("FormulateStrategicPlan falló: %v", err)
	}

	if plan.Title != "Auditoría de Archivos Grandes y Reporte en Excel" {
		t.Errorf("título inesperado: %s", plan.Title)
	}
	if len(plan.Steps) != 2 {
		t.Errorf("esperaba 2 pasos, obtuve %d", len(plan.Steps))
	}
	if !strings.Contains(plan.PythonScratchpad, "Espacio libre verificado") {
		t.Errorf("scratchpad no contiene código esperado: %s", plan.PythonScratchpad)
	}

	// Test execNineStrategicPlan ToolCall
	tc := providers.ToolCall{
		ID:    "call_nine_plan",
		Name:  "nine_strategic_plan",
		Input: []byte(`{"objective": "Planifica optimización de disco", "executeScratchpad": true}`),
	}
	out, ok := execNineStrategicPlan(ctx, tc)
	if !ok || !strings.Contains(out, "PLAN ESTRATÉGICO DE NINE") {
		t.Errorf("execNineStrategicPlan falló o salida inesperada: %s", out)
	}
}

func TestNine_MCTSSolve(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Resolver objetivo normal con MCTS
	res, err := SolveWithMCTS(ctx, "capturar foto de la camara y verificar", "", "", "", 50)
	if err != nil {
		t.Fatalf("SolveWithMCTS falló: %v", err)
	}
	if res.Status != "success" {
		t.Errorf("esperaba status success, obtuve: %s", res.Status)
	}
	if len(res.OptimalTrajectory) == 0 {
		t.Errorf("trayectoria MCTS vacía")
	}

	// 2. Resolver recuperación de error
	resErr, err := SolveWithMCTS(ctx, "detener proceso bloqueado", "", "os_close_window", "Acceso denegado error 740", 50)
	if err != nil {
		t.Fatalf("SolveWithMCTS error recovery falló: %v", err)
	}
	if resErr.RecoveryAdvice == "" {
		t.Errorf("esperaba consejo de recuperación en resErr")
	}

	// 3. Probar herramienta nine_mcts_solve
	tc := providers.ToolCall{
		ID:    "call_mcts_test",
		Name:  "nine_mcts_solve",
		Input: []byte(`{"goal": "optimizar archivos pesados", "iterations": 40}`),
	}
	out, ok := execNineMCTSSolve(ctx, tc)
	if !ok || !strings.Contains(out, "NINE MCTS PLANNER") {
		t.Errorf("execNineMCTSSolve falló: %s", out)
	}
}

func TestNine_QuerySystemKnowledge(t *testing.T) {
	ctx := context.Background()
	tc := providers.ToolCall{
		ID:    "call_sys_kn",
		Name:  "query_system_knowledge",
		Input: []byte(`{"query": "camara nokhwa en rust", "topK": 2}`),
	}
	out, ok := execQuerySystemKnowledge(ctx, tc)
	if !ok || !strings.Contains(out, "CONOCIMIENTO ONTOLÓGICO DEL SISTEMA") {
		t.Errorf("execQuerySystemKnowledge falló: %s", out)
	}
	if !strings.Contains(out, "Cámara en Rust") && !strings.Contains(out, "os_camera_capture") {
		t.Errorf("resultado no contiene nodo esperado: %s", out)
	}
}

