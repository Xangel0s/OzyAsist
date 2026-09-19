package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ozyassist/backend/internal/providers"
)

func TestExecOSKeyboardLayout_StatusAndList(t *testing.T) {
	ctx := context.Background()

	// 1. Status
	tcStatus := providers.ToolCall{
		ID:    "call-kb-status",
		Name:  "os_keyboard_layout",
		Input: json.RawMessage(`{"action": "status"}`),
	}
	outStatus, okStatus := execOSKeyboardLayout(ctx, tcStatus)
	if !okStatus {
		t.Fatalf("execOSKeyboardLayout status falló: %s", outStatus)
	}
	t.Logf("Estado de teclado: %s", outStatus)
	if !strings.Contains(outStatus, "DISTRIBUCIÓN DE TECLADO") || !strings.Contains(outStatus, "Distribución activa:") {
		t.Errorf("Se esperaba encabezado y distribución activa: %s", outStatus)
	}

	// 2. List
	tcList := providers.ToolCall{
		ID:    "call-kb-list",
		Name:  "os_keyboard_layout",
		Input: json.RawMessage(`{"action": "list"}`),
	}
	outList, okList := execOSKeyboardLayout(ctx, tcList)
	if !okList {
		t.Fatalf("execOSKeyboardLayout list falló: %s", outList)
	}
	t.Logf("Teclados instalados: %s", outList)
	if !strings.Contains(outList, "DISTRIBUCIONES DE TECLADO INSTALADAS") {
		t.Errorf("Se esperaba listado de teclados: %s", outList)
	}
}

func TestExecOSKeyboardLayout_SwitchValidation(t *testing.T) {
	ctx := context.Background()

	// Error sin layout
	tcEmpty := providers.ToolCall{
		ID:    "call-kb-empty",
		Name:  "os_keyboard_layout",
		Input: json.RawMessage(`{"action": "set", "layout": ""}`),
	}
	outEmpty, okEmpty := execOSKeyboardLayout(ctx, tcEmpty)
	if okEmpty {
		t.Fatalf("Se esperaba fallo al pasar layout vacío: %s", outEmpty)
	}

	// Cambio válido a 'latam'
	tcLatam := providers.ToolCall{
		ID:    "call-kb-latam",
		Name:  "os_keyboard_layout",
		Input: json.RawMessage(`{"action": "set", "layout": "latam"}`),
	}
	outLatam, okLatam := execOSKeyboardLayout(ctx, tcLatam)
	if !okLatam {
		t.Fatalf("Fallo cambiando a latam: %s", outLatam)
	}
	t.Logf("Resultado cambio a latam: %s", outLatam)
	if !strings.Contains(outLatam, "0000080A") {
		t.Errorf("Se esperaba KLID 0000080A en la respuesta: %s", outLatam)
	}
}

func TestExecOSStartupManager_ListAndAddRemove(t *testing.T) {
	ctx := context.Background()

	// 1. List
	tcList := providers.ToolCall{
		ID:    "call-startup-list",
		Name:  "os_startup_manager",
		Input: json.RawMessage(`{"action": "list"}`),
	}
	outList, okList := execOSStartupManager(ctx, tcList)
	if !okList {
		t.Fatalf("execOSStartupManager list falló: %s", outList)
	}
	t.Logf("Programas de inicio: %s", outList)
	if !strings.Contains(outList, "PROGRAMAS DE INICIO AUTOMÁTICO") {
		t.Errorf("Se esperaba encabezado de programas de inicio: %s", outList)
	}

	// 2. Add
	testName := "OzyAssist_TestEntry"
	testCmd := `C:\Windows\System32\cmd.exe /c echo test`
	tcAdd := providers.ToolCall{
		ID:    "call-startup-add",
		Name:  "os_startup_manager",
		Input: json.RawMessage(`{"action": "add", "name": "` + testName + `", "command": "` + strings.ReplaceAll(testCmd, `\`, `\\`) + `"}`),
	}
	outAdd, okAdd := execOSStartupManager(ctx, tcAdd)
	if !okAdd {
		t.Fatalf("execOSStartupManager add falló: %s", outAdd)
	}
	t.Logf("Resultado add: %s", outAdd)

	// 3. Remove
	tcRemove := providers.ToolCall{
		ID:    "call-startup-remove",
		Name:  "os_startup_manager",
		Input: json.RawMessage(`{"action": "remove", "name": "` + testName + `"}`),
	}
	outRemove, okRemove := execOSStartupManager(ctx, tcRemove)
	if !okRemove {
		t.Fatalf("execOSStartupManager remove falló: %s", outRemove)
	}
	t.Logf("Resultado remove: %s", outRemove)
	if !strings.Contains(outRemove, "retirado exitosamente") {
		t.Errorf("Se esperaba confirmación de eliminación: %s", outRemove)
	}
}

func TestExecOSNotificationFocus_StatusAndToggle(t *testing.T) {
	ctx := context.Background()

	// 1. Status
	tcStatus := providers.ToolCall{
		ID:    "call-notif-status",
		Name:  "os_notification_focus",
		Input: json.RawMessage(`{"action": "status"}`),
	}
	outStatus, okStatus := execOSNotificationFocus(ctx, tcStatus)
	if !okStatus {
		t.Fatalf("execOSNotificationFocus status falló: %s", outStatus)
	}
	t.Logf("Estado notificaciones: %s", outStatus)
	if !strings.Contains(outStatus, "ESTADO DE NOTIFICACIONES DE WINDOWS") {
		t.Errorf("Se esperaba encabezado de notificaciones: %s", outStatus)
	}

	// 2. Set habilitado
	enabled := true
	tcSet := providers.ToolCall{
		ID:    "call-notif-set",
		Name:  "os_notification_focus",
		Input: json.RawMessage(`{"action": "set", "enabled": true}`),
	}
	outSet, okSet := execOSNotificationFocus(ctx, tcSet)
	if !okSet {
		t.Fatalf("execOSNotificationFocus set falló: %s", outSet)
	}
	t.Logf("Resultado set habilitado: %s", outSet)
	_ = enabled
}

func TestExecOSClipboard_ReadAndWrite(t *testing.T) {
	ctx := context.Background()

	// 1. Escribir texto al portapapeles
	testText := "TextoDePruebaParaOzyAssistPortapapeles"
	tcWrite := providers.ToolCall{
		ID:    "call-clip-write",
		Name:  "os_set_clipboard",
		Input: json.RawMessage(`{"text": "` + testText + `"}`),
	}
	outWrite, okWrite := execOSSetClipboard(ctx, tcWrite)
	if !okWrite {
		t.Fatalf("execOSSetClipboard falló: %s", outWrite)
	}
	t.Logf("Resultado write: %s", outWrite)

	// 2. Leer texto del portapapeles
	outRead, okRead := execOSGetClipboard(ctx)
	if !okRead {
		t.Fatalf("execOSGetClipboard falló: %s", outRead)
	}
	t.Logf("Resultado read: %s", outRead)
	if !strings.Contains(outRead, testText) {
		t.Errorf("Se esperaba contenido '%s' en el portapapeles: %s", testText, outRead)
	}
}
