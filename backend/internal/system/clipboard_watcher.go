package system

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type ClipboardEvent struct {
	Type            string    `json:"type"`             // "stack_trace", "error_log", "code_snippet", "url", "text"
	Content         string    `json:"content"`
	Preview         string    `json:"preview"`
	SuggestedAction string    `json:"suggested_action"` // "analyze_and_heal", "deep_search", "refactor"
	Timestamp       time.Time `json:"timestamp"`
}

type ClipboardBroadcaster interface {
	BroadcastJSON(event string, payload interface{})
}

type ClipboardWatcher struct {
	broadcaster ClipboardBroadcaster
	interval    time.Duration
	lastContent string
	running     bool
	stopCh      chan struct{}
	mu          sync.Mutex
}

func NewClipboardWatcher(broadcaster ClipboardBroadcaster, interval time.Duration) *ClipboardWatcher {
	if interval <= 0 {
		interval = 800 * time.Millisecond
	}
	return &ClipboardWatcher{
		broadcaster: broadcaster,
		interval:    interval,
	}
}

// Start inicia el vigilante proactivo en segundo plano
func (w *ClipboardWatcher) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-w.stopCh:
				return
			case <-ticker.C:
				w.checkClipboard()
			}
		}
	}()
	log.Printf("[Smart Clipboard] Vigilante de portapapeles iniciado (intervalo: %v)", w.interval)
}

// Stop detiene el vigilante
func (w *ClipboardWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return
	}
	w.running = false
	close(w.stopCh)
}

// ReadClipboardContent extrae el texto plano del portapapeles de Windows de forma ligera
func ReadClipboardContent() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Get-Clipboard")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

// WriteClipboardContent escribe texto en el portapapeles de Windows de forma segura
func WriteClipboardContent(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "$Input | Set-Clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// ClassifyContent clasifica el contenido copiado para determinar si requiere asistencia proactiva
func ClassifyContent(content string) (eventType string, action string, isRelevant bool) {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) < 10 {
		return "text", "", false
	}

	lower := strings.ToLower(trimmed)

	// 1. Detección de Stack Traces y Errores de Consola
	if strings.Contains(lower, "traceback (most recent call last)") ||
		strings.Contains(lower, "panic:") ||
		strings.Contains(lower, "error ts") ||
		strings.Contains(lower, "syntaxerror:") ||
		strings.Contains(lower, "nullpointerexception") ||
		strings.Contains(lower, "segfault") ||
		strings.Contains(lower, "fatal error:") {
		return "stack_trace", "analyze_and_heal", true
	}

	// 2. Detección de URLs para DeepSearch o resumen
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return "url", "deep_search", true
	}

	// 3. Detección de bloques de código estructurados
	if strings.Contains(trimmed, "func ") ||
		strings.Contains(trimmed, "def ") ||
		strings.Contains(trimmed, "export const ") ||
		strings.Contains(trimmed, "class ") ||
		strings.Contains(trimmed, "SELECT ") ||
		strings.Contains(trimmed, "CREATE TABLE ") {
		return "code_snippet", "explain_or_refactor", true
	}

	// 4. Detección de API Keys de Inteligencia Artificial (Groq, OpenAI)
	if strings.HasPrefix(trimmed, "gsk_") && len(trimmed) > 20 {
		return "api_key", "setup_groq_key", true
	}
	if strings.HasPrefix(trimmed, "sk-") && len(trimmed) > 20 {
		return "api_key", "setup_openai_key", true
	}

	return "text", "", false
}

func (w *ClipboardWatcher) checkClipboard() {
	content, err := ReadClipboardContent()
	if err != nil || content == "" {
		return
	}

	w.mu.Lock()
	if content == w.lastContent {
		w.mu.Unlock()
		return
	}
	w.lastContent = content
	w.mu.Unlock()

	eventType, action, isRelevant := ClassifyContent(content)
	if !isRelevant {
		return
	}

	preview := content
	if len(preview) > 120 {
		preview = preview[:120] + "..."
	}

	event := ClipboardEvent{
		Type:            eventType,
		Content:         content,
		Preview:         preview,
		SuggestedAction: action,
		Timestamp:       time.Now().UTC(),
	}

	if w.broadcaster != nil {
		w.broadcaster.BroadcastJSON("clipboard:smart_suggestion", event)
	}
}
