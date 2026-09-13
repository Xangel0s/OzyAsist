package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/browser"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

type ScheduledAlarm struct {
	ID         string      `json:"id"`
	Message    string      `json:"message"`
	TargetTime time.Time   `json:"target_time"`
	Duration   string      `json:"duration"`
	Timer      *time.Timer `json:"-"`
}

var (
	alarmsMu     sync.Mutex
	activeAlarms = make(map[string]*ScheduledAlarm)
)

type ScheduleAlarmParams struct {
	TimeIn  string `json:"time_in"` // Duración (ej: "10m", "1h", "45s") o hora fija "15:30"
	Message string `json:"message"` // Mensaje del recordatorio
}

// ParseAlarmDuration analiza cadenas de tiempo relativas ("10m", "1h") o fijas ("15:30")
func ParseAlarmDuration(timeStr string) (time.Duration, error) {
	timeStr = strings.TrimSpace(strings.ToLower(timeStr))
	if d, err := time.ParseDuration(timeStr); err == nil {
		return d, nil
	}

	// Probar formato hora:minuto "15:04"
	now := time.Now()
	if t, err := time.Parse("15:04", timeStr); err == nil {
		target := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		if target.Before(now) {
			target = target.Add(24 * time.Hour)
		}
		return target.Sub(now), nil
	}

	return 0, fmt.Errorf("formato de tiempo no válido ('%s'). Usa '10m', '30s', '1h' o '15:30'", timeStr)
}

func ScheduleAlarm(timeStr, message string) (*ScheduledAlarm, error) {
	duration, err := ParseAlarmDuration(timeStr)
	if err != nil {
		return nil, err
	}

	if duration < 1*time.Second {
		return nil, fmt.Errorf("la duración del recordatorio debe ser de al menos 1 segundo")
	}

	id := uuid.New().String()[:8]
	targetTime := time.Now().Add(duration)

	alarm := &ScheduledAlarm{
		ID:         id,
		Message:    message,
		TargetTime: targetTime,
		Duration:   duration.Round(time.Second).String(),
	}

	alarm.Timer = time.AfterFunc(duration, func() {
		triggerAlarm(alarm)
	})

	alarmsMu.Lock()
	activeAlarms[id] = alarm
	alarmsMu.Unlock()

	return alarm, nil
}

