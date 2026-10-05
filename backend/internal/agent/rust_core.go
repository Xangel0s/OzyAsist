package agent

import (
	_ "embed"
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
)

//go:embed assets/ozy-core.exe
var embeddedOzyCore []byte

func init() {

	tools := []providers.ToolDef{
		{
			Name:        "os_camera_capture",
			Description: "Captura una fotografía en vivo desde la cámara web de Windows usando el motor nativo en Rust (MediaFoundation) sin latencia ni dependencias CGO. Guarda la imagen en la mesa de trabajo o ruta especificada.",
			InputSchema: mustJSON(`{
				"type": "object",
				"properties": {
					"outputPath": {
						"type": "string",
						"description": "Ruta donde guardar la foto (ej: ~/.ozy/workspace/camara.jpg). Si se omite, se guarda en ~/.ozy/workspace/camera_snapshot.jpg"
					},
					"deviceIndex": {
						"type": "integer",
						"description": "Índice de la cámara física a usar (por defecto 0)"
					}
				}
			}`),
		},
		{
			Name:        "os_camera_list",
			Description: "Enumera todas las cámaras web físicas conectadas al equipo con sus índices y nombres usando el motor nativo en Rust.",
			InputSchema: mustJSON(`{"type":"object","properties":{}}`),
		},
		{
			Name:        "os_skeletonize",
			Description: "Extrae el esqueleto estructural y firmas de un archivo de código fuente (.go, .rs, .ts, .py, etc.) ahorrando hasta un 95% de tokens de contexto.",
			InputSchema: mustJSON(`{
				"type": "object",
				"properties": {
					"path": {
						"type": "string",
						"description": "Ruta absoluta o relativa del archivo de código a esqueletonizar"
					}
				},
				"required": ["path"]
			}`),
		},
		{
			Name:        "os_inspect_active_ui",
			Description: "Inspecciona el árbol de controles interactivos (botones, campos de texto, pestañas, diálogos) de la ventana activa en pantalla en 1 ms con CERO tokens de visión multimodal.",
			InputSchema: mustJSON(`{"type":"object","properties":{}}`),
		},
	}

	for _, t := range tools {
		AgentTools = append(AgentTools, t)
		VoiceAgentTools = append(VoiceAgentTools, t)
	}
}

// FindRustCoreBinary busca el binario ozy-core.exe en el sistema o lo auto-extrae desde los assets embebidos
func FindRustCoreBinary() (string, error) {
	// 1. Variable de entorno explícita
	if envPath := os.Getenv("OZY_CORE_PATH"); envPath != "" {
		if fi, err := os.Stat(envPath); err == nil && !fi.IsDir() {
			return envPath, nil
		}
	}

	// 2. Comprobar en ~/.ozy/bin/ozy-core.exe (directorio local del usuario)
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		userProfile = os.Getenv("HOME")
	}
	userBinPath := filepath.Join(userProfile, ".ozy", "bin", "ozy-core.exe")
	if fi, err := os.Stat(userBinPath); err == nil && !fi.IsDir() && fi.Size() > 0 {
		return userBinPath, nil
	}

	// 3. Travesía hacia arriba desde CWD para encontrar la raíz del repo o backend
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for i := 0; i < 5; i++ {
			checkPaths := []string{
				filepath.Join(dir, "bin", "ozy-core.exe"),
				filepath.Join(dir, "backend", "bin", "ozy-core.exe"),
				filepath.Join(dir, "ozy-core", "target", "release", "ozy-core.exe"),
				filepath.Join(dir, "backend", "ozy-core", "target", "release", "ozy-core.exe"),
				filepath.Join(dir, "ozy-core.exe"),
			}
			for _, cp := range checkPaths {
				if fi, err := os.Stat(cp); err == nil && !fi.IsDir() {
					abs, err := filepath.Abs(cp)
					if err == nil {
						return abs, nil
					}
					return cp, nil
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 4. Buscar en el PATH del sistema
	if p, err := exec.LookPath("ozy-core.exe"); err == nil {
		return p, nil
	}

	// 5. Fallback Zero-Docker autónomo: auto-extraer el binario embebido a ~/.ozy/bin/ozy-core.exe
	if len(embeddedOzyCore) > 0 {
		if err := os.MkdirAll(filepath.Dir(userBinPath), 0755); err == nil {
			if err := os.WriteFile(userBinPath, embeddedOzyCore, 0755); err == nil {
				log.Printf("✓ Binario nativo ozy-core auto-extraído en: %s (%d bytes)", userBinPath, len(embeddedOzyCore))
				return userBinPath, nil
			}
		}
	}

	return "", fmt.Errorf("binario ozy-core.exe no encontrado en backend/bin ni ~/.ozy/bin")
}

// CallRustCameraCapture ejecuta la captura de cámara en ozy-core
func CallRustCameraCapture(ctx context.Context, outputPath string, deviceIndex int) (string, error) {
	bin, err := FindRustCoreBinary()
	if err != nil {
		return "", err
	}

	if outputPath == "" {
		ws := GetWorkspaceDir()
		outputPath = filepath.Join(ws, fmt.Sprintf("camera_snapshot_%s.jpg", time.Now().Format("20060102_150405")))
	} else if strings.HasPrefix(outputPath, "~/") || strings.HasPrefix(outputPath, "~\\") {
		home, _ := os.UserHomeDir()
		outputPath = filepath.Join(home, outputPath[2:])
	}

	// Asegurar directorio padre
	_ = os.MkdirAll(filepath.Dir(outputPath), 0755)

	cmd := exec.CommandContext(ctx, bin, "camera-capture", "--output", outputPath, "--device", fmt.Sprintf("%d", deviceIndex))
	outBytes, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(outBytes))

	if err != nil {
		return "", fmt.Errorf("fallo ejecutando ozy-core: %v (Salida: %s)", err, outStr)
	}

	return outStr, nil
}

// CallRustCameraList lista las cámaras detectadas mediante ozy-core
func CallRustCameraList(ctx context.Context) (string, error) {
	bin, err := FindRustCoreBinary()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, bin, "camera-list")
	outBytes, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(outBytes))

	if err != nil {
		return "", fmt.Errorf("fallo ejecutando ozy-core: %v (Salida: %s)", err, outStr)
	}

	return outStr, nil
}

