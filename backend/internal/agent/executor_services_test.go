package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ozyassist/backend/internal/providers"
)

func TestExecOSServiceManager_ListAndStatus(t *testing.T) {
	ctx := context.Background()

	// 1. Listar servicios con filtro
	tcList := providers.ToolCall{
		ID:   "call-svc-list",
		Name: "os_service_manager",
		Input: json.RawMessage(`{
			"action": "list",
			"filter": "EventLog"
		}`),
	}

	outList, okList := execOSServiceManager(ctx, tcList)
	if !okList {
		t.Fatalf("execOSServiceManager list falló: %s", outList)
	}
	if !strings.Contains(outList, "SERVICIOS DE WINDOWS") {
		t.Errorf("Se esperaba encabezado de servicios, se obtuvo: %s", outList)
	}

	// 2. Consultar estado específico de EventLog (servicio esencial de Windows siempre presente)
	tcStatus := providers.ToolCall{
		ID:   "call-svc-status",
		Name: "os_service_manager",
		Input: json.RawMessage(`{
			"action": "status",
			"name": "EventLog"
		}`),
	}

	outStatus, okStatus := execOSServiceManager(ctx, tcStatus)
	if !okStatus {
		t.Fatalf("execOSServiceManager status falló: %s", outStatus)
	}
	if !strings.Contains(outStatus, "ESTADO DEL SERVICIO: EventLog") {
		t.Errorf("Se esperaba estado de EventLog, se obtuvo: %s", outStatus)
	}
}

func TestExecOSDockerManager_StatusOrUnavailable(t *testing.T) {
	ctx := context.Background()

	tc := providers.ToolCall{
		ID:    "call-docker-status",
		Name:  "os_docker_manager",
		Input: json.RawMessage(`{"action": "status"}`),
	}

	out, _ := execOSDockerManager(ctx, tc)
	// Puede retornar que el daemon está activo o que no está en ejecución/instalado
	t.Logf("Resultado de Docker status: %s", out)
	if len(out) == 0 {
		t.Errorf("Se esperaba un mensaje descriptivo de docker status, se obtuvo vacío")
	}
}

func TestExecOSAnalyzeLogs_FileAnalysis(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "app.log")
	logContent := `2026-09-19 10:00:01 INFO Starting application
2026-09-19 10:00:02 DEBUG Initializing SQLite in RAM
2026-09-19 10:00:03 WARN Disk space below 10%
2026-09-19 10:00:04 ERROR Database connection timeout occurred at internal/db/pool.go:42
2026-09-19 10:00:05 INFO Retrying connection
2026-09-19 10:00:06 FATAL Panic: nil pointer dereference in agent loop
goroutine 1 [running]:
main.main()
	main.go:15
`
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("Error creando archivo de log: %v", err)
	}

	// 1. Analizar errores
	tc := providers.ToolCall{
		ID:   "call-analyze-file",
		Name: "os_analyze_logs",
		Input: json.RawMessage(`{
			"source": "` + strings.ReplaceAll(logPath, `\`, `\\`) + `",
			"severity": "ERROR"
		}`),
	}

	out, ok := execOSAnalyzeLogs(ctx, tc)
	if !ok {
		t.Fatalf("execOSAnalyzeLogs falló: %s", out)
	}
	if !strings.Contains(out, "Database connection timeout") || !strings.Contains(out, "Panic") {
		t.Errorf("Se esperaba encontrar las líneas de error y panic en el log, se obtuvo: %s", out)
	}

	// 2. Analizar visor de eventos de Windows
	tcWin := providers.ToolCall{
		ID:   "call-analyze-winevent",
		Name: "os_analyze_logs",
		Input: json.RawMessage(`{
			"source": "windows-events",
			"lines": 10
		}`),
	}

	outWin, okWin := execOSAnalyzeLogs(ctx, tcWin)
	if !okWin {
		t.Fatalf("execOSAnalyzeLogs con windows-events falló: %s", outWin)
	}
	t.Logf("Eventos de Windows obtenidos: %s", outWin)
}

func TestExecOSCloseWindow_ValidationAndClose(t *testing.T) {
	ctx := context.Background()

	// 1. Validación de parámetros vacíos
	tcEmpty := providers.ToolCall{
		ID:    "call-close-empty",
		Name:  "os_close_window",
		Input: json.RawMessage(`{}`),
	}
	outEmpty, okEmpty := execOSCloseWindow(ctx, tcEmpty)
	if okEmpty {
		t.Errorf("Se esperaba fallo por falta de parámetros, pero retornó éxito: %s", outEmpty)
	}

	// 2. Cierre de ventana inexistente
	tcNotExist := providers.ToolCall{
		ID:   "call-close-notexist",
		Name: "os_close_window",
		Input: json.RawMessage(`{
			"title": "VentanaTotalmenteInexistenteParaTest_99999"
		}`),
	}
	outNotExist, _ := execOSCloseWindow(ctx, tcNotExist)
	t.Logf("Resultado de ventana inexistente: %s", outNotExist)
}
