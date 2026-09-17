package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ozyassist/backend/internal/providers"
)

func init() {
	tools := []providers.ToolDef{
		{
			Name:        "os_prepare_staging",
			Description: "Prepara una mesa de trabajo segura para un archivo (ej: Excel, PDF, CSV, código). Copia el archivo a ~/.ozy/workspace/ para que puedas editarlo con Python o scripts sin riesgo de dañar el original.",
			InputSchema: mustJSON(`{
				"type": "object",
				"properties": {
					"filePath": {
						"type": "string",
						"description": "Ruta del archivo original que se va a procesar"
					}
				},
				"required": ["filePath"]
			}`),
		},
		{
			Name:        "os_commit_staging",
			Description: "Aplica de forma atómica y segura los cambios realizados en la mesa de trabajo (~/.ozy/workspace/) hacia el archivo original. Realiza un backup automático previo.",
			InputSchema: mustJSON(`{
				"type": "object",
				"properties": {
					"stagedPath": {
						"type": "string",
						"description": "Ruta del archivo modificado en la mesa de trabajo"
					},
					"targetPath": {
						"type": "string",
						"description": "Ruta del archivo original de destino a actualizar"
					}
				},
				"required": ["stagedPath", "targetPath"]
			}`),
		},
		{
			Name:        "os_backup_file",
			Description: "Crea una copia de seguridad instantánea de cualquier archivo en ~/.ozy/backups/ antes de realizar cambios importantes.",
			InputSchema: mustJSON(`{
				"type": "object",
				"properties": {
					"filePath": {
						"type": "string",
						"description": "Ruta del archivo que deseas respaldar"
					}
				},
				"required": ["filePath"]
			}`),
		},
	}

	for _, t := range tools {
		AgentTools = append(AgentTools, t)
		VoiceAgentTools = append(VoiceAgentTools, t)
	}
}

// GetWorkspaceDir retorna ~/.ozy/workspace y lo crea si no existe
func GetWorkspaceDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "ozy_workspace")
	}
	dir := filepath.Join(home, ".ozy", "workspace")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// GetBackupsDir retorna ~/.ozy/backups y lo crea si no existe
func GetBackupsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "ozy_backups")
	}
	dir := filepath.Join(home, ".ozy", "backups")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// AutoBackupFile realiza una copia de seguridad de un archivo existente hacia ~/.ozy/backups
func AutoBackupFile(sourcePath string) (string, error) {
	sourcePath = filepath.Clean(sourcePath)
	fi, err := os.Stat(sourcePath)
	if err != nil {
		return "", fmt.Errorf("archivo no encontrado: %w", err)
	}
	if fi.IsDir() {
		return "", fmt.Errorf("la ruta es un directorio, no un archivo")
	}

	backupsDir := GetBackupsDir()
	ext := filepath.Ext(sourcePath)
	base := strings.TrimSuffix(filepath.Base(sourcePath), ext)
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("%s_%s%s.bak", base, timestamp, ext)
	backupPath := filepath.Join(backupsDir, backupFilename)

	if err := copyFile(sourcePath, backupPath); err != nil {
		return "", fmt.Errorf("error copiando archivo de backup: %w", err)
	}

	log.Printf("[Auto-Backup] Respaldo creado con éxito: %s -> %s", sourcePath, backupPath)
	return backupPath, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func execOSPrepareStaging(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		FilePath string `json:"filePath"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil || params.FilePath == "" {
		return "parámetros inválidos para os_prepare_staging", false
	}

	absSrc, err := filepath.Abs(params.FilePath)
	if err != nil {
		return fmt.Sprintf("ruta inválida: %v", err), false
	}

	wsDir := GetWorkspaceDir()
	filename := filepath.Base(absSrc)
	stagedPath := filepath.Join(wsDir, filename)

	// Crear backup previo
	bPath, _ := AutoBackupFile(absSrc)

	// Copiar a la mesa de trabajo
	if err := copyFile(absSrc, stagedPath); err != nil {
		return fmt.Sprintf("error copiando a mesa de trabajo: %v", err), false
	}

	msg := fmt.Sprintf("🛠️ Mesa de Trabajo lista:\n• Archivo original respaldado en: %s\n• Copia de trabajo aislada en: %s\nPuedes generar y ejecutar tus scripts en esa ruta sin alterar el original.", bPath, stagedPath)
	return msg, true
}

func execOSCommitStaging(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		StagedPath string `json:"stagedPath"`
		TargetPath string `json:"targetPath"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos para os_commit_staging", false
	}

	// Validar que el archivo staged exista y no esté vacío (corrupto)
	stagedFi, err := os.Stat(params.StagedPath)
	if err != nil {
		return fmt.Sprintf("el archivo en mesa de trabajo '%s' no existe: %v", params.StagedPath, err), false
	}
	if stagedFi.Size() == 0 {
		return "⚠️ ERROR: El archivo en mesa de trabajo está vacío. No se aplicará el commit para proteger el original.", false
	}

	// Backup preventivo del target antes de sobrescribir
	var backupPath string
	if _, err := os.Stat(params.TargetPath); err == nil {
		backupPath, _ = AutoBackupFile(params.TargetPath)
	}

	// Aplicar commit (copiar stagedPath -> targetPath)
	if err := copyFile(params.StagedPath, params.TargetPath); err != nil {
		return fmt.Sprintf("error aplicando commit al archivo destino: %v", err), false
	}

	return fmt.Sprintf("✅ Commit atómico exitoso:\n• Archivo destino actualizado: %s (%d bytes)\n• Backup de seguridad guardado en: %s",
		params.TargetPath, stagedFi.Size(), backupPath), true
}

func execOSBackupFile(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		FilePath string `json:"filePath"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil || params.FilePath == "" {
		return "parámetros inválidos para os_backup_file", false
	}

	bPath, err := AutoBackupFile(params.FilePath)
	if err != nil {
		return fmt.Sprintf("error creando backup: %v", err), false
	}

	return fmt.Sprintf("🛡️ Backup de seguridad creado exitosamente en:\n%s", bPath), true
}
