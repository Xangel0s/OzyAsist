package agent

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
)

var (
	pyModuleRe   = regexp.MustCompile(`(?:ModuleNotFoundError:\s+No\s+module\s+named|No\s+module\s+named)\s+['"]?([a-zA-Z0-9_\-]+)['"]?`)
	nodeModuleRe = regexp.MustCompile(`Cannot\s+find\s+module\s+['"]([^'"]+)['"]`)
)

// mapPackageAlias mapea módulos importados de Python a sus nombres de paquete en PyPI
func mapPackageAlias(mod string) string {
	lower := strings.ToLower(mod)
	switch lower {
	case "pil":
		return "Pillow"
	case "yaml":
		return "pyyaml"
	case "bs4":
		return "beautifulsoup4"
	case "cv2":
		return "opencv-python"
	case "sklearn":
		return "scikit-learn"
	case "docx":
		return "python-docx"
	case "pptx":
		return "python-pptx"
	case "fitz":
		return "pymupdf"
	default:
		return mod
	}
}

// executeWithSelfHealing envuelve la ejecución de comandos de consola con detección y reparación autónoma de errores comunes (hasta 3 reintentos).
func executeWithSelfHealing(ctx context.Context, command, cwd string) (*CmdResult, string, error) {
	maxRetries := 3
	var healingLogs strings.Builder

	currentCmd := command

	for attempt := 1; attempt <= maxRetries; attempt++ {
		res, err := ExecuteShellInDir(ctx, "powershell", currentCmd, cwd)
		if err != nil {
			return nil, healingLogs.String(), err
		}

		if res.ExitCode == 0 {
			return res, healingLogs.String(), nil
		}

		// Si falló, analizar stderr y stdout para auto-reparación
		output := res.Stderr + "\n" + res.Stdout

		// Caso 1: Falta de módulo en Python
		if m := pyModuleRe.FindStringSubmatch(output); len(m) > 1 {
			rawMod := m[1]
			pkg := mapPackageAlias(rawMod)
			msg := fmt.Sprintf("\n[🛡️ Self-Healing Auto-Repair]: Se detectó que falta la librería de Python '%s' (%s). Instalando automáticamente con pip...\n", rawMod, pkg)
			healingLogs.WriteString(msg)
			log.Printf("[Self-Healing] Instalando dependencia de Python: %s", pkg)

			// Instalar la librería silenciosamente
			installCmd := fmt.Sprintf("pip install %s", pkg)
			installRes, _ := ExecuteShellInDir(ctx, "powershell", installCmd, cwd)
			if installRes != nil && installRes.ExitCode == 0 {
				healingLogs.WriteString(fmt.Sprintf("[🛡️ Self-Healing Auto-Repair]: ¡Librería '%s' instalada con éxito! Reejecutando comando original (Intento %d/%d)...\n", pkg, attempt+1, maxRetries))
				continue
			} else {
				healingLogs.WriteString(fmt.Sprintf("[🛡️ Self-Healing]: No se pudo instalar automáticamente '%s'.\n", pkg))
			}
		}

		// Caso 2: Restricción de ExecutionPolicy en PowerShell para scripts .ps1
		if strings.Contains(strings.ToLower(output), "cannot be loaded because running scripts is disabled") {
			healingLogs.WriteString("\n[🛡️ Self-Healing Auto-Repair]: Detectada restricción de scripts de PowerShell. Aplicando bypass temporal de ejecución...\n")
			currentCmd = "Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force; " + command
			continue
		}

		// Caso 3: Falta de módulo en Node.js
		if m := nodeModuleRe.FindStringSubmatch(output); len(m) > 1 {
			pkg := m[1]
			healingLogs.WriteString(fmt.Sprintf("\n[🛡️ Self-Healing Auto-Repair]: Se detectó falta del paquete de Node '%s'. Instalando con npm...\n", pkg))
			installCmd := fmt.Sprintf("npm install %s", pkg)
			installRes, _ := ExecuteShellInDir(ctx, "powershell", installCmd, cwd)
			if installRes != nil && installRes.ExitCode == 0 {
				healingLogs.WriteString(fmt.Sprintf("[🛡️ Self-Healing Auto-Repair]: ¡Paquete '%s' instalado! Reejecutando...\n", pkg))
				continue
			}
		}

		// Si no pudimos diagnosticar un patrón conocido de auto-reparación, retornar resultado actual
		return res, healingLogs.String(), nil
	}

	// Si agotó los reintentos
	finalRes, err := ExecuteShellInDir(ctx, "powershell", currentCmd, cwd)
	return finalRes, healingLogs.String(), err
}
