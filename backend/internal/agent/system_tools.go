package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ozyassist/backend/internal/browser"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

// execBrowserListProfiles lista los perfiles de navegador detectados
func execBrowserListProfiles(_ context.Context) (string, bool) {
	profiles := browser.DetectProfiles()
	if len(profiles) == 0 {
		return "No se detectaron perfiles de navegador en las rutas estándar.", true
	}

	var sb strings.Builder
	sb.WriteString("🌐 === PERFILES DE NAVEGADOR DETECTADOS ===\n")
	for _, p := range profiles {
		sb.WriteString(fmt.Sprintf("• [%s] Perfil: %q (Dir: %s)\n", strings.ToUpper(p.Browser), p.Name, p.Directory))
		if p.Email != "" {
			sb.WriteString(fmt.Sprintf("  - Cuenta: %s\n", p.Email))
		}
		if p.DisplayName != "" {
			sb.WriteString(fmt.Sprintf("  - Titular: %s\n", p.DisplayName))
		}
		if p.ExecPath != "" {
			sb.WriteString(fmt.Sprintf("  - Ejecutable: %s\n", p.ExecPath))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("💡 Puedes usar cualquiera de estas cuentas o nombres en 'os_draft_email' (parámetro from_account) para abrir Gmail directamente con esa sesión autenticada.")

	return sb.String(), true
}

// execOSGetClipboard lee el portapapeles del sistema
func execOSGetClipboard(_ context.Context) (string, bool) {
	text, err := system.ReadClipboardContent()
	if err != nil {
		return fmt.Sprintf("Error leyendo portapapeles: %v", err), false
	}
	if text == "" {
		return "El portapapeles está vacío o contiene datos no textuales (como imágenes).", true
	}
	return fmt.Sprintf("[CONTENIDO DEL PORTAPAPELES]:\n%s", text), true
}

// execOSSetClipboard escribe en el portapapeles del sistema
func execOSSetClipboard(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos: %v", err), false
	}
	if err := system.WriteClipboardContent(params.Text); err != nil {
		return fmt.Sprintf("Error escribiendo en portapapeles: %v", err), false
	}
	return fmt.Sprintf("Texto copiado al portapapeles exitosamente (%d caracteres).", len(params.Text)), true
}

// SendToastNotification envía una notificación toast nativa en Windows
func SendToastNotification(title, message string) {
	if runtime.GOOS == "windows" {
		script := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$xml = [xml]$template.GetXml()
$xml.GetElementsByTagName('text')[0].AppendChild($xml.CreateTextNode('%s')) | Out-Null
$xml.GetElementsByTagName('text')[1].AppendChild($xml.CreateTextNode('%s')) | Out-Null
$toastXml = New-Object Windows.Data.Xml.Dom.XmlDocument
$toastXml.LoadXml($xml.OuterXml)
$toast = [Windows.UI.Notifications.ToastNotification]::new($toastXml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('OzyAssist').Show($toast)
`, strings.ReplaceAll(title, "'", "''"), strings.ReplaceAll(message, "'", "''"))

		cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
		_ = cmd.Run()
	}
}

// execOSNotify envía una notificación toast nativa en Windows
func execOSNotify(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos: %v", err), false
	}

	if strings.TrimSpace(params.Title) == "" {
		params.Title = "OzyAssist"
	}

	SendToastNotification(params.Title, params.Message)
	return fmt.Sprintf("✓ Notificación nativa enviada: %s - %s", params.Title, params.Message), true
}