func execOSCameraCapture(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		OutputPath  string `json:"outputPath"`
		DeviceIndex int    `json:"deviceIndex"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	msg, err := CallRustCameraCapture(ctx, params.OutputPath, params.DeviceIndex)
	if err != nil {
		log.Printf("[Rust-Core:Camera] Error: %v", err)
		return fmt.Sprintf("❌ Error al capturar cámara web: %v", err), false
	}

	return msg, true
}

func execOSCameraList(ctx context.Context, tc providers.ToolCall) (string, bool) {
	msg, err := CallRustCameraList(ctx)
	if err != nil {
		log.Printf("[Rust-Core:CameraList] Error: %v", err)
		return fmt.Sprintf("❌ Error listando cámaras web: %v", err), false
	}

	return msg, true
}

// CallRustSkeletonize extrae el esqueleto estructural y firmas de un archivo de código usando ozy-core en Rust
func CallRustSkeletonize(ctx context.Context, filePath string) (string, error) {
	bin, err := FindRustCoreBinary()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, bin, "skeletonize", filePath)
	outBytes, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(outBytes))

	if err != nil {
		return "", fmt.Errorf("fallo ejecutando ozy-core skeletonize: %v (Salida: %s)", err, outStr)
	}

	return outStr, nil
}

// CallRustInspectUI inspecciona la UI de la ventana activa mediante ozy-core en Rust
func CallRustInspectUI(ctx context.Context) (string, error) {
	bin, err := FindRustCoreBinary()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, bin, "inspect-ui")
	outBytes, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(outBytes))

	if err != nil {
		return "", fmt.Errorf("fallo ejecutando ozy-core inspect-ui: %v (Salida: %s)", err, outStr)
	}

	return outStr, nil
}

func execOSSkeletonize(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	if params.Path == "" {
		return "❌ Falta el parámetro 'path'", false
	}

	res, err := CallRustSkeletonize(ctx, params.Path)
	if err != nil {
		log.Printf("[Rust-Core:Skeletonize] Error: %v", err)
		return fmt.Sprintf("❌ Error esqueletonizando código: %v", err), false
	}

	return res, true
}

func execOSInspectActiveUI(ctx context.Context, tc providers.ToolCall) (string, bool) {
	res, err := CallRustInspectUI(ctx)
	if err != nil {
		log.Printf("[Rust-Core:InspectUI] Error: %v", err)
		return fmt.Sprintf("❌ Error inspeccionando interfaz: %v", err), false
	}

	return res, true
}
