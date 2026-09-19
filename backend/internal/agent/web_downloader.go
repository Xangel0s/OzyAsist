package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

// DownloadResult contiene el resumen de un archivo descargado
type DownloadResult struct {
	URL         string `json:"url"`
	Destination string `json:"destination"`
	SizeBytes   int64  `json:"size_bytes"`
	ContentType string `json:"content_type"`
}

// DownloadFile descarga un recurso HTTP/HTTPS directamente a disco de forma segura y eficiente
func DownloadFile(ctx context.Context, fileURL string, destPath string) (*DownloadResult, error) {
	fileURL = strings.TrimSpace(fileURL)
	if fileURL == "" {
		return nil, fmt.Errorf("URL de descarga vacía")
	}

	parsedURL, err := url.Parse(fileURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fmt.Errorf("URL inválida (debe iniciar con http o https): %s", fileURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error construyendo petición HTTP: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 OzyAssist/1.0")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{
		Timeout: 90 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error de conexión al descargar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("el servidor respondió con código de error HTTP %d (%s)", resp.StatusCode, resp.Status)
	}

	// Determinar nombre del archivo
	var suggestedName string

	// 1. Probar cabecera Content-Disposition
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
			if filename, ok := params["filename"]; ok && filename != "" {
				suggestedName = filepath.Base(filename)
			}
		}
	}

	// 2. Probar ruta de la URL
	if suggestedName == "" {
		urlPath := parsedURL.Path
		base := filepath.Base(urlPath)
		if base != "" && base != "." && base != "/" {
			suggestedName = base
		}
	}

	// 3. Fallback con timestamp y extensión por Content-Type
	if suggestedName == "" {
		ext := ".bin"
		cType := resp.Header.Get("Content-Type")
		if exts, _ := mime.ExtensionsByType(cType); len(exts) > 0 {
			ext = exts[0]
		}
		suggestedName = fmt.Sprintf("download_%d%s", time.Now().Unix(), ext)
	}

	// Resolver ruta destino final
	targetPath := strings.TrimSpace(destPath)
	if targetPath == "" {
		// Por defecto a la carpeta Descargas del usuario
		userProfile := os.Getenv("USERPROFILE")
		targetPath = filepath.Join(userProfile, "Downloads", suggestedName)
	} else {
		targetPath = system.ResolveUserPath(targetPath)
		// Si targetPath apunta a un directorio existente, concatenar el archivo sugerido
		if fi, err := os.Stat(targetPath); err == nil && fi.IsDir() {
			targetPath = filepath.Join(targetPath, suggestedName)
		} else if strings.HasSuffix(targetPath, "/") || strings.HasSuffix(targetPath, "\\") {
			targetPath = filepath.Join(targetPath, suggestedName)
		}
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return nil, fmt.Errorf("no se pudo crear directorio destino: %w", err)
	}

	outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear archivo en disco: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error escribiendo archivo descargado: %w", err)
	}

	return &DownloadResult{
		URL:         fileURL,
		Destination: targetPath,
		SizeBytes:   written,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func execOSDownloadFile(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		URL  string `json:"url"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_download_file: %v", err), false
	}

	res, err := DownloadFile(ctx, params.URL, params.Path)
	if err != nil {
		return fmt.Sprintf("Error descargando archivo: %v", err), false
	}

	sizeMB := float64(res.SizeBytes) / (1024 * 1024)
	return fmt.Sprintf("📥 === ARCHIVO DESCARGADO EXITOSAMENTE ===\n"+
		"• Origen:   %s\n"+
		"• Destino:  %s\n"+
		"• Tamaño:   %.2f MB (%d bytes)\n"+
		"• Tipo:     %s",
		res.URL, res.Destination, sizeMB, res.SizeBytes, res.ContentType), true
}
