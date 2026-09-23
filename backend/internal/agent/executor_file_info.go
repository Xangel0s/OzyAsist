package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

// execOSFileInfo extrae metadatos precisos de un archivo (tamaño, fecha de creación y modificación de Windows).
func execOSFileInfo(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path   string `json:"path"`
		Target string `json:"target"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	target := strings.TrimSpace(params.Path)
	if target == "" {
		target = strings.TrimSpace(params.Target)
	}

	// Si no especificó ruta, resolver mediante el último archivo en foco de la conversación
	if target == "" {
		if focused, ok := GetFocusedFile(); ok && focused != "" {
			target = focused
		}
	}

	if target == "" {
		return "Error: debes especificar la ruta del archivo o informe a inspeccionar en el campo 'path'.", false
	}

	resolvedPath := system.ResolveUserPath(target)
	fi, err := os.Stat(resolvedPath)
	if err != nil {
		// Fallback inteligente: buscar en Documents o Desktop si solo se proporcionó el nombre base
		baseName := filepath.Base(target)
		docsDir := filepath.Join(os.Getenv("USERPROFILE"), "Documents")
		candDocs := filepath.Join(docsDir, baseName)
		if fiDocs, errDocs := os.Stat(candDocs); errDocs == nil {
			resolvedPath = candDocs
			fi = fiDocs
			err = nil
		} else {
			deskDir := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
			candDesk := filepath.Join(deskDir, baseName)
			if fiDesk, errDesk := os.Stat(candDesk); errDesk == nil {
				resolvedPath = candDesk
				fi = fiDesk
				err = nil
			}
		}
	}

	if err != nil {
		return fmt.Sprintf("No se encontró el archivo o carpeta '%s'. Verifica la ruta o búscalo con 'os_find_files'.", target), false
	}

	// Registrar como archivo en foco en la conversación
	SetFocusedFile(resolvedPath)

	modTime := fi.ModTime().Format("02/01/2006 15:04:05")
	createTime := modTime
	// Extraer fecha y hora de creación nativa Win32
	if d, ok := fi.Sys().(*syscall.Win32FileAttributeData); ok {
		createTime = time.Unix(0, d.CreationTime.Nanoseconds()).Format("02/01/2006 15:04:05")
	}

	sizeStr := formatFileSize(fi.Size())
	itemType := "Archivo"
	if fi.IsDir() {
		itemType = "Carpeta / Directorio"
		sizeStr = "N/A"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== INFORMACIÓN DE METADATOS: %s ===\n", fi.Name()))
	sb.WriteString(fmt.Sprintf("• Tipo: %s (%s)\n", itemType, filepath.Ext(resolvedPath)))
	sb.WriteString(fmt.Sprintf("• Ruta completa: %s\n", resolvedPath))
	sb.WriteString(fmt.Sprintf("• Tamaño: %s (%d bytes)\n", sizeStr, fi.Size()))
	sb.WriteString(fmt.Sprintf("• Fecha de creación: %s\n", createTime))
	sb.WriteString(fmt.Sprintf("• Última modificación: %s\n", modTime))

	return sb.String(), true
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