func triggerAlarm(alarm *ScheduledAlarm) {
	alarmsMu.Lock()
	delete(activeAlarms, alarm.ID)
	alarmsMu.Unlock()

	// 1. Mostrar notificación Toast nativa en Windows
	if runtime.GOOS == "windows" {
		script := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$xml = [xml]$template.GetXml()
$xml.GetElementsByTagName('text')[0].AppendChild($xml.CreateTextNode('⏰ Recordatorio de OzyAssist')) | Out-Null
$xml.GetElementsByTagName('text')[1].AppendChild($xml.CreateTextNode('%s')) | Out-Null
$toastXml = New-Object Windows.Data.Xml.Dom.XmlDocument
$toastXml.LoadXml($xml.OuterXml)
$toast = [Windows.UI.Notifications.ToastNotification]::new($toastXml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('OzyAssist').Show($toast)
`, strings.ReplaceAll(alarm.Message, "'", "''"))
		cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
		_ = cmd.Run()
	}

	// 2. Si Piper TTS está instalado localmente, alertar por voz
	piperExe := filepath.Join("tools", "piper", "piper", "piper.exe")
	piperModel := filepath.Join("tools", "piper", "piper", "es_ES-davefx-medium.onnx")
	if _, err := os.Stat(piperExe); err == nil {
		go func() {
			voiceScript := fmt.Sprintf(`& "%s" --model "%s" --output_file "temp_alarm.wav"`, piperExe, piperModel)
			c := exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf(`"%s" | %s; (New-Object Media.SoundPlayer 'temp_alarm.wav').PlaySync()`, fmt.Sprintf("Atención. Recordatorio: %s", alarm.Message), voiceScript))
			_ = c.Run()
		}()
	}
}

func execOSScheduleAlarm(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params ScheduleAlarmParams
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_schedule_alarm: %v", err), false
	}

	if strings.TrimSpace(params.Message) == "" {
		params.Message = "¡Recordatorio activado!"
	}

	alarm, err := ScheduleAlarm(params.TimeIn, params.Message)
	if err != nil {
		return fmt.Sprintf("Error programando alarma: %v", err), false
	}

	return fmt.Sprintf("⏰ === RECORDATORIO PROGRAMADO ===\n• ID:        %s\n• Tiempo:    En %s (%s)\n• Mensaje:   %s\n💡 Cuando se cumpla el tiempo recibirás una notificación de Windows y aviso sonoro.",
		alarm.ID, alarm.Duration, alarm.TargetTime.Format("15:04:05"), alarm.Message), true
}

func execOSListAlarms(_ context.Context) (string, bool) {
	alarmsMu.Lock()
	defer alarmsMu.Unlock()

	if len(activeAlarms) == 0 {
		return "No hay recordatorios ni alarmas pendientes activas.", true
	}

	var sb strings.Builder
	sb.WriteString("⏰ === RECORDATORIOS ACTIVOS ===\n")
	for _, a := range activeAlarms {
		remaining := time.Until(a.TargetTime).Round(time.Second)
		sb.WriteString(fmt.Sprintf("• [ID: %s] Faltan: %s (a las %s) - %s\n", a.ID, remaining, a.TargetTime.Format("15:04:05"), a.Message))
	}
	return sb.String(), true
}

// ==========================================
// ASISTENTE AUTO-SUSTENTABLE GROQ
// ==========================================

func execBrowserOpenGroqConsole(ctx context.Context) (string, bool) {
	profile := browser.FindMatchingProfile("chrome", "")
	groqURL := "https://console.groq.com/keys"

	if profile != nil && profile.ExecPath != "" {
		if err := browser.LaunchWithProfile(ctx, profile, groqURL); err == nil {
			return fmt.Sprintf("🚀 Abriendo la consola de Groq Keys en Google Chrome (Perfil: %s | %s)...\n" +
				"👉 Pasos a seguir:\n" +
				"1. En la ventana que se abrió, haz clic en 'Continue with Google' para usar tu cuenta %s.\n" +
				"2. Haz clic en 'Create API Key', dale un nombre y presiona 'Copy'.\n" +
				"3. Una vez copiada, simplemente dime: 'Ozy, guarda mi clave de Groq' y la configuraré automáticamente desde tu portapapeles.",
				profile.Name, profile.Email, profile.Email), true
		}
	}

	// Fallback a apertura normal en SO
	_ = LaunchURLInOS(ctx, groqURL)
	return "🚀 Abriendo https://console.groq.com/keys en tu navegador...\nUna vez generada y copiada la clave, indícame 'guarda mi clave de Groq' para activarla automáticamente.", true
}

func execOSSetupGroqKey(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		ApiKey string `json:"api_key,omitempty"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	apiKey := strings.TrimSpace(params.ApiKey)
	// Si no se pasó explícitamente, intentar leer del portapapeles
	if apiKey == "" {
		clip, err := system.ReadClipboardContent()
		if err == nil && strings.HasPrefix(strings.TrimSpace(clip), "gsk_") {
			apiKey = strings.TrimSpace(clip)
		}
	}

	if apiKey == "" {
		return "No se proporcionó una clave ni se detectó una API key válida de Groq (iniciada con 'gsk_') en el portapapeles.\n" +
			"👉 Genera una en https://console.groq.com/keys, dale a Copiar y vuelve a pedirme que la guarde.", false
	}

	if !strings.HasPrefix(apiKey, "gsk_") {
		return fmt.Sprintf("La clave proporcionada ('%s...') no parece ser una clave válida de Groq (debe empezar con 'gsk_').", apiKey[:min(5, len(apiKey))]), false
	}

	// Validar la clave en vivo haciendo una consulta ultrarrápida a la API de Groq
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.groq.com/openai/v1/models", nil)
	if err != nil {
		return fmt.Sprintf("Error preparando validación: %v", err), false
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error conectando con la API de Groq: %v", err), false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("❌ La API Key de Groq fue rechazada por los servidores de Groq (HTTP %d). Verifica que la clave esté activa.", resp.StatusCode), false
	}

	// Guardar en el archivo .env
	envPath := ".env"
	envContent, _ := os.ReadFile(envPath)
	lines := strings.Split(string(envContent), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "GROQ_API_KEY=") {
			lines[i] = "GROQ_API_KEY=" + apiKey
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, "GROQ_API_KEY="+apiKey)
	}

	if err := os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Sprintf("Error guardando en archivo .env: %v", err), false
	}

	// Guardar en variable de entorno activa del proceso
	os.Setenv("GROQ_API_KEY", apiKey)

	return fmt.Sprintf("🎉 ¡ÉXITO TOTAL! API Key de Groq validada y guardada en .env.\n" +
		"• Motor de Transcripción STT: ACTIVADO (Groq Whisper ~150ms de latencia).\n" +
		"• Wake Word ('Hey Ozy'): Ya puedes hablarle directamente a Ozy por micrófono.\n" +
		"• Clave: %s...%s", apiKey[:7], apiKey[len(apiKey)-4:]), true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
