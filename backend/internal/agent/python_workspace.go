package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

func init() {
	tools := []providers.ToolDef{
		{
			Name:        "os_python_exec",
			Description: "Ejecuta un script o bloque de código Python 3 nativo dentro de la mesa de trabajo segura (~/.ozy/workspace/). Acceso completo a librerías de análisis y automatización (pandas, openpyxl, numpy, pillow, opencv, pymupdf, requests). Incluye auto-reparación automática con pip si falta alguna librería.",
			InputSchema: mustJSON(`{
				"type": "object",
				"properties": {
					"code": {
						"type": "string",
						"description": "Código fuente en Python 3 a ejecutar"
					},
					"scriptName": {
						"type": "string",
						"description": "Nombre del archivo script (opcional, ej: analisis_datos.py)"
					},
					"timeoutSeconds": {
						"type": "integer",
						"description": "Tiempo límite de ejecución en segundos (por defecto 60)"
					}
				},
				"required": ["code"]
			}`),
		},
		{
			Name:        "os_workspace_list",
			Description: "Lista todos los archivos generados o en proceso dentro de la mesa de trabajo segura (~/.ozy/workspace/), con detalles de tamaño y fecha.",
			InputSchema: mustJSON(`{"type":"object","properties":{}}`),
		},
	}

	for _, t := range tools {
		AgentTools = append(AgentTools, t)
		VoiceAgentTools = append(VoiceAgentTools, t)
	}
}

// FindPythonExecutable busca el binario de Python en el host
func FindPythonExecutable() string {
	candidates := []string{"python", "python3", "py"}
	for _, cand := range candidates {
		if p, err := exec.LookPath(cand); err == nil {
			return p
		}
	}
	// Fallback estándar en Windows
	defaultWinPath := `C:\Program Files\Python311\python.exe`
	if fi, err := os.Stat(defaultWinPath); err == nil && !fi.IsDir() {
		return defaultWinPath
	}
	return "python"
}

// ExecutePythonInWorkspace ejecuta código Python en la mesa de trabajo con soporte de Self-Healing
func ExecutePythonInWorkspace(ctx context.Context, code, scriptName string, timeoutSec int) (string, bool) {
	if strings.TrimSpace(code) == "" {
		return "❌ Error: Código Python vacío.", false
	}

	if timeoutSec <= 0 || timeoutSec > 300 {
		timeoutSec = 60
	}

	wsDir := GetWorkspaceDir()
	if scriptName == "" {
		scriptName = fmt.Sprintf("task_%s.py", time.Now().Format("20060102_150405"))
	} else if !strings.HasSuffix(strings.ToLower(scriptName), ".py") {
		scriptName += ".py"
	}

	scriptPath := filepath.Join(wsDir, scriptName)

	// Escribir el script en la mesa de trabajo
	if err := os.WriteFile(scriptPath, []byte(code), 0644); err != nil {
		return fmt.Sprintf("❌ Error guardando script en la mesa de trabajo: %v", err), false
	}

	pyExe := FindPythonExecutable()

	// Snapshot previo de archivos en la mesa de trabajo
	preFiles := getWorkspaceFileSet(wsDir)

	start := time.Now()
	maxAttempts := 3
	var finalOutput string
	var success bool

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		cmd := exec.CommandContext(execCtx, pyExe, scriptPath)
		cmd.Dir = wsDir
		cmd.Env = append(os.Environ(),
			"PYTHONIOENCODING=utf-8",
			"PYTHONUNBUFFERED=1",
			fmt.Sprintf("PYTHONPATH=%s", wsDir),
		)

		outBytes, err := cmd.CombinedOutput()
		cancel()

		output := strings.TrimSpace(string(outBytes))
		elapsed := time.Since(start).Round(time.Millisecond)

		if err == nil {
			success = true
			postFiles := getWorkspaceFileSet(wsDir)
			newFiles := diffFiles(preFiles, postFiles)

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("🐍 Python OK (%s) | Script: %s\n", elapsed, system.CleanCanonicalPath(scriptPath)))
			if len(newFiles) > 0 {
				sb.WriteString(fmt.Sprintf("• Archivos generados (%d):\n", len(newFiles)))
				for _, nf := range newFiles {
					sb.WriteString(fmt.Sprintf("   -> %s\n", system.CleanCanonicalPath(nf)))
				}
			}
			if output != "" {
				sb.WriteString("\n--- SALIDA (stdout/stderr) ---\n")
				sb.WriteString(output)
			} else {
				sb.WriteString("\n(Script completado sin salida por consola)")
			}
			finalOutput = sb.String()
			break
		}

		// Si falló, verificar si es ModuleNotFoundError y auto-reparar
		if strings.Contains(output, "ModuleNotFoundError:") || strings.Contains(output, "No module named ") {
			rawMod := extractMissingModule(output)
			if rawMod != "" && attempt < maxAttempts {
				pkgName := mapPackageAlias(rawMod)
				log.Printf("[Python-Workspace:Self-Healing] Módulo faltante '%s' (%s). Instalando intento %d...", rawMod, pkgName, attempt)
				installCmd := exec.CommandContext(ctx, pyExe, "-m", "pip", "install", pkgName)
				installCmd.Dir = wsDir
				_ = installCmd.Run()
				// Reintentar ejecución
				continue
			}
		}

		// Fallo no recuperable o se agotaron los intentos
		finalOutput = fmt.Sprintf("❌ Error de ejecución en Python (%s):\n• Script: %s\n\n--- SALIDA / ERROR ---\n%s",
			elapsed, scriptPath, output)
		success = false
		break
	}

	return finalOutput, success
}

