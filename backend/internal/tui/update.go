package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/ozyassist/backend/internal/agent"
	"github.com/ozyassist/backend/internal/browser"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/mcp"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width - 4)

		headerH := lipgloss.Height(m.renderHeader())
		footerH := lipgloss.Height(m.renderFooter())
		// Exact calculation: 1 newline after header, 1 newline after viewport
		vpHeight := msg.Height - headerH - footerH - 2
		if vpHeight < 4 {
			vpHeight = 4
		}

		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.viewport.SetContent(m.renderConversation())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
			m.viewport.SetContent(m.renderConversation())
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.state != StateIdle {
				if m.loopCancel != nil {
					m.loopCancel()
				}
				if m.loopSessionID != "" {
					agent.CancelSession(m.loopSessionID)
				}
				m.loopCancel = nil
				m.loopSessionID = ""
				m.state = StateIdle
				m.systemStatus = "Acción cancelada con tecla [Esc]"
				msgText := "⚠️ Operación interrumpida con la tecla [Esc]."
				if len(m.messageQueue) > 0 {
					msgText += fmt.Sprintf(" (Quedan %d mensajes en cola. Usa /queue para verlos o /clearqueue para descartarlos).", len(m.messageQueue))
				}
				m.entries = append(m.entries, ChatEntry{
					Role:    "system",
					Content: msgText,
				})
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				return m, nil
			}

		case tea.KeyCtrlC:
			if m.state != StateIdle {
				if m.loopCancel != nil {
					m.loopCancel()
				}
				if m.loopSessionID != "" {
					agent.CancelSession(m.loopSessionID)
				}
				m.loopCancel = nil
				m.loopSessionID = ""
				m.state = StateIdle
				m.systemStatus = "Acción cancelada por el usuario"
				msgText := "⚠️ Operación interrumpida con Ctrl+C."
				if len(m.messageQueue) > 0 {
					msgText += fmt.Sprintf(" (Quedan %d mensajes en cola. Usa /queue o /clearqueue).", len(m.messageQueue))
				}
				m.entries = append(m.entries, ChatEntry{
					Role:    "system",
					Content: msgText,
				})
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				return m, nil
			}
			return m, tea.Quit

		case tea.KeyCtrlL:
			m.entries = nil
			m.viewport.SetContent(m.renderConversation())
			return m, nil

		case tea.KeyCtrlT:
			m.showThinking = !m.showThinking
			if m.showThinking {
				m.systemStatus = "💭 Hilo de pensamiento: VISIBLE (Presiona Ctrl+T para ocultar)"
			} else {
				m.systemStatus = "💭 Hilo de pensamiento: PLEGADO (Presiona Ctrl+T para desplegar)"
			}
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, nil

		case tea.KeyEnter:
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}

			// Manejo de Comandos Slash prioritarios (disponibles siempre)
			if strings.HasPrefix(input, "/") {
				parts := strings.Fields(input)
				slashCmd := strings.ToLower(parts[0])

				// 1. Cancelación inmediata (/cancel, /stop, /abort, /cancelar, /parar)
				if slashCmd == "/cancel" || slashCmd == "/stop" || slashCmd == "/abort" || slashCmd == "/cancelar" || slashCmd == "/parar" {
					m.textarea.Reset()
					clearAll := len(parts) > 1 && strings.ToLower(parts[1]) == "all"
					if m.state != StateIdle {
						if m.loopCancel != nil {
							m.loopCancel()
						}
						if m.loopSessionID != "" {
							agent.CancelSession(m.loopSessionID)
						}
						m.loopCancel = nil
						m.loopSessionID = ""
						m.state = StateIdle
						m.systemStatus = "Petición cancelada por el usuario"
						msgText := "⚠️ Petición cancelada con éxito."
						if clearAll {
							discarded := m.ClearQueue()
							msgText += fmt.Sprintf(" Y se descartaron %d mensajes en cola.", discarded)
						} else if len(m.messageQueue) > 0 {
							msgText += fmt.Sprintf(" (Quedan %d mensajes en cola. Usa /queue para verlos o /clearqueue para vaciarla).", len(m.messageQueue))
						}
						m.entries = append(m.entries, ChatEntry{
							Role:    "system",
							Content: msgText,
						})
					} else {
						if clearAll || len(m.messageQueue) > 0 {
							discarded := m.ClearQueue()
							m.entries = append(m.entries, ChatEntry{
								Role:    "system",
								Content: fmt.Sprintf("🗑️ Se vació la cola de mensajes (%d descartados).", discarded),
							})
						} else {
							m.entries = append(m.entries, ChatEntry{
								Role:    "system",
								Content: "ℹ️ No hay ninguna petición activa ni mensajes en cola.",
							})
						}
					}
					m.viewport.SetContent(m.renderConversation())
					m.viewport.GotoBottom()
					return m, nil
				}

				// 2. Envío directo / steer (/now, /steer, /send, /ya, /direct)
				if slashCmd == "/now" || slashCmd == "/steer" || slashCmd == "/send" || slashCmd == "/ya" || slashCmd == "/direct" {
					m.textarea.Reset()
					if len(parts) < 2 {
						m.entries = append(m.entries, ChatEntry{
							Role:    "system",
							Content: "ℹ️ Uso: /now <orden> — Cancela la tarea actual y ejecuta la nueva orden inmediatamente.",
						})
						m.viewport.SetContent(m.renderConversation())
						m.viewport.GotoBottom()
						return m, nil
					}
					directPrompt := strings.TrimSpace(strings.TrimPrefix(input, parts[0]))
					if m.state != StateIdle {
						if m.loopCancel != nil {
							m.loopCancel()
						}
						if m.loopSessionID != "" {
							agent.CancelSession(m.loopSessionID)
						}
						m.loopCancel = nil
						m.loopSessionID = ""
						m.entries = append(m.entries, ChatEntry{
							Role:    "system",
							Content: "⚡ Tarea anterior interrumpida. Ejecutando nueva orden directamente...",
						})
					}
					m.promptHistory = append(m.promptHistory, directPrompt)
					m.historyIndex = len(m.promptHistory)
					m.entries = append(m.entries, ChatEntry{
						Role:    "user",
						Content: directPrompt,
					})
					m.state = StateThinking
					m.systemStatus = "Procesando razonamiento agéntico..."
					m.currentStream = ""
					m.viewport.SetContent(m.renderConversation())
					m.viewport.GotoBottom()
					return m, m.startAgentTurn(directPrompt)
				}

				// 3. Inspección de cola (/queue, /cola)
				if slashCmd == "/queue" || slashCmd == "/cola" {
					m.textarea.Reset()
					if len(m.messageQueue) == 0 {
						m.entries = append(m.entries, ChatEntry{
							Role:    "system",
							Content: "ℹ️ La cola de mensajes está vacía.",
						})
					} else {
						var sb strings.Builder
						sb.WriteString(fmt.Sprintf("📥 Mensajes en cola de espera (%d):\n", len(m.messageQueue)))
						for i, q := range m.messageQueue {
							vTag := ""
							if q.IsVoice {
								vTag = "🎙️ [Voz] "
							}
							sb.WriteString(fmt.Sprintf("  %d. %s\"%s\"\n", i+1, vTag, q.Prompt))
						}
						sb.WriteString("💡 Usa /clearqueue para vaciarla, /now <orden> para ejecutar de inmediato o /cancel para detener la tarea activa.")
						m.entries = append(m.entries, ChatEntry{
							Role:    "system",
							Content: sb.String(),
						})
					}
					m.viewport.SetContent(m.renderConversation())
					m.viewport.GotoBottom()
					return m, nil
				}

				// 4. Vaciar la cola (/clearqueue, /dropqueue, /vaciarcola)
				if slashCmd == "/clearqueue" || slashCmd == "/dropqueue" || slashCmd == "/vaciarcola" {
					m.textarea.Reset()
					discarded := m.ClearQueue()
					m.entries = append(m.entries, ChatEntry{
						Role:    "system",
						Content: fmt.Sprintf("🗑️ Cola de mensajes descartada con éxito (%d mensajes eliminados).", discarded),
					})
					m.viewport.SetContent(m.renderConversation())
					m.viewport.GotoBottom()
					return m, nil
				}

				// 5. Otros comandos slash (/help, /provider, /key, /tools, etc.)
				cmdOutput := m.handleSlashCommand(input)
				m.textarea.Reset()
				if cmdOutput == "QUIT" {
					return m, tea.Quit
				}
				if cmdOutput != "" {
					m.entries = append(m.entries, ChatEntry{
						Role:    "system",
						Content: cmdOutput,
					})
					m.viewport.SetContent(m.renderConversation())
					m.viewport.GotoBottom()
				}
				return m, nil
			}

			// Si el agente está ocupado y no es un comando slash, encolar el mensaje
			if m.state != StateIdle {
				m.promptHistory = append(m.promptHistory, input)
				m.historyIndex = len(m.promptHistory)
				qPos := m.EnqueuePrompt(input, false)
				m.textarea.Reset()
				m.entries = append(m.entries, ChatEntry{
					Role:    "system",
					Content: fmt.Sprintf("📥 Mensaje añadido a la cola [#%d]: \"%s\"\n(Se ejecutará automáticamente al finalizar la tarea actual. Usa /now <orden> para ejecutar de inmediato o /cancel para cancelar la actual).", qPos, input),
				})
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				return m, nil
			}

			// Registro de historial y entrada normal de usuario (StateIdle)
			m.promptHistory = append(m.promptHistory, input)
			m.historyIndex = len(m.promptHistory)
			m.entries = append(m.entries, ChatEntry{
				Role:    "user",
				Content: input,
			})
			m.textarea.Reset()
			m.state = StateThinking
			m.systemStatus = "Procesando razonamiento agéntico..."
			m.currentStream = ""

			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()

			// Iniciar Agent Loop
			return m, m.startAgentTurn(input)
		}

	case agentEventMsg:
		evt := agent.AgentEvent(msg)
		switch evt.Type {
		case "message:thinking", "agent:thinking":
			m.currentThinking += evt.Content
			if m.showThinking {
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
			}

		case "message:delta":
			m.state = StateStreaming
			m.currentStream += evt.Content
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()

		case "tool:call":
			m.state = StateExecutingTool
			m.activeToolName = evt.ToolName
			m.activeToolInput = evt.ToolInput
			m.systemStatus = fmt.Sprintf("Ejecutando herramienta: %s", evt.ToolName)
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()

		case "tool:result":
			m.entries = append(m.entries, ChatEntry{
				Role:        "tool",
				ToolName:    evt.ToolName,
				Content:     evt.ToolOutput,
				ToolSuccess: evt.ToolSuccess,
				DurationMs:  evt.DurationMs,
			})
			m.activeToolName = ""
			m.activeToolInput = ""
			m.state = StateThinking
			m.systemStatus = "Analizando resultado de herramienta..."
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()

		case "agent:completed":
			content := cleanAssistantText(m.currentStream)
			if content == "" && strings.TrimSpace(evt.Content) != "" {
				content = cleanAssistantText(evt.Content)
			}
			if content != "" {
				thinking := strings.TrimSpace(m.currentThinking)
				if thinking == "" {
					thinking = strings.TrimSpace(evt.Thinking)
				}
				m.entries = append(m.entries, ChatEntry{
					Role:     "assistant",
					Content:  content,
					Thinking: thinking,
				})
			}
			m.currentStream = ""
			m.currentThinking = ""
			m.loopCancel = nil
			m.loopSessionID = ""
			m.state = StateIdle
			m.systemStatus = "Listo para actuar"
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()

			// Si hay mensajes en cola, desencolar el siguiente y procesarlo automáticamente
			if nextPrompt, ok := m.DequeuePrompt(); ok {
				roleContent := nextPrompt.Prompt
				if nextPrompt.IsVoice {
					roleContent = "🎙️ " + nextPrompt.Prompt
				}
				m.entries = append(m.entries, ChatEntry{
					Role:    "user",
					Content: roleContent,
				})
				m.state = StateThinking
				m.systemStatus = fmt.Sprintf("Procesando orden encolada (restantes: %d)...", len(m.messageQueue))
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				if nextPrompt.IsVoice {
					return m, m.startAgentTurnVoice(nextPrompt.Prompt)
				}
				return m, m.startAgentTurn(nextPrompt.Prompt)
			}

		case "error":
			m.entries = append(m.entries, ChatEntry{
				Role:    "system",
				Content: fmt.Sprintf("❌ Error: %s", evt.Error),
			})
			m.loopCancel = nil
			m.loopSessionID = ""
			m.state = StateIdle
			m.systemStatus = "Error en la ejecución"
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()

			// Si hay mensajes en cola tras un error, continuar con el siguiente
			if nextPrompt, ok := m.DequeuePrompt(); ok {
				roleContent := nextPrompt.Prompt
				if nextPrompt.IsVoice {
					roleContent = "🎙️ " + nextPrompt.Prompt
				}
				m.entries = append(m.entries, ChatEntry{
					Role:    "user",
					Content: roleContent,
				})
				m.state = StateThinking
				m.systemStatus = fmt.Sprintf("Procesando orden encolada tras error (restantes: %d)...", len(m.messageQueue))
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				if nextPrompt.IsVoice {
					return m, m.startAgentTurnVoice(nextPrompt.Prompt)
				}
				return m, m.startAgentTurn(nextPrompt.Prompt)
			}
		}

	case loopStartedMsg:
		m.loopSessionID = msg.sessionID
		m.loopCancel = msg.cancel
		return m, nil

	case errMsg:
		m.entries = append(m.entries, ChatEntry{
			Role:    "system",
			Content: fmt.Sprintf("❌ Error: %v", msg.err),
		})
		m.state = StateIdle
		m.systemStatus = "Error del sistema"
		m.viewport.SetContent(m.renderConversation())
		m.viewport.GotoBottom()

	case VoiceCommandMsg:
		prompt := strings.TrimSpace(msg.Command)
		if prompt == "" {
			return m, nil
		}

		if m.state != StateIdle {
			m.promptHistory = append(m.promptHistory, prompt)
			m.historyIndex = len(m.promptHistory)
			qPos := m.EnqueuePrompt(prompt, true)
			m.entries = append(m.entries, ChatEntry{
				Role:    "system",
				Content: fmt.Sprintf("📥 Orden de voz añadida a la cola [#%d]: \"%s\" (se procesará automáticamente al terminar la tarea actual).", qPos, prompt),
			})
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, nil
		}

		m.promptHistory = append(m.promptHistory, prompt)
		m.historyIndex = len(m.promptHistory)
		m.entries = append(m.entries, ChatEntry{
			Role:    "user",
			Content: "🎙️ " + prompt,
		})
		m.state = StateThinking
		m.systemStatus = "Escuchado por voz. Procesando orden..."
		m.currentStream = ""

		m.viewport.SetContent(m.renderConversation())
		m.viewport.GotoBottom()

		return m, m.startAgentTurnVoice(prompt)

	case VoiceStatusMsg:
		m.voiceEnabled = msg.Listening
		if msg.Status != "" {
			m.systemStatus = msg.Status
		}
		return m, nil
	}

	// Manejo del spinner (siempre se actualiza para no romper el ciclo Tick)
	var spinCmd tea.Cmd
	m.spinner, spinCmd = m.spinner.Update(msg)
	cmds = append(cmds, spinCmd)

	// Manejo del input de texto (siempre activo para permitir escribir órdenes en cola o comandos)
	var taCmd tea.Cmd
	m.textarea, taCmd = m.textarea.Update(msg)
	cmds = append(cmds, taCmd)

	// Actualización del viewport de scroll
	// Evitamos que las pulsaciones normales muevan el scroll
	var vpCmd tea.Cmd
	isKeyMsg := false
	var keyMsg tea.KeyMsg
	if k, ok := msg.(tea.KeyMsg); ok {
		isKeyMsg = true
		keyMsg = k
	}

	if !isKeyMsg {
		m.viewport, vpCmd = m.viewport.Update(msg)
	} else if keyMsg.Type == tea.KeyPgUp || keyMsg.Type == tea.KeyPgDown || keyMsg.Type == tea.KeyUp || keyMsg.Type == tea.KeyDown {
		m.viewport, vpCmd = m.viewport.Update(msg)
	}
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) startAgentTurn(prompt string) tea.Cmd {
	return func() tea.Msg {
		if m.chat != nil && m.chat.ID != "" {
			userMsg := &models.Message{
				ID:        uuid.NewString(),
				ChatID:    m.chat.ID,
				Role:      "user",
				Content:   prompt,
				CreatedAt: time.Now(),
			}
			_ = db.CreateMessage(userMsg)
			memory.StoreChatMessage(m.chat.UserID, m.chat.ProjectID, m.chat.ID, "user", prompt)
		}

		ctx, cancel := context.WithCancel(context.Background())

		params := agent.AgentLoopParams{
			Provider:        m.provider,
			Chat:            m.chat,
			UserMessage:     prompt,
			PermissionLevel: m.permissionLevel,
			VoiceMode:       false,
			Emit: func(evt agent.AgentEvent) {
				// El dispatcher emite eventos hacia el loop de Bubble Tea
				if currentProgram != nil {
					currentProgram.Send(agentEventMsg(evt))
				}
			},
		}

		sessionID := agent.StartAgentLoop(ctx, params)
		return loopStartedMsg{
			sessionID: sessionID,
			cancel:    cancel,
		}
	}
}

