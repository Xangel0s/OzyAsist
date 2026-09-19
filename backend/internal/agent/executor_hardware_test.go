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

	// 4. Modo health / salud
	tcHealth := providers.ToolCall{
		ID:    "call-hw-health",
		Name:  "os_hardware_inspector",
		Input: json.RawMessage(`{"action": "health"}`),
	}
	outHealth, okHealth := execOSHardwareInspector(ctx, tcHealth)
	if !okHealth {
		t.Fatalf("execOSHardwareInspector health falló: %s", outHealth)
	}
	t.Logf("Auditoría de salud de hardware: %s", outHealth)
	if !strings.Contains(outHealth, "AUDITORÍA Y SALUD DEL HARDWARE") || !strings.Contains(outHealth, "Diagnóstico General:") {
		t.Errorf("Se esperaba reporte de auditoría de salud: %s", outHealth)
	}
}

func TestExecOSPowerProfile_StatusAndValidation(t *testing.T) {
	ctx := context.Background()

	// 1. status
	tcStatus := providers.ToolCall{
		ID:    "call-power-status",
		Name:  "os_power_profile",
		Input: json.RawMessage(`{"action": "status"}`),
	}
	outStatus, okStatus := execOSPowerProfile(ctx, tcStatus)
	if !okStatus {
		t.Fatalf("execOSPowerProfile status falló: %s", outStatus)
	}
	t.Logf("Perfil de energía: %s", outStatus)
	if !strings.Contains(outStatus, "PERFIL DE ENERGÍA Y PANTALLA") {
		t.Errorf("Se esperaba encabezado de perfil de energía: %s", outStatus)
	}

	// 2. Plan inválido
	tcInvalid := providers.ToolCall{
		ID:    "call-power-invalid",
		Name:  "os_power_profile",
		Input: json.RawMessage(`{"action": "set_plan", "plan": "plan_inexistente"}`),
	}
	outInvalid, okInvalid := execOSPowerProfile(ctx, tcInvalid)
	if okInvalid {
		t.Errorf("Se esperaba fallo con plan inexistente, pero tuvo éxito: %s", outInvalid)
	}
}

func TestExecOSToastNotify_ValidationAndSend(t *testing.T) {
	ctx := context.Background()

	// 1. Fallo sin mensaje
	tcNoMsg := providers.ToolCall{
		ID:    "call-toast-nomsg",
		Name:  "os_toast_notify",
		Input: json.RawMessage(`{"title": "Test"}`),
	}
	outNoMsg, okNoMsg := execOSToastNotify(ctx, tcNoMsg)
	if okNoMsg {
		t.Errorf("Se esperaba fallo sin message, pero tuvo éxito: %s", outNoMsg)
	}

	// 2. Notificación válida
	tcValid := providers.ToolCall{
		ID:    "call-toast-valid",
		Name:  "os_toast_notify",
		Input: json.RawMessage(`{"title": "Ozy Test", "message": "Verificación de prueba unitaria"}`),
	}
	outValid, okValid := execOSToastNotify(ctx, tcValid)
	if !okValid {
		t.Fatalf("execOSToastNotify falló: %s", outValid)
	}
	if !strings.Contains(outValid, "Notificación Toast nativa enviada") {
		t.Errorf("Se esperaba confirmación de toast enviado: %s", outValid)
	}
}

func TestExecOSNetworkDiagnostics_PingAndIPInfo(t *testing.T) {
	ctx := context.Background()

	// 1. IP info
	tcIP := providers.ToolCall{
		ID:    "call-net-ip",
		Name:  "os_network_diagnostics",
		Input: json.RawMessage(`{"action": "ip_info"}`),
	}
	outIP, okIP := execOSNetworkDiagnostics(ctx, tcIP)
	if !okIP {
		t.Fatalf("execOSNetworkDiagnostics ip_info falló: %s", outIP)
	}
	t.Logf("IP Info: %s", outIP)
	if !strings.Contains(outIP, "CONFIGURACIÓN DE RED LOCAL") {
		t.Errorf("Se esperaba encabezado de configuración de red: %s", outIP)
	}

	// 2. Ping
	tcPing := providers.ToolCall{
		ID:    "call-net-ping",
		Name:  "os_network_diagnostics",
		Input: json.RawMessage(`{"action": "test", "host": "1.1.1.1"}`),
	}
	outPing, okPing := execOSNetworkDiagnostics(ctx, tcPing)
	if !okPing {
		t.Fatalf("execOSNetworkDiagnostics ping falló: %s", outPing)
	}
	t.Logf("Diagnóstico de red: %s", outPing)
	if !strings.Contains(outPing, "DIAGNÓSTICO DE RED Y LATENCIA") {
		t.Errorf("Se esperaba resultado de diagnóstico de red: %s", outPing)
	}
}

func TestExecOSSmartOrganizer_Duplicates(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	// Crear 2 archivos idénticos mayores a 50KB
	dummyContent := strings.Repeat("AlineacionDePruebaDeArchivosDuplicadosParaOzyAssist1234567890\n", 1500) // ~90KB
	_ = os.WriteFile(filepath.Join(tmpDir, "archivo_original.bin"), []byte(dummyContent), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "archivo_copia.bin"), []byte(dummyContent), 0644)

	tcDup := providers.ToolCall{
		ID:    "call-dup-test",
		Name:  "os_smart_organizer",
		Input: json.RawMessage(`{"action": "duplicates", "target_dir": "` + strings.ReplaceAll(tmpDir, `\`, `\\`) + `"}`),
	}
	outDup, okDup := execOSSmartOrganizer(ctx, tcDup)
	if !okDup {
		t.Fatalf("execOSSmartOrganizer duplicates falló: %s", outDup)
	}
	t.Logf("Duplicados: %s", outDup)
	if !strings.Contains(outDup, "ARCHIVOS DUPLICADOS DETECTADOS") || !strings.Contains(outDup, "Grupo #1") {
		t.Errorf("Se esperaba detección de duplicados en la carpeta temporal: %s", outDup)
	}
}