func getWorkspaceFileSet(dir string) map[string]int64 {
	m := make(map[string]int64)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return m
	}
	for _, e := range entries {
		if !e.IsDir() {
			if fi, err := e.Info(); err == nil {
				m[e.Name()] = fi.Size()
			}
		}
	}
	return m
}

func diffFiles(pre, post map[string]int64) []string {
	var created []string
	for name, size := range post {
		oldSize, exists := pre[name]
		if !exists {
			created = append(created, fmt.Sprintf("%s (%d bytes)", name, size))
		} else if oldSize != size {
			created = append(created, fmt.Sprintf("%s (modificado: %d -> %d bytes)", name, oldSize, size))
		}
	}
	return created
}

func extractMissingModule(errOutput string) string {
	for _, line := range strings.Split(errOutput, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "No module named "); idx != -1 {
			target := line[idx+len("No module named "):]
			target = strings.Trim(target, "'\"")
			parts := strings.Split(target, ".")
			if len(parts) > 0 {
				return parts[0]
			}
			return target
		}
	}
	return ""
}

func execOSPythonExec(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Code           string `json:"code"`
		ScriptName     string `json:"scriptName"`
		TimeoutSeconds int    `json:"timeoutSeconds"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil || strings.TrimSpace(params.Code) == "" {
		return "❌ Parámetros inválidos: se requiere el campo 'code' con el código Python a ejecutar.", false
	}

	return ExecutePythonInWorkspace(ctx, params.Code, params.ScriptName, params.TimeoutSeconds)
}

func execOSWorkspaceList(ctx context.Context) (string, bool) {
	wsDir := GetWorkspaceDir()
	entries, err := os.ReadDir(wsDir)
	if err != nil {
		return fmt.Sprintf("Error leyendo mesa de trabajo: %v", err), false
	}

	cleanWsDir := system.CleanCanonicalPath(wsDir)
	if len(entries) == 0 {
		return fmt.Sprintf("Mesa de trabajo vacía en: %s", cleanWsDir), true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📁 Mesa de trabajo (~/.ozy/workspace - %s, %d elementos):\n", cleanWsDir, len(entries)))

	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if e.IsDir() {
			sb.WriteString(fmt.Sprintf("• [carpeta] %s/\n", e.Name()))
		} else {
			sb.WriteString(fmt.Sprintf("• [archivo] %s (%s | %s)\n", e.Name(), formatFileSize(info.Size()), info.ModTime().Format("2006-01-02")))
		}
	}

	return strings.TrimSpace(sb.String()), true
}