func (m *Model) startAgentTurnVoice(prompt string) tea.Cmd {
	return func() tea.Msg {
		if m.chat != nil && m.chat.ID != "" {
			userMsg := &models.Message{
				ID:        uuid.NewString(),
				ChatID:    m.chat.ID,
				Role:      "user",
				Content:   prompt,
				CreatedAt: time.Now(),
			}
			_ = db.CreateMessage(userMsg)
			memory.StoreChatMessage(m.chat.UserID, m.chat.ProjectID, m.chat.ID, "user", prompt)
		}

		ctx, cancel := context.WithCancel(context.Background())

		params := agent.AgentLoopParams{
			Provider:        m.provider,
			Chat:            m.chat,
			UserMessage:     prompt,
			PermissionLevel: m.permissionLevel,
			VoiceMode:       true, // VoiceMode sintetiza voz hablada con Piper TTS al finalizar
			Emit: func(evt agent.AgentEvent) {
				if currentProgram != nil {
					currentProgram.Send(agentEventMsg(evt))
				}
			},
		}

		sessionID := agent.StartAgentLoop(ctx, params)
		return loopStartedMsg{
			sessionID: sessionID,
			cancel:    cancel,
		}
	}
}

func (m *Model) handleSlashCommand(cmdStr string) string {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return ""
	}

	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "/help":
		return `📌 Comandos disponibles en OzyAssist TUI:
  /cancel [all]     - Cancela la petición activa en curso (o 'all' para vaciar cola)
  /now <orden>      - Interrumpe la tarea actual y ejecuta la orden inmediatamente
  /queue, /cola     - Consulta los mensajes pendientes en la cola de espera
  /clearqueue       - Vacía la cola de mensajes pendientes
  /provider [nom]   - Consulta o cambia el proveedor LLM activo (cohere, groq, openai, etc.)
  /key <prov> <key> - Configura API Key (cohere, groq, openrouter, openai, deepseek, anthropic, mistral) o URL local
  /groq [key]       - Auto-configura Groq desde portapapeles o abre Chrome autenticado para obtenerla
  /voice            - Alterna la escucha activa de voz ("Hey Ozy")
  /model [nombre]   - Consulta o cambia el proveedor/modelo actual
  /tools            - Lista todas las herramientas activas (SO + MCP)
  /mcp [reload]     - Consulta o recarga en caliente los servidores MCP (mcp_servers.json)
  /organize [dir]   - Ejecuta la organización rápida de una carpeta
  /clear            - Limpia el historial de la pantalla (Ctrl+L)
  /perm [modo]      - Cambia nivel de permisos: autonomous | supervised
  /exit, /quit      - Cierra la aplicación (Ctrl+C)
💡 Atajos: [Esc] o [Ctrl+C] para cancelar tarea • [Enter] encola si Ozy está ocupado`

	case "/cancel", "/stop", "/abort", "/cancelar", "/parar":
		clearAll := len(parts) > 1 && strings.ToLower(parts[1]) == "all"
		if m.state != StateIdle {
			if m.loopCancel != nil {
				m.loopCancel()
			}
			if m.loopSessionID != "" {
				agent.CancelSession(m.loopSessionID)
			}
			m.loopCancel = nil
			m.loopSessionID = ""
			m.state = StateIdle
			m.systemStatus = "Petición cancelada por el usuario"
			if clearAll {
				discarded := m.ClearQueue()
				return fmt.Sprintf("⚠️ Petición activa cancelada y %d mensajes en cola descartados.", discarded)
			}
			if len(m.messageQueue) > 0 {
				return fmt.Sprintf("⚠️ Petición cancelada. (Quedan %d mensajes en cola. Usa /queue para verlos o /clearqueue para vaciarla).", len(m.messageQueue))
			}
			return "⚠️ Petición cancelada con éxito."
		}
		if clearAll || len(m.messageQueue) > 0 {
			discarded := m.ClearQueue()
			return fmt.Sprintf("🗑️ Cola de mensajes vaciada (%d descartados).", discarded)
		}
		return "ℹ️ No hay ninguna petición activa ni mensajes en cola."

	case "/queue", "/cola":
		if len(m.messageQueue) == 0 {
			return "ℹ️ La cola de mensajes está vacía."
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📥 Mensajes en cola de espera (%d):\n", len(m.messageQueue)))
		for i, q := range m.messageQueue {
			vTag := ""
			if q.IsVoice {
				vTag = "🎙️ [Voz] "
			}
			sb.WriteString(fmt.Sprintf("  %d. %s\"%s\"\n", i+1, vTag, q.Prompt))
		}
		sb.WriteString("💡 Usa /clearqueue para vaciarla, /now <orden> para ejecutar de inmediato o /cancel para detener la tarea activa.")
		return sb.String()

	case "/clearqueue", "/dropqueue", "/vaciarcola":
		discarded := m.ClearQueue()
		return fmt.Sprintf("🗑️ Cola de mensajes descartada con éxito (%d mensajes eliminados).", discarded)

	case "/groq":
		targetKey := ""
		if len(parts) > 1 {
			targetKey = strings.TrimSpace(parts[1])
		} else {
			// Intentar leer del portapapeles del sistema
			clip, err := system.ReadClipboardContent()
			if err == nil {
				clip = strings.TrimSpace(clip)
				if strings.HasPrefix(clip, "gsk_") {
					targetKey = clip
				}
			}
		}

		if targetKey != "" {
			if !strings.HasPrefix(targetKey, "gsk_") {
				minLen := 6
				if len(targetKey) < minLen {
					minLen = len(targetKey)
				}
				return fmt.Sprintf("⚠️ La clave '%s...' no parece ser válida de Groq (debe comenzar con 'gsk_').", targetKey[:minLen])
			}
			_ = providers.SaveConfigKey("GROQ_API_KEY", targetKey)
			_ = os.Setenv("GROQ_API_KEY", targetKey)
			providers.RegisterProviderKey("groq", targetKey)
			if p, err := providers.Get("groq"); err == nil {
				m.provider = p
				if m.chat != nil {
					m.chat.Provider = "groq"
					if len(p.Models()) > 0 {
						m.chat.Model = p.Models()[0]
					}
				}
			}
			voiceMsg := ""
			if activeVoiceController != nil {
				_ = activeVoiceController.UpdateKey("groq", targetKey)
				if err := activeVoiceController.Start(); err == nil {
					m.voiceEnabled = true
					m.systemStatus = "Escuchando Wake Word ('Hey Ozy')..."
					voiceMsg = "\n🎙️ Motor de voz 'Hey Ozy' ACTIVADO y escuchando."
				}
			}
			return fmt.Sprintf("⚡ ¡Groq API Key configurada con éxito!\n✓ Guardada en .env\n✓ Proveedor activo: Groq (llama-3.3-70b-versatile @ 800 tokens/s)%s", voiceMsg)
		}

		// Si no hay clave, abrir navegador Chrome con el perfil autenticado del usuario
		profile := browser.FindMatchingProfile("chrome", "")
		groqURL := "https://console.groq.com/keys"
		chromeInfo := "tu navegador"
		if profile != nil && profile.ExecPath != "" {
			chromeInfo = fmt.Sprintf("Google Chrome (Perfil: %s | %s)", profile.Name, profile.Email)
			_ = browser.LaunchWithProfile(context.Background(), profile, groqURL)
		} else {
			_ = browser.LaunchWithProfile(context.Background(), nil, groqURL)
		}

		return fmt.Sprintf("🌐 Abriendo la consola de Groq en %s:\n"+
			"   👉 URL: %s\n"+
			"1. Inicia sesión con 1 clic (Continuar con Google).\n"+
			"2. Haz clic en 'Create API Key' y copia la clave generada.\n"+
			"3. Vuelve a esta terminal y escribe '/groq' (o simplemente dime por voz: 'guarda mi clave'). Ozy la leerá directamente del portapapeles.",
			chromeInfo, groqURL)

	case "/tools":
		var sb strings.Builder
		sb.WriteString("🛠️ Herramientas de Control del SO disponibles:\n")
		for _, t := range agent.AgentTools {
			sb.WriteString(fmt.Sprintf("  • %-22s: %s\n", t.Name, t.Description))
		}
		mcpTools := mcp.DefaultRegistry.GetAllTools()
		if len(mcpTools) > 0 {
			sb.WriteString("\n🔌 Herramientas MCP Externas Conectadas:\n")
			for name, t := range mcpTools {
				sb.WriteString(fmt.Sprintf("  • %-26s: %s\n", name, t.Description))
			}
		}
		return sb.String()

	case "/thinking", "/thought":
		m.showThinking = !m.showThinking
		if m.showThinking {
			return "💭 Hilo de pensamiento: VISIBLE y desplegado. (Presiona Ctrl+T o /thinking para ocultar)"
		}
		return "💭 Hilo de pensamiento: PLEGADO y oculto. (Presiona Ctrl+T o /thinking para desplegar)"

	case "/mcp":
		if len(parts) > 1 && strings.ToLower(parts[1]) == "reload" {
			cfgPath := mcp.FindDefaultConfigFile()
			if cfgPath == "" {
				return "⚠️ No se encontró ningún archivo mcp_servers.json en las rutas canónicas (backend/mcp_servers.json o ~/.ozy/mcp_servers.json)."
			}
			configs, err := mcp.LoadConfigFile(cfgPath)
			if err != nil {
				return fmt.Sprintf("❌ Error leyendo %s: %v", cfgPath, err)
			}
			if err := mcp.DefaultRegistry.Reload(context.Background(), configs); err != nil {
				return fmt.Sprintf("❌ Error recargando servidores MCP: %v", err)
			}
			statuses := mcp.DefaultRegistry.GetServerStatus()
			totalTools := len(mcp.DefaultRegistry.GetAllTools())
			return fmt.Sprintf("✓ Servidores MCP recargados desde %s (%d servidores, %d herramientas activas).", cfgPath, len(statuses), totalTools)
		}

		statuses := mcp.DefaultRegistry.GetServerStatus()
		if len(statuses) == 0 {
			cfgPath := mcp.FindDefaultConfigFile()
			hint := "Crea un archivo 'mcp_servers.json' para conectar servidores estándar (Claude Desktop / Cursor)."
			if cfgPath != "" {
				hint = fmt.Sprintf("Archivo detectado: %s (sin servidores activos o con errores).", cfgPath)
			}
			return fmt.Sprintf("🔌 No hay servidores MCP conectados actualmente.\n💡 %s\nUsa '/mcp reload' para recargar en caliente.", hint)
		}

		var sb strings.Builder
		sb.WriteString("🔌 Servidores MCP (Model Context Protocol) Conectados:\n")
		for _, s := range statuses {
			icon := "🟢"
			if s.Status == "error" {
				icon = "🔴"
			} else if s.Status == "stopped" {
				icon = "⚪"
			}
			sb.WriteString(fmt.Sprintf("  %s %-14s [%s] — %d herramientas (%s)\n", icon, s.Name, s.Status, s.ToolCount, s.Command))
			if s.Error != "" {
				sb.WriteString(fmt.Sprintf("     ⚠️ Error: %s\n", s.Error))
			}
		}
		sb.WriteString("\n💡 Usa '/mcp reload' para recargar en caliente tras editar mcp_servers.json.")
		return sb.String()

	case "/provider", "/providers":
		if len(parts) > 1 {
			target := strings.ToLower(parts[1])
			if target == "select" && len(parts) > 2 {
				target = strings.ToLower(parts[2])
			}

			// Intentar obtener el proveedor
			p, err := providers.Get(target)
			if err != nil {
				key := providers.GetProviderKey(target)
				if key != "" {
					providers.RegisterProviderKey(target, key)
					p, err = providers.Get(target)
				}
			}

			if err != nil || p == nil {
				return fmt.Sprintf("⚠️ Proveedor '%s' no reconocido o sin registrar.\n"+
					"💡 Usa '/provider' sin argumentos para ver los proveedores disponibles.", target)
			}

			key := providers.GetProviderKey(target)
			if key == "" && target != "lmstudio" && target != "ollama" && target != "local" {
				return fmt.Sprintf("⚠️ El proveedor '%s' no tiene una API key configurada.\n"+
					"👉 Configúrala escribiendo: /key %s <tu-api-key>", target, target)
			}

			m.provider = p
			defaultModel := ""
			if len(p.Models()) > 0 {
				defaultModel = p.Models()[0]
			}
			if m.chat != nil {
				m.chat.Provider = target
				m.chat.Model = defaultModel
			}

			return fmt.Sprintf("✓ Proveedor activo cambiado a: %s\n✓ Modelo predeterminado: %s", target, defaultModel)
		}

		allProviders := []struct {
			id       string
			label    string
			desc     string
			defModel string
		}{
			{"cohere", "Cohere", "Command R+ / R7B (potente, agéntico y multilingüe)", "command-r-plus-08-2024"},
			{"groq", "Groq", "Llama 3.3 70B ultra-rápido (~800 tok/s)", "llama-3.3-70b-versatile"},
			{"openai", "OpenAI", "GPT-4o, GPT-4o-mini", "gpt-4o"},
			{"openrouter", "OpenRouter", "Multi-proveedor / DeepSeek / Claude / Llama", "deepseek/deepseek-chat"},
			{"anthropic", "Anthropic", "Claude 3.5 Sonnet", "claude-3-5-sonnet-20241022"},
			{"deepseek", "DeepSeek", "DeepSeek-V3 / R1 nativo", "deepseek-chat"},
			{"mistral", "Mistral AI", "mistral-large, codestral, mixtral-8x22b", "mistral-large-latest"},
			{"lmstudio", "LM Studio / Local", "Modelos locales vía HTTP", "local-model"},
			{"ollama", "Ollama", "Modelos locales vía Ollama API", "local-model"},
		}

		currProv := ""
		if m.provider != nil {
			currProv = strings.ToLower(m.provider.Name())
		} else if m.chat != nil && m.chat.Provider != "" {
			currProv = strings.ToLower(m.chat.Provider)
		}

		var sb strings.Builder
		sb.WriteString("🤖 Proveedores de IA disponibles en OzyAssist:\n")
		for _, p := range allProviders {
			key := providers.GetProviderKey(p.id)
			hasConfig := key != "" || (p.id == "lmstudio" || p.id == "ollama")

			icon := "⚪"
			statusText := "Sin configurar"
			if hasConfig {
				icon = "🟡"
				statusText = "Clave Añadida (sin validar)"
			}
			if p.id == currProv {
				icon = "⭐ 🟢"
				statusText = "ACTIVO"
			}

			sb.WriteString(fmt.Sprintf("  %-5s %-14s [%s] — %s (%s)\n", icon, p.label, statusText, p.desc, p.defModel))
		}

		sb.WriteString("\n👉 Para cambiar de proveedor activo escribe:\n")
		sb.WriteString("   /provider cohere\n")
		sb.WriteString("   /provider groq\n")
		sb.WriteString("   /provider openai\n")
		sb.WriteString("💡 Para configurar la clave de un proveedor usa:\n")
		sb.WriteString("   /key <proveedor> <tu-api-key>")
		return sb.String()

	case "/model":
		if len(parts) > 1 {
			target := parts[1]
			// 1. Si es un proveedor registrado (ej: /model openrouter)
			if prov, err := providers.Get(target); err == nil {
				m.provider = prov
				if m.chat != nil {
					m.chat.Provider = target
					if len(prov.Models()) > 0 {
						m.chat.Model = prov.Models()[0]
					}
				}
				return fmt.Sprintf("✓ Proveedor: %s (Modelo: %s)", target, m.chat.Model)
			}
			// 2. Si es un nombre de modelo directo (ej: /model deepseek/deepseek-chat)
			if m.chat != nil {
				m.chat.Model = target
				return fmt.Sprintf("✓ Modelo establecido a: %s", target)
			}
			return fmt.Sprintf("⚠️ No se pudo asignar el modelo: %s", target)
		}
		currentM := "por defecto"
		if m.chat != nil && m.chat.Model != "" {
			currentM = m.chat.Model
		}
		if m.provider != nil {
			return fmt.Sprintf("Proveedor actual: %s | Modelo: %s (Modelos recomendados: %v)", m.provider.Name(), currentM, m.provider.Models())
		}
		return "Ningún proveedor configurado."

	case "/organize":
		target := "mis descargas"
		if len(parts) > 1 {
			target = strings.Join(parts[1:], " ")
		}
		// Inyectar orden directa al loop
		m.textarea.SetValue(fmt.Sprintf("Organiza la carpeta %s", target))
		return ""

	case "/perm":
		if len(parts) > 1 {
			mode := strings.ToLower(parts[1])
			if mode == "autonomous" || mode == "supervised" || mode == "sandboxed" {
				m.permissionLevel = mode
				return fmt.Sprintf("✓ Nivel de permisos establecido en: %s", mode)
			}
			return "Modos válidos: autonomous | supervised | sandboxed"
		}
		return fmt.Sprintf("Nivel de permisos actual: %s", m.permissionLevel)

	case "/voice":
		if activeVoiceController == nil {
			m.voiceEnabled = !m.voiceEnabled
			if m.voiceEnabled {
				m.systemStatus = "Escuchando Wake Word ('Hey Ozy')..."
				return "🎙️ Escucha nativa de Wake Word 'Hey Ozy' ACTIVADA."
			}
			m.systemStatus = "Listo para actuar"
			return "🔇 Escucha nativa de voz DESACTIVADA."
		}

		if m.voiceEnabled {
			activeVoiceController.Stop()
			m.voiceEnabled = false
			m.systemStatus = "Listo para actuar"
			return "🔇 Escucha nativa de voz DESACTIVADA."
		}

		if !activeVoiceController.HasSTT() {
			return `⚠️ Para activar la escucha por voz ("Hey Ozy"), necesitas un motor de transcripción (STT).
👉 Configura una API Key gratuita de Groq (~150ms) escribiendo:
   /key groq <tu-api-key>
   (Obtén una gratis en https://console.groq.com/keys)
O con OpenAI Whisper:
   /key openai <tu-api-key>`
		}

		if err := activeVoiceController.Start(); err != nil {
			return fmt.Sprintf("❌ Error al arrancar la escucha de voz: %v", err)
		}
		m.voiceEnabled = true
		m.systemStatus = "Escuchando Wake Word ('Hey Ozy')..."
		return "🎙️ Escucha nativa de Wake Word 'Hey Ozy' ACTIVADA."

	case "/clear":
		m.entries = nil
		return ""

	case "/key":
		if len(parts) < 3 {
			return `📌 Uso de /key:
  /key cohere <tu-api-key>      (Command R+ de Cohere)
  /key groq <tu-api-key>        (Voz 'Hey Ozy' ultrarrápida gratis)
  /key openai <tu-api-key>
  /key openrouter <tu-api-key>
  /key deepseek <tu-api-key>
  /key anthropic <tu-api-key>
  /key mistral <tu-api-key>     (mistral-large-latest, codestral, etc.)
  /key local http://localhost:1234 (para LM Studio u Ollama)`
		}
		targetProv := strings.ToLower(parts[1])
		keyVal := strings.TrimSpace(parts[2])

		if targetProv == "cohere" {
			_ = providers.SaveConfigKey("COHERE_API_KEY", keyVal)
			_ = os.Setenv("COHERE_API_KEY", keyVal)
			providers.RegisterProviderKey("cohere", keyVal)
			if p, err := providers.Get("cohere"); err == nil {
				m.provider = p
				if m.chat != nil {
					m.chat.Provider = "cohere"
					if len(p.Models()) > 0 {
						m.chat.Model = p.Models()[0]
					}
				}
			}
			modelName := "command-r-plus-08-2024"
			if m.chat != nil && m.chat.Model != "" {
				modelName = m.chat.Model
			}
			return fmt.Sprintf("✓ Proveedor 'cohere' configurado y activado (Modelo: %s).", modelName)
		}

		if targetProv == "groq" {
			_ = providers.SaveConfigKey("GROQ_API_KEY", keyVal)
			_ = os.Setenv("GROQ_API_KEY", keyVal)
			providers.RegisterProviderKey("groq", keyVal)
			if p, err := providers.Get("groq"); err == nil {
				m.provider = p
				if m.chat != nil {
					m.chat.Provider = "groq"
					if len(p.Models()) > 0 {
						m.chat.Model = p.Models()[0]
					}
				}
			}
			if activeVoiceController != nil {
				_ = activeVoiceController.UpdateKey("groq", keyVal)
				if err := activeVoiceController.Start(); err == nil {
					m.voiceEnabled = true
					m.systemStatus = "Escuchando Wake Word ('Hey Ozy')..."
					return "✓ API Key de Groq guardada en .env. Motor de voz 'Hey Ozy' ACTIVADO y escuchando."
				}
			}
			return "✓ API Key de Groq guardada en .env. Activa la voz con /voice."
		}

		if targetProv == "openai" {
			_ = providers.SaveConfigKey("OPENAI_API_KEY", keyVal)
			_ = os.Setenv("OPENAI_API_KEY", keyVal)
			providers.RegisterProviderKey("openai", keyVal)
			if activeVoiceController != nil {
				_ = activeVoiceController.UpdateKey("openai", keyVal)
			}
			if p, err := providers.Get("openai"); err == nil {
				m.provider = p
				if m.chat != nil {
					m.chat.Provider = "openai"
					if len(p.Models()) > 0 {
						m.chat.Model = p.Models()[0]
					}
				}
			}
			extraHint := ""
			if !strings.HasPrefix(keyVal, "sk-") && len(keyVal) == 40 {
				extraHint = "\n💡 Nota: Esta clave parece de Cohere. Puedes activarla como proveedor nativo con: /key cohere " + keyVal + " o /provider cohere"
			}
			return fmt.Sprintf("✓ Proveedor 'openai' configurado y guardado en .env (disponible para LLM y Whisper STT).%s", extraHint)
		}

		if targetProv == "local" || targetProv == "lmstudio" || targetProv == "ollama" && strings.HasPrefix(keyVal, "http") {
			providers.RegisterLocalHostURL(keyVal)
			_ = providers.SaveConfigKey("LMSTUDIO_URL", keyVal)
			if p, err := providers.Get("lmstudio"); err == nil {
				m.provider = p
			}
			return fmt.Sprintf("✓ Endpoint local actualizado y guardado: %s", keyVal)
		}

		if targetProv == "mistral" {
			_ = providers.SaveConfigKey("MISTRAL_API_KEY", keyVal)
			_ = os.Setenv("MISTRAL_API_KEY", keyVal)
			providers.RegisterProviderKey("mistral", keyVal)
			if p, err := providers.Get("mistral"); err == nil {
				m.provider = p
				if m.chat != nil {
					m.chat.Provider = "mistral"
					if len(p.Models()) > 0 {
						m.chat.Model = p.Models()[0]
					}
				}
			}
			modelName := "mistral-large-latest"
			if m.chat != nil && m.chat.Model != "" {
				modelName = m.chat.Model
			}
			return fmt.Sprintf("🌪️ Proveedor 'mistral' configurado y activado (Modelo: %s).\n"+
				"✓ Clave guardada en .env\n"+
				"💡 Modelos disponibles: mistral-large-latest, codestral-latest, open-mixtral-8x22b\n"+
				"   Cambia modelo con: /model mistral-large-latest", modelName)
		}

		providers.RegisterProviderKey(targetProv, keyVal)
		_ = providers.SaveConfigKey(strings.ToUpper(targetProv)+"_API_KEY", keyVal)
		if p, err := providers.Get(targetProv); err == nil {
			m.provider = p
			modelName := ""
			if len(p.Models()) > 0 {
				modelName = p.Models()[0]
			}
			if m.chat != nil {
				m.chat.Provider = targetProv
				m.chat.Model = modelName
			}
			return fmt.Sprintf("✓ Proveedor '%s' configurado y guardado en .env (Modelo: %s).", targetProv, modelName)
		}
		return fmt.Sprintf("⚠️ Proveedor '%s' no reconocido. Disponibles: groq, openrouter, openai, deepseek, anthropic, mistral, local", targetProv)

	case "/exit", "/quit":
		return "QUIT"

	default:
		return fmt.Sprintf("Comando desconocido: %s. Escribe /help para ver la lista.", cmd)
	}
}

// VoiceController define las operaciones de control de voz que la TUI puede invocar
type VoiceController interface {
	HasSTT() bool
	IsRunning() bool
	Start() error
	Stop()
	UpdateKey(provider, key string) error
}

var activeVoiceController VoiceController

func SetVoiceController(vc VoiceController) {
	activeVoiceController = vc
}

// Singleton reference for goroutines emitting events into Bubble Tea
var currentProgram *tea.Program

func SetProgram(p *tea.Program) {
	currentProgram = p
}

// SendVoiceCommand envía una orden reconocida por voz a la interfaz TUI activa
func SendVoiceCommand(command string) {
	if currentProgram != nil {
		currentProgram.Send(VoiceCommandMsg{Command: command})
	}
}

// SendVoiceStatus actualiza el estado del motor de voz en la barra de estado de la TUI
func SendVoiceStatus(listening bool, status string) {
	if currentProgram != nil {
		currentProgram.Send(VoiceStatusMsg{Listening: listening, Status: status})
	}
}
