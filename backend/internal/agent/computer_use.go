package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/vision"
)

var (
	user32Lib        = syscall.NewLazyDLL("user32.dll")
	procSetCursorPos = user32Lib.NewProc("SetCursorPos")
	procMouseEvent   = user32Lib.NewProc("mouse_event")
)

const (
	mouseEventFLeftDown  = 0x0002
	mouseEventFLeftUp    = 0x0004
	mouseEventFRightDown = 0x0008
	mouseEventFRightUp   = 0x0010
)

// OSTakeScreenshot captura la pantalla completa de forma nativa
func OSTakeScreenshot(ctx context.Context) (string, error) {
	sc := vision.NewScreenCapture()
	res, err := sc.CapturePrimaryScreen(ctx)
	if err != nil {
		return "", fmt.Errorf("error capturando pantalla: %v", err)
	}

	dir := filepath.Join("data", "screenshots")
	_ = os.MkdirAll(dir, 0755)
	filename := fmt.Sprintf("screen_%d.png", time.Now().UnixMilli())
	fullPath := filepath.Join(dir, filename)

	if err := os.WriteFile(fullPath, res.ImagePNG, 0644); err != nil {
		return "", fmt.Errorf("error guardando captura de pantalla: %v", err)
	}

	absPath, _ := filepath.Abs(fullPath)
	return fmt.Sprintf("📸 === CAPTURA DE PANTALLA EXITOSA ===\n• Resolución: %dx%d píxeles\n• Archivo guardado: %s\n• Tamaño: %d KB\n• Fecha: %s",
		res.Width, res.Height, absPath, len(res.ImagePNG)/1024, res.CapturedAt.Format("15:04:05")), nil
}

// OSMouseClick desplaza el puntero y ejecuta un clic físico en coordenadas (x, y)
func OSMouseClick(ctx context.Context, x, y int, button string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("os_mouse_click solo está disponible en Windows")
	}

	button = strings.ToLower(strings.TrimSpace(button))
	if button == "" {
		button = "left"
	}

	// Mover cursor a coordenadas
	procSetCursorPos.Call(uintptr(x), uintptr(y))
	time.Sleep(40 * time.Millisecond)

	switch button {
	case "right":
		procMouseEvent.Call(mouseEventFRightDown, 0, 0, 0, 0)
		time.Sleep(25 * time.Millisecond)
		procMouseEvent.Call(mouseEventFRightUp, 0, 0, 0, 0)
	case "double", "double_click":
		procMouseEvent.Call(mouseEventFLeftDown, 0, 0, 0, 0)
		procMouseEvent.Call(mouseEventFLeftUp, 0, 0, 0, 0)
		time.Sleep(60 * time.Millisecond)
		procMouseEvent.Call(mouseEventFLeftDown, 0, 0, 0, 0)
		procMouseEvent.Call(mouseEventFLeftUp, 0, 0, 0, 0)
	default: // "left"
		procMouseEvent.Call(mouseEventFLeftDown, 0, 0, 0, 0)
		time.Sleep(25 * time.Millisecond)
		procMouseEvent.Call(mouseEventFLeftUp, 0, 0, 0, 0)
	}

	return fmt.Sprintf("🖱️ Clic (%s) ejecutado con éxito en coordenadas X=%d, Y=%d", button, x, y), nil
}

// OSTypeText simula escritura de texto o pulsación de teclas en la ventana enfocada
func OSTypeText(ctx context.Context, text string, pressEnter bool) (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("os_type_text solo está disponible en Windows")
	}
	if text == "" && !pressEnter {
		return "Ningún texto o tecla especificada", nil
	}

	// Escapar caracteres reservados de SendKeys (+ ^ % ~ { } [ ])
	var sb strings.Builder
	for _, ch := range text {
		switch ch {
		case '+', '^', '%', '~', '(', ')', '{', '}', '[', ']':
			sb.WriteString("{" + string(ch) + "}")
		case '\n':
			sb.WriteString("{ENTER}")
		case '\t':
			sb.WriteString("{TAB}")
		default:
			sb.WriteRune(ch)
		}
	}
	if pressEnter && !strings.HasSuffix(text, "\n") {
		sb.WriteString("{ENTER}")
	}

	escapedScript := strings.ReplaceAll(sb.String(), `"`, `""`)
	psScript := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.SendKeys]::SendWait("%s")
`, escapedScript)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("fallo al enviar pulsaciones de teclado: %v", err)
	}

	return fmt.Sprintf("⌨️ Texto ingresado en la ventana enfocada (%d caracteres).", len(text)), nil
}

func execOSTakeScreenshot(ctx context.Context) (string, bool) {
	out, err := OSTakeScreenshot(ctx)
	if err != nil {
		return fmt.Sprintf("Error tomando captura de pantalla: %v", err), false
	}
	return out, true
}

func execOSMouseClick(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Button string `json:"button"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_mouse_click: %v", err), false
	}

	out, err := OSMouseClick(ctx, params.X, params.Y, params.Button)
	if err != nil {
		return fmt.Sprintf("Error ejecutando clic de mouse: %v", err), false
	}
	return out, true
}

func execOSTypeText(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Text       string `json:"text"`
		PressEnter bool   `json:"press_enter"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_type_text: %v", err), false
	}

	out, err := OSTypeText(ctx, params.Text, params.PressEnter)
	if err != nil {
		return fmt.Sprintf("Error escribiendo texto: %v", err), false
	}
	return out, true
}
