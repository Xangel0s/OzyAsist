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

func TestExecOSTileWindows_LayoutsAndValidation(t *testing.T) {
	ctx := context.Background()

	// 1. Validar show_desktop
	tcDesk := providers.ToolCall{
		ID:   "call-tile-desktop",
		Name: "os_tile_windows",
		Input: json.RawMessage(`{
			"layout": "show_desktop"
		}`),
	}
	outDesk, okDesk := execOSTileWindows(ctx, tcDesk)
	if !okDesk || !strings.Contains(outDesk, "Escritorio mostrado") {
		t.Errorf("execOSTileWindows show_desktop falló: %s", outDesk)
	}

	// 2. Ventana inexistente
	tcNotExist := providers.ToolCall{
		ID:   "call-tile-notexist",
		Name: "os_tile_windows",
		Input: json.RawMessage(`{
			"title": "VentanaTotalmenteInexistente_12345",
			"layout": "left"
		}`),
	}
	outNotExist, okNotExist := execOSTileWindows(ctx, tcNotExist)
	if okNotExist {
		t.Errorf("Se esperaba fallo con ventana inexistente, pero tuvo éxito: %s", outNotExist)
	}
}

func TestExecOSDiskCleaner_AnalyzeAndClean(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	// Crear algunos archivos simulados
	_ = os.WriteFile(filepath.Join(tmpDir, "temp1.tmp"), []byte("sample test data"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "temp2.tmp"), []byte("another temp file"), 0644)

	// 1. Modo análisis
	tcAnalyze := providers.ToolCall{
		ID:   "call-cleaner-analyze",
		Name: "os_disk_cleaner",
		Input: json.RawMessage(`{
			"action": "analyze",
			"target_dir": "` + strings.ReplaceAll(tmpDir, `\`, `\\`) + `"
		}`),
	}
	outAnalyze, okAnalyze := execOSDiskCleaner(ctx, tcAnalyze)
	if !okAnalyze {
		t.Fatalf("execOSDiskCleaner analyze falló: %s", outAnalyze)
	}
	if !strings.Contains(outAnalyze, "ANÁLISIS DE ESPACIO PURGABLE") {
		t.Errorf("Se esperaba informe de análisis, se obtuvo: %s", outAnalyze)
	}

	// 2. Modo limpieza segura en carpeta aislada de test
	tcClean := providers.ToolCall{
		ID:   "call-cleaner-clean",
		Name: "os_disk_cleaner",
		Input: json.RawMessage(`{
			"action": "clean",
			"target_dir": "` + strings.ReplaceAll(tmpDir, `\`, `\\`) + `",
			"empty_recycle_bin": false
		}`),
	}
	outClean, okClean := execOSDiskCleaner(ctx, tcClean)
	if !okClean {
		t.Fatalf("execOSDiskCleaner clean falló: %s", outClean)
	}
	if !strings.Contains(outClean, "Limpieza completada") {
		t.Errorf("Se esperaba confirmación de limpieza, se obtuvo: %s", outClean)
	}
}

func TestExecOSAudioDevice_List(t *testing.T) {
	ctx := context.Background()

	tcList := providers.ToolCall{
		ID:    "call-audio-list",
		Name:  "os_audio_device",
		Input: json.RawMessage(`{"action": "list"}`),
	}
	outList, okList := execOSAudioDevice(ctx, tcList)
	if !okList {
		t.Fatalf("execOSAudioDevice list falló: %s", outList)
	}
	t.Logf("Dispositivos de audio: %s", outList)
	if len(outList) == 0 {
		t.Errorf("Se esperaba respuesta con dispositivos de audio")
	}
}

func TestExecOSWifiManager_Status(t *testing.T) {
	ctx := context.Background()

	tcStatus := providers.ToolCall{
		ID:    "call-wifi-status",
		Name:  "os_wifi_manager",
		Input: json.RawMessage(`{"action": "status"}`),
	}
	outStatus, okStatus := execOSWifiManager(ctx, tcStatus)
	if !okStatus {
		t.Fatalf("execOSWifiManager status falló: %s", outStatus)
	}
	t.Logf("Estado de Wi-Fi: %s", outStatus)
	if len(outStatus) == 0 {
		t.Errorf("Se esperaba telemetría de Wi-Fi")
	}
}

func TestExecOSScheduleTask_ValidationAndList(t *testing.T) {
	ctx := context.Background()

	// 1. Listar tareas programadas
	tcList := providers.ToolCall{
		ID:    "call-task-list",
		Name:  "os_schedule_task",
		Input: json.RawMessage(`{"action": "list"}`),
	}
	outList, okList := execOSScheduleTask(ctx, tcList)
	if !okList {
		t.Fatalf("execOSScheduleTask list falló: %s", outList)
	}
	t.Logf("Tareas programadas: %s", outList)

	// 2. Validación de creación con campos vacíos
	tcCreateInvalid := providers.ToolCall{
		ID:    "call-task-create-invalid",
		Name:  "os_schedule_task",
		Input: json.RawMessage(`{"action": "create"}`),
	}
	outCreate, okCreate := execOSScheduleTask(ctx, tcCreateInvalid)
	if okCreate {
		t.Errorf("Se esperaba fallo por parámetros vacíos, pero tuvo éxito: %s", outCreate)
	}
}
