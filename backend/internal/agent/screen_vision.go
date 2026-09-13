package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
	"github.com/ozyassist/backend/internal/vision"
)

type AnalyzeScreenParams struct {
	Question string `json:"question,omitempty"` // Pregunta sobre la pantalla (ej: '¿qué error se muestra en pantalla?')
}

// AnalyzeScreen captura la pantalla actual y la analiza con un modelo multimodal.
func AnalyzeScreen(ctx context.Context, question string) (string, error) {
	if question == "" {
		question = "Describe qué ventanas y contenido están visibles en esta pantalla. Indica si hay errores, alertas o detalles relevantes."
	}

	sc := vision.NewScreenCapture()
	res, err := sc.CapturePrimaryScreen(ctx)
	if err != nil {
		return "", fmt.Errorf("no se pudo capturar la pantalla: %v", err)
	}

	// Guardar en disco para trazabilidad
	dir := filepath.Join("data", "screenshots")
	_ = os.MkdirAll(dir, 0755)
	filename := fmt.Sprintf("vision_%d.png", time.Now().UnixMilli())
	savedPath := filepath.Join(dir, filename)
	_ = os.WriteFile(savedPath, res.ImagePNG, 0644)
	absPath, _ := filepath.Abs(savedPath)

	b64Image := base64.StdEncoding.EncodeToString(res.ImagePNG)
	dataURL := "data:image/png;base64," + b64Image

	// Determinar proveedor multimodal disponible
	apiKey, baseURL, model := detectVisionProvider()
	if apiKey == "" && baseURL == "" {
		// Modo fallback: reportar metadatos y ventanas activas si no hay API multimodal configurada
		nav := system.NewWindowsNavigator()
		windows, _ := nav.GetActiveWindows(ctx)
		var topWin []string
		for i, w := range windows {
			if i >= 5 {
				break
			}
			topWin = append(topWin, fmt.Sprintf("- Handle %d: %s (PID %d)", w.Handle, w.Title, w.ProcessID))
		}
		return fmt.Sprintf("📸 === CAPTURA TOMADA (MODO TEXTO) ===\n"+
			"• Archivo: %s\n"+
			"• Resolución: %dx%d píxeles\n\n"+
			"💡 Para habilitar análisis visual por IA, configura una clave de OpenRouter o OpenAI (`/key openrouter <tu-clave>`).\n"+
			"Ventanas activas detectadas en pantalla:\n%s", absPath, res.Width, res.Height, stringsJoin(topWin, "\n")), nil
	}

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": question},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": dataURL,
						},
					},
				},
			},
		},
		"max_tokens": 1000,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error serializando petición de visión: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("error creando request de visión: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error conectando con el modelo de visión: %v", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error leyendo respuesta de visión: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("📸 Captura guardada en: %s (%dx%d px)\nAviso visión (HTTP %d): %s", absPath, res.Width, res.Height, resp.StatusCode, string(respBytes)), nil
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil || len(parsed.Choices) == 0 {
		return fmt.Sprintf("📸 Captura guardada en: %s (%dx%d px)\nNo se pudo parsear el análisis del modelo.", absPath, res.Width, res.Height), nil
	}

	analysis := parsed.Choices[0].Message.Content

	return fmt.Sprintf("👁️ === ANÁLISIS VISUAL DE PANTALLA ===\n"+
		"• Archivo:     %s\n"+
		"• Resolución:  %dx%d px\n"+
		"• Modelo IA:   %s\n\n"+
		"🔎 DIAGNÓSTICO VISUAL:\n%s", absPath, res.Width, res.Height, model, analysis), nil
}

func execOSAnalyzeScreen(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params AnalyzeScreenParams
	if len(tc.Input) > 0 {
		_ = json.Unmarshal(tc.Input, &params)
	}

	out, err := AnalyzeScreen(ctx, params.Question)
	if err != nil {
		return fmt.Sprintf("Error en os_analyze_screen: %v", err), false
	}

	return out, true
}

func detectVisionProvider() (apiKey, baseURL, model string) {
	// 1. OpenRouter (suele tener modelos visión gratuitos o de alta velocidad)
	orKey := os.Getenv("OPENROUTER_API_KEY")
	if orKey != "" {
		return orKey, "https://openrouter.ai/api/v1", "google/gemini-2.0-flash-exp:free"
	}

	// 2. OpenAI
	oaKey := os.Getenv("OPENAI_API_KEY")
	if oaKey != "" {
		return oaKey, "https://api.openai.com/v1", "gpt-4o-mini"
	}

	// 3. LM Studio local (si está iniciado en puerto 1234)
	if checkPortOpen("localhost:1234") {
		return "", "http://localhost:1234/v1", "local-vision"
	}

	// 4. Ollama local
	if checkPortOpen("localhost:11434") {
		return "", "http://localhost:11434/v1", "llama3.2-vision"
	}

	return "", "", ""
}

func stringsJoin(items []string, sep string) string {
	res := ""
	for i, item := range items {
		if i > 0 {
			res += sep
		}
		res += item
	}
	return res
}

func checkPortOpen(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

