package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ozyassist/backend/internal/providers"
)

var autoDatasetMu sync.Mutex

// RecordInteractionForTraining guarda interacciones reales de alta fidelidad entre el usuario y Ozy
// para el auto-entrenamiento continuo y autónomo del modelo (Continuous Self-Improvement).
func RecordInteractionForTraining(userMsg, assistantMsg string, tools []providers.ToolCall) {
	cleanUser := strings.TrimSpace(userMsg)
	cleanAss := strings.TrimSpace(assistantMsg)
	if len(cleanUser) < 3 || len(cleanAss) < 3 {
		return
	}

	// Ignorar comandos de control de TUI
	if strings.HasPrefix(cleanUser, "/") {
		return
	}

	go func() {
		autoDatasetMu.Lock()
		defer autoDatasetMu.Unlock()

		type Msg struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		type Entry struct {
			Messages []Msg `json:"messages"`
		}

		var msgs []Msg
		msgs = append(msgs, Msg{Role: "user", Content: cleanUser})

		// Si hubo llamadas a herramientas, incluir sintaxis
		if len(tools) > 0 {
			var toolCallsStr []string
			for _, tc := range tools {
				toolCallsStr = append(toolCallsStr, string(tc.Input))
			}
			msgs = append(msgs, Msg{Role: "assistant", Content: cleanAss})
		} else {
			msgs = append(msgs, Msg{Role: "assistant", Content: cleanAss})
		}

		entry := Entry{Messages: msgs}
		line, err := json.Marshal(entry)
		if err != nil {
			return
		}

		// Registrar en localizaciones estándar
		targetPaths := []string{
			filepath.Join("data", "auto_dataset.jsonl"),
			filepath.Join("..", "training", "dataset", "auto_dataset.jsonl"),
		}

		for _, p := range targetPaths {
			dir := filepath.Dir(p)
			_ = os.MkdirAll(dir, 0755)
			f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				_, _ = f.Write(append(line, '\n'))
				_ = f.Close()
			}
		}
	}()
}
