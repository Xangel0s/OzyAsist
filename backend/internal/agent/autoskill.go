package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ozyassist/backend/internal/providers"
)

func EvaluateAndSaveSkill(provider providers.Provider, chatHistory []providers.Message, userPrompt string) {
	if provider == nil || len(chatHistory) < 3 {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("AutoSkill panic recuperado: %v", r)
			}
		}()

		log.Println("Iniciando evaluación de Auto-Skill en segundo plano...")

		var sb strings.Builder
		for _, m := range chatHistory {
			if m.Role == "user" || m.Role == "assistant" || m.Role == "tool" {
				if m.ToolResult != nil {
					sb.WriteString(fmt.Sprintf("[%s]: Resultado: %s\n", m.Role, m.ToolResult.Content))
				} else {
					sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, m.Content))
				}
			}
		}

		prompt := "Eres el Evaluador de Aprendizaje de OzyAssist. Tu trabajo es analizar la siguiente conversación completada y determinar si el agente resolvió un problema complejo (ej. configurar servidor, Python, Excel).\n\nConversación:\n" + sb.String() + "\n\nReglas:\n1. Si la tarea fue muy simple, responde únicamente NO_SKILL.\n2. Si fue compleja y tuvo éxito, responde extrayendo la habilidad en formato Markdown puro empezando con '# Nombre Habilidad', sin bloques de código."

		messages := []providers.Message{
			{Role: "user", Content: prompt},
		}

		ctx := context.Background()
		chunkCh, err := provider.StreamCompletion(ctx, messages, providers.CompletionOptions{
			Temperature: 0.1,
			MaxTokens:   800,
			Stream:      false,
		})
		if err != nil {
			log.Printf("Error evaluando Auto-Skill: %v", err)
			return
		}

		var result string
		for chunk := range chunkCh {
			if chunk.Type == "text" {
				result += chunk.Content
			}
		}

		result = strings.TrimSpace(result)
		if result == "NO_SKILL" || result == "" || strings.HasPrefix(result, "NO_SKILL") {
			log.Println("AutoSkill: No se detectó habilidad compleja.")
			return
		}

		result = strings.TrimPrefix(result, "```markdown")
		result = strings.TrimPrefix(result, "```")
		result = strings.TrimSuffix(result, "```")
		result = strings.TrimSpace(result)

		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		skillsDir := filepath.Join(home, ".ozy", "skills")
		os.MkdirAll(skillsDir, 0755)

		filename := fmt.Sprintf("skill_%d.md", time.Now().Unix())
		filePath := filepath.Join(skillsDir, filename)

		err = os.WriteFile(filePath, []byte(result), 0644)
		if err != nil {
			log.Printf("Error guardando Auto-Skill en %s: %v", filePath, err)
			return
		}

		log.Printf("¡Nueva Auto-Skill aprendida y guardada en %s!", filePath)
	}()
}

func LoadAutoSkills() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	skillsDir := filepath.Join(home, ".ozy", "skills")
	
	files, err := os.ReadDir(skillsDir)
	if err != nil || len(files) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n=== HABILIDADES APRENDIDAS PREVIAMENTE (AUTO-SKILLS) ===\n")
	sb.WriteString("El sistema ha aprendido flujos de trabajo complejos y guardó la siguiente memoria procedimental. Úsalos como guía cuando se te pida realizar tareas similares:\n\n")

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			content, err := os.ReadFile(filepath.Join(skillsDir, f.Name()))
			if err == nil {
				sb.WriteString(string(content))
				sb.WriteString("\n---\n")
			}
		}
	}
	return sb.String()
}
