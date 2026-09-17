package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ozyassist/backend/internal/providers"
)

func init() {
	// Registrar herramienta en AgentTools y VoiceAgentTools
	toolDef := providers.ToolDef{
		Name:        "os_save_custom_tool",
		Description: "Guarda un script reusable (Python o PowerShell) en la fábrica de herramientas (~/.ozy/tools). Úsalo cuando crees una automatización útil para tenerla disponible permanentemente.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"description": "Nombre de la herramienta o script (ej: 'convert_pdf_word.py', 'clean_temp.ps1')"
				},
				"description": {
					"type": "string",
					"description": "Breve explicación de qué hace y cómo se usa"
				},
				"code": {
					"type": "string",
					"description": "Código completo del script"
				}
			},
			"required": ["name", "code"]
		}`),
	}

	AgentTools = append(AgentTools, toolDef)
	VoiceAgentTools = append(VoiceAgentTools, toolDef)
}

// GetToolsDir retorna la ruta de ~/.ozy/tools y la crea si no existe
func GetToolsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".ozy", "tools")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// LoadCustomTools lista las herramientas guardadas en ~/.ozy/tools para inyectarlas en el prompt
func LoadCustomTools() string {
	toolsDir := GetToolsDir()
	if toolsDir == "" {
		return ""
	}

	entries, err := os.ReadDir(toolsDir)
	if err != nil || len(entries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n=== HERRAMIENTAS PERSONALIZADAS EN ~/.ozy/tools ===\n")
	sb.WriteString("Tienes estos scripts disponibles creados en sesiones anteriores. Puedes ejecutarlos con 'os_run_command':\n")

	for _, e := range entries {
		if !e.IsDir() {
			sb.WriteString(fmt.Sprintf("- %s (Ruta: %s)\n", e.Name(), filepath.Join(toolsDir, e.Name())))
		}
	}

	return sb.String()
}

func execOSSaveCustomTool(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Code        string `json:"code"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos para os_save_custom_tool", false
	}

	toolsDir := GetToolsDir()
	if toolsDir == "" {
		return "no se pudo acceder a ~/.ozy/tools", false
	}

	filename := filepath.Base(params.Name)
	if !strings.HasSuffix(filename, ".py") && !strings.HasSuffix(filename, ".ps1") && !strings.HasSuffix(filename, ".bat") {
		filename += ".py"
	}

	targetPath := filepath.Join(toolsDir, filename)
	header := ""
	if params.Description != "" {
		header = fmt.Sprintf("# Descripción: %s\n\n", params.Description)
	}

	err := os.WriteFile(targetPath, []byte(header+params.Code), 0755)
	if err != nil {
		return fmt.Sprintf("error guardando herramienta: %v", err), false
	}

	log.Printf("[ToolFactory] Nueva herramienta registrada: %s", targetPath)
	return fmt.Sprintf("✅ Herramienta guardada con éxito en: %s. Puedes invocarla con 'os_run_command' cuando la necesites.", targetPath), true
}
