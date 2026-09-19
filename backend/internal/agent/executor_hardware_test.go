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

func TestExecOSAudioDevice_VolumeAndMute(t *testing.T) {
	ctx := context.Background()

	// 1. get_volume
	tcGet := providers.ToolCall{
		ID:    "call-audio-get-vol",
		Name:  "os_audio_device",
		Input: json.RawMessage(`{"action": "get_volume"}`),
	}
	outGet, okGet := execOSAudioDevice(ctx, tcGet)
	if !okGet {
		t.Fatalf("execOSAudioDevice get_volume falló: %s", outGet)
	}
	t.Logf("Volumen actual: %s", outGet)
	if !strings.Contains(outGet, "Volumen maestro:") {
		t.Errorf("Se esperaba reporte de volumen maestro, se obtuvo: %s", outGet)
	}

	// 2. set_volume sin level
	tcSetInvalid := providers.ToolCall{
		ID:    "call-audio-set-invalid",
		Name:  "os_audio_device",
		Input: json.RawMessage(`{"action": "set_volume"}`),
	}
	outSetInvalid, okSetInvalid := execOSAudioDevice(ctx, tcSetInvalid)
	if okSetInvalid {
		t.Errorf("Se esperaba fallo por falta de level en set_volume, pero tuvo éxito: %s", outSetInvalid)
	}
}

func TestExecOSHardwareInspector_Modes(t *testing.T) {
	ctx := context.Background()

	// 1. Modo devices / usb
	tcDevices := providers.ToolCall{
		ID:    "call-hw-devices",
		Name:  "os_hardware_inspector",
		Input: json.RawMessage(`{"action": "devices"}`),
	}
	outDev, okDev := execOSHardwareInspector(ctx, tcDevices)
	if !okDev {
		t.Fatalf("execOSHardwareInspector devices falló: %s", outDev)
	}
	t.Logf("Dispositivos físicos: %s", outDev)
	if !strings.Contains(outDev, "DISPOSITIVOS Y PUERTOS FÍSICOS CONECTADOS") {
		t.Errorf("Se esperaba encabezado de dispositivos conectados: %s", outDev)
	}

	// 2. Modo in_use / privacy
	tcInUse := providers.ToolCall{
		ID:    "call-hw-inuse",
		Name:  "os_hardware_inspector",
		Input: json.RawMessage(`{"action": "in_use"}`),
	}
	outInUse, okInUse := execOSHardwareInspector(ctx, tcInUse)
	if !okInUse {
		t.Fatalf("execOSHardwareInspector in_use falló: %s", outInUse)
	}
	t.Logf("Estado de periféricos en uso: %s", outInUse)
	if !strings.Contains(outInUse, "Cámara Web:") || !strings.Contains(outInUse, "Micrófono:") {
		t.Errorf("Se esperaba estado de cámara y micrófono: %s", outInUse)
	}

	// 3. Modo telemetry / system
	tcTelem := providers.ToolCall{
		ID:    "call-hw-telemetry",
		Name:  "os_hardware_inspector",
		Input: json.RawMessage(`{"action": "telemetry"}`),
	}
	outTelem, okTelem := execOSHardwareInspector(ctx, tcTelem)
	if !okTelem {
		t.Fatalf("execOSHardwareInspector telemetry falló: %s", outTelem)
	}
	t.Logf("Telemetría de hardware: %s", outTelem)
	if !strings.Contains(outTelem, "TELEMETRÍA DE HARDWARE") || !strings.Contains(outTelem, "CPU:") || !strings.Contains(outTelem, "RAM:") {
		t.Errorf("Se esperaba telemetría de CPU y RAM: %s", outTelem)
	}
}

