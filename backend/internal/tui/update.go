package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
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
		taWidth := msg.Width - 14
		if taWidth < 20 {
			taWidth = 20
		}
		m.textarea.SetWidth(taWidth)

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
		if m.state == StateStartMenu {
			switch msg.Type {
			case tea.KeyUp:
				m.menuIndex--
				if m.menuIndex < 0 {
					m.menuIndex = 3
				}
				return m, nil
			case tea.KeyDown:
				m.menuIndex++
				if m.menuIndex > 3 {
					m.menuIndex = 0
				}
				return m, nil
			case tea.KeyEnter:
				switch m.menuIndex {
				case 0:
					m.state = StateIdle
					m.textarea.Focus()
					if m.ready {
						m.viewport.SetContent(m.renderConversation())
						m.viewport.GotoBottom()
					}
					return m, textarea.Blink
				case 1:
					// Historial de conversaciones
					_ = db.CleanupEmptyChats("")
					chats, err := db.ListChats()
					if err != nil {
						chats = nil
					}
					m.historyChats = chats
					m.chatHistoryIndex = 0
					m.historyConfirmDelete = ""
					m.historySearchInput.Reset()
					m.historySearchInput.Focus()
					m.state = StateChatHistory
					return m, textinput.Blink
				case 2:
					m.state = StateSettingsMenu
					m.settingsIndex = 0
					m.settingsNotice = ""
					return m, nil
				case 3:
					return m, tea.Quit
				}
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyRunes:
				s := string(msg.Runes)
				switch s {
				case "k", "w":
					m.menuIndex--
					if m.menuIndex < 0 {
						m.menuIndex = 3
					}
					return m, nil
				case "j", "s":
					m.menuIndex++
					if m.menuIndex > 3 {
						m.menuIndex = 0
					}
					return m, nil
				case "1":
					m.menuIndex = 0
					m.state = StateIdle
					m.textarea.Focus()
					if m.ready {
						m.viewport.SetContent(m.renderConversation())
						m.viewport.GotoBottom()
					}
					return m, textarea.Blink
				case "2":
					_ = db.CleanupEmptyChats("")
					chats, err := db.ListChats()
					if err != nil {
						chats = nil
					}
					m.historyChats = chats
					m.chatHistoryIndex = 0
					m.historyConfirmDelete = ""
					m.historySearchInput.Reset()
					m.historySearchInput.Focus()
					m.state = StateChatHistory
					return m, textinput.Blink
				case "3":
					m.menuIndex = 2
					m.state = StateSettingsMenu
					m.settingsIndex = 0
					m.settingsNotice = ""
					return m, nil
				case "4", "q", "Q":
					return m, tea.Quit
				}
			}
			return m, nil
		}

		if m.state == StateChatHistory {
			filtered := m.getFilteredHistoryChats()
			totalItems := len(filtered) + 2

			// Si hay confirmación de borrado activa, solo procesar confirmación s/n/Esc
			if m.historyConfirmDelete != "" {
				switch msg.Type {
				case tea.KeyEsc:
					m.historyConfirmDelete = ""
					return m, nil
				case tea.KeyRunes:
					switch string(msg.Runes) {
					case "s", "S", "y", "Y":
						if err := db.DeleteChat(m.historyConfirmDelete); err == nil {
							_ = db.CleanupEmptyChats("")
							chats, _ := db.ListChats()
							m.historyChats = chats
							filteredNow := m.getFilteredHistoryChats()
							if m.chatHistoryIndex >= len(filteredNow)+2 {
								m.chatHistoryIndex = max(0, len(filteredNow)+1)
							}
						}
						m.historyConfirmDelete = ""
						return m, nil
					case "n", "N":
						m.historyConfirmDelete = ""
						return m, nil
					}
				case tea.KeyCtrlC:
					return m, tea.Quit
				}
				return m, nil
			}

			switch msg.Type {
			case tea.KeyUp:
				m.chatHistoryIndex--
				if m.chatHistoryIndex < 0 {
					m.chatHistoryIndex = totalItems - 1
				}
				return m, nil

			case tea.KeyDown:
				m.chatHistoryIndex++
				if m.chatHistoryIndex >= totalItems {
					m.chatHistoryIndex = 0
				}
				return m, nil

			case tea.KeyEsc:
				if m.historySearchInput.Value() != "" {
					m.historySearchInput.Reset()
					m.chatHistoryIndex = 0
					return m, nil
				}
				m.state = StateStartMenu
				return m, nil

			case tea.KeyCtrlC:
				return m, tea.Quit

			case tea.KeyEnter:
				idx := m.chatHistoryIndex
				if idx < len(filtered) {
					// Reanudar sesión seleccionada
					selected := filtered[idx]
					m.chat = &selected
					msgs, err := db.GetMessages(selected.ID)
					m.entries = nil
					if err == nil {
						for _, msg := range msgs {
							role := msg.Role
							if role != "user" && role != "assistant" && role != "system" && role != "tool" {
								role = "system"
							}
							m.entries = append(m.entries, ChatEntry{
								Role:    role,
								Content: msg.Content,
							})
						}
					}
					if m.provider != nil {
						m.chat.Provider = m.provider.Name()
						if len(m.provider.Models()) > 0 && m.chat.Model == "" {
							m.chat.Model = m.provider.Models()[0]
						}
						_ = db.UpdateChat(m.chat)
					}
					m.activeCard = nil
					m.state = StateIdle
					m.textarea.Focus()
					if m.ready {
						m.viewport.SetContent(m.renderConversation())
						m.viewport.GotoBottom()
					}
					return m, textarea.Blink
				} else if idx == len(filtered) {
					// Nueva conversación limpia
					return m.handleNewChat()
				} else {
					// Volver al menú principal
					m.state = StateStartMenu
					return m, nil
				}

			case tea.KeyRunes:
				s := string(msg.Runes)
				// Si el buscador está vacío y se presiona 'd' o 'D', activar eliminación
				if m.historySearchInput.Value() == "" && (s == "d" || s == "D") {
					if m.chatHistoryIndex < len(filtered) {
						m.historyConfirmDelete = filtered[m.chatHistoryIndex].ID
					}
					return m, nil
				}
				// Escribir en el buscador interactivo
				var cmd tea.Cmd
				m.historySearchInput, cmd = m.historySearchInput.Update(msg)
				m.chatHistoryIndex = 0
				return m, cmd

			case tea.KeyBackspace, tea.KeyDelete:
				var cmd tea.Cmd
				m.historySearchInput, cmd = m.historySearchInput.Update(msg)
				m.chatHistoryIndex = 0
				return m, cmd

			default:
				var cmd tea.Cmd
				m.historySearchInput, cmd = m.historySearchInput.Update(msg)
				return m, cmd
			}
		}

		if m.state == StateSettingsMenu {
			switch msg.Type {
			case tea.KeyUp:
				m.settingsIndex--
				if m.settingsIndex < 0 {
					m.settingsIndex = 6
				}
				return m, nil
			case tea.KeyDown:
				m.settingsIndex++
				if m.settingsIndex > 6 {
					m.settingsIndex = 0
				}
				return m, nil
			case tea.KeyEsc:
				m.state = StateStartMenu
				m.settingsNotice = ""
				return m, nil
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyEnter:
				return m.handleSettingsEnter()
			case tea.KeyRunes:
				s := string(msg.Runes)
				switch s {
				case "k", "w":
					m.settingsIndex--
					if m.settingsIndex < 0 {
						m.settingsIndex = 6
					}
					return m, nil
				case "j", "s":
					m.settingsIndex++
					if m.settingsIndex > 6 {
						m.settingsIndex = 0
					}
					return m, nil
				case "q", "Q":
					m.state = StateStartMenu
					m.settingsNotice = ""
					return m, nil
				case "1":
					m.settingsIndex = 0
					return m.handleSettingsEnter()
				case "2":
					m.settingsIndex = 1
					return m.handleSettingsEnter()
				case "3":
					m.settingsIndex = 2
					return m.handleSettingsEnter()
				case "4":
					m.settingsIndex = 3
					return m.handleSettingsEnter()
				case "5":
					m.settingsIndex = 4
					return m.handleSettingsEnter()
				case "6":
					m.settingsIndex = 5
					return m.handleSettingsEnter()
				case "7":
					m.state = StateStartMenu
					m.settingsNotice = ""
					return m, nil
				}
			}
			return m, nil
		}

		if m.state == StateProviderMenu {
			catalog := providers.GetSupportedCatalog()
			maxIdx := len(catalog) // len(catalog) es la opción "Volver a Configuraciones"
			switch msg.Type {
			case tea.KeyUp:
				m.providerIndex--
				if m.providerIndex < 0 {
					m.providerIndex = maxIdx
				}
				return m, nil
			case tea.KeyDown:
				m.providerIndex++
				if m.providerIndex > maxIdx {
					m.providerIndex = 0
				}
				return m, nil
			case tea.KeyEsc:
				m.state = StateSettingsMenu
				m.settingsNotice = ""
				return m, nil
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyEnter:
				return m.handleProviderSelectEnter()
			case tea.KeyRunes:
				s := string(msg.Runes)
				switch s {
				case "k", "w":
					m.providerIndex--
					if m.providerIndex < 0 {
						m.providerIndex = maxIdx
					}
					return m, nil
				case "j", "s":
					m.providerIndex++
					if m.providerIndex > maxIdx {
						m.providerIndex = 0
					}
					return m, nil
				case "q", "Q":
					m.state = StateSettingsMenu
					m.settingsNotice = ""
					return m, nil
				case "e", "E":
					if m.providerIndex < len(catalog) {
						cat := catalog[m.providerIndex]
						if !cat.IsLocal {
							m.apiKeyTarget = cat.ID
							m.apiKeyInput.SetValue("")
							m.apiKeyInput.Focus()
							m.state = StateAPIKeyInput
							m.settingsNotice = ""
							return m, textinput.Blink
						} else {
							m.settingsNotice = fmt.Sprintf("[INFO] %s es un servidor local y no requiere clave API.", cat.DisplayName)
							return m, nil
						}
					}
					return m, nil
				default:
					if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
						idx := int(s[0] - '1')
						if idx <= maxIdx {
							m.providerIndex = idx
							return m.handleProviderSelectEnter()
						}
					}
				}
			}
			return m, nil
		}

		if m.state == StateAPIKeySelect {
			catalog := providers.GetSupportedCatalog()
			maxIdx := len(catalog) // última opción es volver
			switch msg.Type {
			case tea.KeyUp:
				m.apiKeyIndex--
				if m.apiKeyIndex < 0 {
					m.apiKeyIndex = maxIdx
				}
				return m, nil
			case tea.KeyDown:
				m.apiKeyIndex++
				if m.apiKeyIndex > maxIdx {
					m.apiKeyIndex = 0
				}
				return m, nil
			case tea.KeyEsc:
				m.state = StateSettingsMenu
				m.settingsNotice = ""
				return m, nil
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyEnter:
				if m.apiKeyIndex >= len(catalog) {
					m.state = StateSettingsMenu
					m.settingsNotice = ""
					return m, nil
				}
				selected := catalog[m.apiKeyIndex]
				m.apiKeyTarget = selected.ID
				m.apiKeyInput.SetValue("")
				m.apiKeyInput.Focus()
				m.state = StateAPIKeyInput
				m.settingsNotice = ""
				return m, textinput.Blink
			case tea.KeyRunes:
				s := string(msg.Runes)
				switch s {
				case "k", "w":
					m.apiKeyIndex--
					if m.apiKeyIndex < 0 {
						m.apiKeyIndex = maxIdx
					}
					return m, nil
				case "j", "s":
					m.apiKeyIndex++
					if m.apiKeyIndex > maxIdx {
						m.apiKeyIndex = 0
					}
					return m, nil
				case "q", "Q":
					m.state = StateSettingsMenu
					m.settingsNotice = ""
					return m, nil
				default:
					if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
						idx := int(s[0] - '1')
						if idx <= maxIdx {
							m.apiKeyIndex = idx
							if m.apiKeyIndex >= len(catalog) {
								m.state = StateSettingsMenu
								m.settingsNotice = ""
								return m, nil
							}
							selected := catalog[m.apiKeyIndex]
							m.apiKeyTarget = selected.ID
							m.apiKeyInput.SetValue("")
							m.apiKeyInput.Focus()
							m.state = StateAPIKeyInput
							m.settingsNotice = ""
							return m, textinput.Blink
						}
					}
				}
			}
			return m, nil
		}

		if m.state == StateAPIKeyInput {
			switch msg.Type {
			case tea.KeyEsc:
				m.state = StateAPIKeySelect
				m.apiKeyInput.Blur()
				return m, nil
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyEnter:
				keyVal := strings.TrimSpace(m.apiKeyInput.Value())
				if keyVal != "" {
					targetProv := m.apiKeyTarget
					envVar := strings.ToUpper(targetProv) + "_API_KEY"
					_ = providers.SaveConfigKey(envVar, keyVal)
					_ = os.Setenv(envVar, keyVal)
					providers.RegisterProviderKey(targetProv, keyVal)
					if p, err := providers.Get(targetProv); err == nil && p != nil {
						m.provider = p
					}
					m.settingsNotice = fmt.Sprintf("[OK] Clave API de %s guardada y registrada exitosamente.", strings.ToUpper(targetProv))
				} else {
					m.settingsNotice = "[INFO] Entrada vacía, no se guardó ninguna clave."
				}
				m.apiKeyInput.Blur()
				m.state = StateAPIKeySelect
				return m, nil
			default:
				var cmd tea.Cmd
				m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
				return m, cmd
			}
		}

		if m.state == StateProfileEdit {
			switch msg.Type {
			case tea.KeyEsc:
				m.state = StateSettingsMenu
				m.profileInput.Blur()
				return m, nil
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyEnter:
				newDirective := strings.TrimSpace(m.profileInput.Value())
				if newDirective != "" && db.DB != nil {
					currProfile := ""
					if u, err := db.GetUser(db.DefaultUserID()); err == nil && u != nil {
						currProfile = u.ProfileMd
					}
					var updated string
					if strings.TrimSpace(currProfile) == "" {
						updated = fmt.Sprintf("# Perfil del Usuario\n- %s", newDirective)
					} else {
						updated = fmt.Sprintf("%s\n- %s", strings.TrimSpace(currProfile), newDirective)
					}
					_ = db.UpdateUserProfile(db.DefaultUserID(), updated)
					m.settingsNotice = "[OK] Perfil de usuario actualizado y guardado permanentemente en SQLite."
				} else {
					m.settingsNotice = "[INFO] Entrada vacía, no se realizaron cambios en el perfil."
				}
				m.profileInput.Blur()
				m.state = StateSettingsMenu
				return m, nil
			default:
				var cmd tea.Cmd
				m.profileInput, cmd = m.profileInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.Type {
		case tea.KeyEsc:
			if m.isBusy() {
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
				msgText := "[ALERTA] Operación interrumpida con la tecla [Esc]."
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

			// En reposo (StateIdle), si el campo de texto está vacío, regresar al Menú Principal
			if strings.TrimSpace(m.textarea.Value()) == "" {
				if m.chat != nil && m.chat.ID != "" {
					count, _ := db.CountMessagesByChat(m.chat.ID)
					if count == 0 {
						_ = db.DeleteChat(m.chat.ID)
					}
				}
				_ = db.CleanupEmptyChats("")

				m.state = StateStartMenu
				m.settingsNotice = ""
				m.activeCard = nil
				return m, nil
			}
			// Si hay texto escrito, limpiar el campo de entrada
			m.textarea.Reset()
			return m, nil

		case tea.KeyUp:
			if m.state == StateIdle && m.activeCard != nil && strings.TrimSpace(m.textarea.Value()) == "" && len(m.activeCard.Options) > 0 {
				m.activeCard.SelectedIndex--
				if m.activeCard.SelectedIndex < 0 {
					m.activeCard.SelectedIndex = len(m.activeCard.Options) - 1
				}
				m.viewport.SetContent(m.renderConversation())
				return m, nil
			}

		case tea.KeyDown:
			if m.state == StateIdle && m.activeCard != nil && strings.TrimSpace(m.textarea.Value()) == "" && len(m.activeCard.Options) > 0 {
				m.activeCard.SelectedIndex++
				if m.activeCard.SelectedIndex >= len(m.activeCard.Options) {
					m.activeCard.SelectedIndex = 0
				}
				m.viewport.SetContent(m.renderConversation())
				return m, nil
			}

		case tea.KeyTab:
			if m.state == StateIdle && m.activeCard != nil && len(m.activeCard.Tabs) > 0 && strings.TrimSpace(m.textarea.Value()) == "" {
				m.activeCard.ActiveTab = (m.activeCard.ActiveTab + 1) % len(m.activeCard.Tabs)
				m.viewport.SetContent(m.renderConversation())
				return m, nil
			}

		case tea.KeyCtrlC:
			if m.isBusy() {
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
				msgText := "[ALERTA] Operación interrumpida con Ctrl+C."
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
			m.activeCard = nil
			m.viewport.SetContent(m.renderConversation())
			return m, nil

		case tea.KeyCtrlT:
			m.showThinking = !m.showThinking
			if m.showThinking {
				m.systemStatus = "[PENSAMIENTO] Hilo de pensamiento: VISIBLE (Presiona Ctrl+T para ocultar)"
			} else {
				m.systemStatus = "[PENSAMIENTO] Hilo de pensamiento: PLEGADO (Presiona Ctrl+T para desplegar)"
			}
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, nil

		case tea.KeyEnter:
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				if m.state == StateIdle && m.activeCard != nil && len(m.activeCard.Options) > 0 {
					selectedOpt := m.activeCard.Options[m.activeCard.SelectedIndex]
					m.activeCard.IsAnswered = true
					m.activeCard.Answer = selectedOpt
					m.activeCard = nil

					// Determinar orden según la opción elegida
					prompt := selectedOpt
					if strings.HasPrefix(selectedOpt, "1.") {
						prompt = "Explorar y ordenar mis proyectos locales"
					} else if strings.HasPrefix(selectedOpt, "2.") {
						prompt = "/paths"
					} else if strings.HasPrefix(selectedOpt, "3.") {
						m.state = StateSettingsMenu
						m.settingsNotice = ""
						return m, nil
					} else if strings.HasPrefix(selectedOpt, "4.") {
						// Solo enfocar textarea para redactar orden libre
						m.viewport.SetContent(m.renderConversation())
						return m, textarea.Blink
					}

					if strings.HasPrefix(prompt, "/") {
						out := m.handleSlashCommand(prompt)
						m.entries = append(m.entries, ChatEntry{
							Role:    "user",
							Content: prompt,
						})
						if out != "" {
							m.entries = append(m.entries, ChatEntry{
								Role:    "system",
								Content: out,
							})
						}
						m.viewport.SetContent(m.renderConversation())
						m.viewport.GotoBottom()
						return m, nil
					}

					m.entries = append(m.entries, ChatEntry{
						Role:    "user",
						Content: prompt,
					})
					m.state = StateThinking
					m.systemStatus = "Procesando razonamiento agéntico..."
					m.currentStream = ""
					m.viewport.SetContent(m.renderConversation())
					m.viewport.GotoBottom()
					return m, m.startAgentTurn(prompt)
				}
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
					if m.isBusy() {
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
						msgText := "[ALERTA] Petición cancelada con éxito."
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
								Content: fmt.Sprintf("[LIMPIEZA] Se vació la cola de mensajes (%d descartados).", discarded),
							})
						} else {
							m.entries = append(m.entries, ChatEntry{
								Role:    "system",
								Content: "[INFO] No hay ninguna petición activa ni mensajes en cola.",
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
							Content: "[INFO] Uso: /now <orden> — Cancela la tarea actual y ejecuta la nueva orden inmediatamente.",
						})
						m.viewport.SetContent(m.renderConversation())
						m.viewport.GotoBottom()
						return m, nil
					}
					directPrompt := strings.TrimSpace(strings.TrimPrefix(input, parts[0]))
					if m.isBusy() {
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
							Content: ">> Tarea anterior interrumpida. Ejecutando nueva orden directamente...",
						})
					}
					m.promptHistory = append(m.promptHistory, directPrompt)
					m.promptHistoryIndex = len(m.promptHistory)
					if m.isWelcomeState() {
						m.entries = nil
					}
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
							Content: "[INFO] La cola de mensajes está vacía.",
						})
					} else {
						var sb strings.Builder
						sb.WriteString(fmt.Sprintf("[COLA] Mensajes en cola de espera (%d):\n", len(m.messageQueue)))
						for i, q := range m.messageQueue {
							vTag := ""
							if q.IsVoice {
								vTag = "[VOZ] "
							}
							sb.WriteString(fmt.Sprintf("  %d. %s\"%s\"\n", i+1, vTag, q.Prompt))
						}
						sb.WriteString(">> Usa /clearqueue para vaciarla, /now <orden> para ejecutar de inmediato o /cancel para detener la tarea activa.")
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
						Content: fmt.Sprintf("[LIMPIEZA] Cola de mensajes descartada con éxito (%d mensajes eliminados).", discarded),
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
			if m.isBusy() {
				m.promptHistory = append(m.promptHistory, input)
				m.promptHistoryIndex = len(m.promptHistory)
				qPos := m.EnqueuePrompt(input, false)
				m.textarea.Reset()
				m.entries = append(m.entries, ChatEntry{
					Role:    "system",
					Content: fmt.Sprintf("[COLA] Mensaje añadido a la cola [#%d]: \"%s\"\n(Se ejecutará automáticamente al finalizar la tarea actual. Usa /now <orden> para ejecutar de inmediato o /cancel para cancelar la actual).", qPos, input),
				})
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				return m, nil
			}

			// Registro de historial y entrada normal de usuario (StateIdle)
			m.promptHistory = append(m.promptHistory, input)
			m.promptHistoryIndex = len(m.promptHistory)
			if m.isWelcomeState() {
				m.entries = nil
			}
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
					roleContent = "[VOZ] " + nextPrompt.Prompt
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
				Content: fmt.Sprintf("[ERROR] %s", evt.Error),
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
					roleContent = "[VOZ] " + nextPrompt.Prompt
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
			Content: fmt.Sprintf("[ERROR] %v", msg.err),
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

		if m.isBusy() {
			m.promptHistory = append(m.promptHistory, prompt)
			m.promptHistoryIndex = len(m.promptHistory)
			qPos := m.EnqueuePrompt(prompt, true)
			m.entries = append(m.entries, ChatEntry{
				Role:    "system",
				Content: fmt.Sprintf("[COLA] Orden de voz añadida a la cola [#%d]: \"%s\" (se procesará automáticamente al terminar la tarea actual).", qPos, prompt),
			})
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, nil
		}

		m.promptHistory = append(m.promptHistory, prompt)
		m.promptHistoryIndex = len(m.promptHistory)
		m.entries = append(m.entries, ChatEntry{
			Role:    "user",
			Content: "[VOZ] " + prompt,
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

	case tea.MouseMsg:
		// Scroll con rueda del mouse en el viewport de chat
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.viewport.ScrollUp(3)
			return m, nil
		case tea.MouseButtonWheelDown:
			m.viewport.ScrollDown(3)
			return m, nil
		case tea.MouseButtonLeft:
			// Clic izquierdo: seleccionar fila en menús de texto basado en coordenada Y
			if m.state == StateStartMenu {
				// Aproximar la fila del menú según la posición Y del clic
				// Cada ítem ocupa aprox 2 líneas; el menú empieza en la línea ~12 de pantalla
				row := msg.Y - 12
				if row >= 0 && row/2 < 4 {
					m.menuIndex = row / 2
				}
				return m, nil
			}
			if m.state == StateChatHistory {
				// Cada ítem ocupa aprox 2 líneas; la lista empieza en la línea ~8
				row := msg.Y - 8
				if row >= 0 {
					idx := row / 2
					totalItems := len(m.getFilteredHistoryChats()) + 2
					if idx >= 0 && idx < totalItems {
						m.chatHistoryIndex = idx
					}
				}
				return m, nil
			}
			if m.state == StateSettingsMenu {
				row := msg.Y - 8
				if row >= 0 {
					idx := row / 2
					if idx >= 0 && idx < 7 {
						m.settingsIndex = idx
					}
				}
				return m, nil
			}
			// En chat activo con tarjeta: clic sobre opción la selecciona
			if (m.state == StateIdle || m.state == StateStreaming) && m.activeCard != nil {
				// Cada opción de la tarjeta ocupa 1 línea; la tarjeta empieza cerca de la parte superior del viewport
				// Aproximación: si el clic está en la mitad superior de la pantalla, mapear a opciones de la tarjeta
				optCount := len(m.activeCard.Options)
				if optCount > 0 {
					row := msg.Y - (m.height/2 - optCount)
					if row >= 0 && row < optCount {
						m.activeCard.SelectedIndex = row
						m.viewport.SetContent(m.renderConversation())
					}
				}
				return m, nil
			}
		}
		return m, nil
	}

	// Manejo del spinner (siempre se actualiza para no romper el ciclo Tick)
	var spinCmd tea.Cmd
	m.spinner, spinCmd = m.spinner.Update(msg)
	cmds = append(cmds, spinCmd)

	// Manejo del input de texto y scroll (solo cuando el chat está activo)
	if m.state != StateStartMenu && m.state != StateSettingsMenu {
		var taCmd tea.Cmd
		m.textarea, taCmd = m.textarea.Update(msg)
		cmds = append(cmds, taCmd)

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
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleSettingsEnter() (tea.Model, tea.Cmd) {
	switch m.settingsIndex {
	case 0:
		// 0: Abrir selector interactivo de proveedores LLM
		m.state = StateProviderMenu
		m.providerIndex = 0
		currentProv := ""
		if m.provider != nil {
			currentProv = strings.ToLower(m.provider.Name())
		} else if m.chat != nil && m.chat.Provider != "" {
			currentProv = strings.ToLower(m.chat.Provider)
		}
		for idx, cat := range providers.GetSupportedCatalog() {
			if strings.ToLower(cat.ID) == currentProv {
				m.providerIndex = idx
				break
			}
		}
		m.settingsNotice = ""
		return m, nil

	case 1:
		// 1: Abrir gestión interactiva de Claves API
		m.state = StateAPIKeySelect
		m.apiKeyIndex = 0
		m.settingsNotice = ""
		return m, nil

	case 2:
		// 2: Abrir editor interactivo de Perfil de Usuario
		m.state = StateProfileEdit
		m.profileInput.SetValue("")
		m.profileInput.Focus()
		m.settingsNotice = ""
		return m, textinput.Blink

	case 3:
		// 3: Alternar nivel de permisos SO
		switch m.permissionLevel {
		case "autonomous":
			m.permissionLevel = "supervised"
		case "supervised":
			m.permissionLevel = "sandboxed"
		default:
			m.permissionLevel = "autonomous"
		}
		m.settingsNotice = fmt.Sprintf("[OK] Nivel de permisos establecido en: %s", strings.ToUpper(m.permissionLevel))
		return m, nil

	case 4:
		// 4: Alternar escucha continua de voz
		out := m.handleSlashCommand("/voice")
		m.settingsNotice = out
		return m, nil

	case 5:
		// 5: Inspeccionar mapa de rutas en RAM
		count := 0
		if reg := system.DefaultPathRegistry(); reg != nil {
			count = len(reg.ListAll())
		}
		m.settingsNotice = fmt.Sprintf("[INFO] %d ubicaciones indexadas en RAM. Escribe /paths en el chat para el listado completo.", count)
		return m, nil

	case 6:
		// 6: Volver al Menú Principal
		m.state = StateStartMenu
		m.settingsNotice = ""
		return m, nil
	}

	return m, nil
}

func (m Model) handleProviderSelectEnter() (tea.Model, tea.Cmd) {
	catalog := providers.GetSupportedCatalog()
	if m.providerIndex >= len(catalog) {
		// Opción Volver a Configuraciones
		m.state = StateSettingsMenu
		m.settingsNotice = ""
		return m, nil
	}

	selected := catalog[m.providerIndex]
	targetID := selected.ID

	// Obtener o instanciar proveedor de forma segura (con o sin clave)
	p := providers.EnsureProvider(targetID)
	if p != nil {
		m.provider = p
	}

	defModel := selected.DefaultModel
	if p != nil && len(p.Models()) > 0 {
		defModel = p.Models()[0]
	}

	if m.chat == nil {
		m.chat = &models.Chat{
			Provider: targetID,
			Model:    defModel,
		}
	} else {
		m.chat.Provider = targetID
		m.chat.Model = defModel
	}

	hasKey := providers.GetProviderKey(targetID) != ""
	if selected.IsLocal {
		m.settingsNotice = fmt.Sprintf("[OK] Proveedor local activado: %s (modelo: %s)", strings.ToUpper(selected.DisplayName), defModel)
	} else if hasKey {
		m.settingsNotice = fmt.Sprintf("[OK] Proveedor activo cambiado a: %s (modelo: %s)", strings.ToUpper(selected.DisplayName), defModel)
	} else {
		m.settingsNotice = fmt.Sprintf("[SELECCIONADO] Proveedor activo: %s (modelo: %s). Sin clave configurada (puedes usarlo o configurarla con /key %s <api-key>)", strings.ToUpper(selected.DisplayName), defModel, targetID)
	}

	return m, nil
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

			// Si el chat no tiene un tema personalizado asignado, titularlo automáticamente con el tema del mensaje
			if m.chat.Name == "" || m.chat.Name == "TUI Session" || m.chat.Name == "Nueva Sesion" || strings.HasPrefix(m.chat.Name, "Sesion ") {
				topic := extractChatTopic(prompt)
				if topic != "" {
					m.chat.Name = topic
					_ = db.UpdateChat(m.chat)
				}
			}
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

			// Si el chat no tiene un tema personalizado asignado, titularlo automáticamente con el tema del mensaje
			if m.chat.Name == "" || m.chat.Name == "TUI Session" || m.chat.Name == "Nueva Sesion" || strings.HasPrefix(m.chat.Name, "Sesion ") {
				topic := extractChatTopic(prompt)
				if topic != "" {
					m.chat.Name = topic
					_ = db.UpdateChat(m.chat)
				}
			}
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
		return `[COMANDOS DISPONIBLES EN OZYASSIST TUI]
  /menu, /inicio    - Abre el menú interactivo retro de inicio
  /profile [texto]  - Consulta o actualiza la ficha del perfil de usuario
  /memories         - Consulta los hechos y preferencias aprendidas en memoria continua
  /remember <hecho> - Registra manualmente un hecho técnico o regla persistente
  /dream            - Ejecuta el subagente DREAMER para consolidar memoria y actualizar perfil
  /paths, /rutas    - Lista proyectos, repositorios y rutas indexadas en memoria RAM
  /scan             - Fuerza un escaneo universal de rutas y proyectos en segundo plano
  /cancel [all]     - Cancela la petición activa en curso (o 'all' para vaciar cola)
  /now <orden>      - Interrumpe la tarea actual y ejecuta la orden inmediatamente
  /queue, /cola     - Consulta los mensajes pendientes en la cola de espera
  /clearqueue       - Vacía la cola de mensajes pendientes
  /provider [nom]   - Consulta o cambia el proveedor LLM activo (cohere, groq, openai, etc.)
  /key <prov> <key> - Configura API Key (cohere, groq, openrouter, openai, deepseek, mistral, kilocode)
  /voice            - Alterna la escucha activa de voz ("Hey Ozy")
  /model [nombre]   - Consulta o cambia el proveedor/modelo actual
  /tools            - Lista todas las herramientas activas (SO + MCP + Memoria)
  /mcp [reload]     - Consulta o recarga los servidores MCP
  /perm [modo]      - Cambia nivel de permisos: autonomous | supervised | sandboxed
  /clear            - Limpia el historial de la pantalla (Ctrl+L)
  /new              - Crea una nueva sesión de chat limpia sin salir al menú
  /history          - Abre el historial de conversaciones guardadas
  /exit, /quit      - Cierra la aplicación (Ctrl+C)
Atajos: [Esc] para cancelar tarea • [Ctrl+T] alternar pensamiento • [Enter] encola si Ozy está ocupado`

	case "/menu", "/inicio", "/start":
		m.state = StateStartMenu
		m.settingsNotice = ""
		m.textarea.Blur()
		return ""

	case "/new", "/nuevo", "/nueva":
		_, cmd := m.handleNewChat()
		_ = cmd
		return "[NUEVO CHAT] Sesión nueva creada. Escribe tu primera instrucción."

	case "/history", "/historial", "/chats":
		_ = db.CleanupEmptyChats("")
		chats, err := db.ListChats()
		if err != nil {
			chats = nil
		}
		m.historyChats = chats
		m.chatHistoryIndex = 0
		m.historyConfirmDelete = ""
		m.historySearchInput.Reset()
		m.historySearchInput.Focus()
		m.state = StateChatHistory
		m.textarea.Blur()
		return ""

	case "/profile", "/perfil":
		if len(parts) == 1 {
			if db.DB == nil {
				return "[PERFIL] Base de datos no inicializada."
			}
			u, err := db.GetUser(db.DefaultUserID())
			if err != nil || u == nil || strings.TrimSpace(u.ProfileMd) == "" {
				return "[PERFIL] No hay perfil definido todavia. Usa /profile <descripcion> para configurar tu rol y proyectos."
			}
			return fmt.Sprintf("[PERFIL DEL USUARIO]\n%s\n\n(Usa '/profile <nuevo contenido>' para actualizar tu ficha)", strings.TrimSpace(u.ProfileMd))
		}
		newProfile := strings.TrimSpace(strings.TrimPrefix(cmdStr, parts[0]))
		if db.DB != nil {
			if err := db.UpdateUserProfile(db.DefaultUserID(), newProfile); err != nil {
				return fmt.Sprintf("[PERFIL] Error al actualizar perfil: %v", err)
			}
		}
		return "[PERFIL] Perfil de usuario actualizado y persistido con exito."

	case "/memories", "/memoria", "/recuerdos":
		store := memory.DefaultStore()
		if store == nil && db.DB != nil {
			store = memory.NewStore(db.DB)
		}
		if store == nil {
			return "[MEMORIA] Almacen de memoria no disponible."
		}
		ctxMem, cancelMem := context.WithTimeout(context.Background(), 2*time.Second)
		facts, err := store.GetRecentFacts(ctxMem, db.DefaultUserID(), 20)
		cancelMem()
		if err != nil || len(facts) == 0 {
			return "[MEMORIA] Aun no se han registrado recuerdos en la memoria persistente."
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("[MEMORIA CONTINUA] Recuerdos registrados (%d):\n", len(facts)))
		for i, f := range facts {
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s (relevancia: %.0f%%, consultas: %d)\n",
				i+1, strings.ToUpper(string(f.Category)), f.Content, f.Confidence*100, f.AccessCount))
		}
		sb.WriteString("\nUsa /remember <hecho> para registrar un nuevo detalle tecnico o preferencia.")
		return sb.String()

	case "/remember", "/recordar":
		if len(parts) < 2 {
			return "[MEMORIA] Uso: /remember <hecho o preferencia a recordar>\nEj: /remember En crmgeofal la API corre en el puerto 4000 con Go 1.22"
		}
		factContent := strings.TrimSpace(strings.TrimPrefix(cmdStr, parts[0]))
		store := memory.DefaultStore()
		if store == nil && db.DB != nil {
			store = memory.NewStore(db.DB)
		}
		if store == nil {
			return "[MEMORIA] Almacen de memoria no disponible."
		}
		ctxMem, cancelMem := context.WithTimeout(context.Background(), 2*time.Second)
		err := store.UpsertFact(ctxMem, db.DefaultUserID(), memory.CategoryPreference, factContent, 1.0)
		cancelMem()
		if err != nil {
			return fmt.Sprintf("[MEMORIA] Error al guardar recuerdo: %v", err)
		}
		return fmt.Sprintf("[MEMORIA] Recordado y persistido con exito: \"%s\"", factContent)

	case "/dream", "/sonar", "/soñar", "/consolidar":
		dreamer := memory.DefaultDreamer()
		if dreamer == nil {
			return "[DREAMING] Subagente DREAMER no inicializado."
		}
		dreamer.DreamAsync(context.Background(), db.DefaultUserID(), nil)
		return "[DREAMING] Proceso de consolidacion de memoria continua iniciado en segundo plano por el subagente DREAMER."

	case "/paths", "/rutas", "/proyectos":
		reg := system.DefaultPathRegistry()
		if reg == nil {
			return "[RUTAS] Registro de rutas no inicializado."
		}
		paths := reg.ListAll()
		if len(paths) == 0 {
			return "[RUTAS] No hay rutas indexadas todavia. Usa /scan para realizar un escaneo del sistema."
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("[RUTAS INDEXADAS EN RAM] Total: %d ubicaciones\n", len(paths)))
		limit := 25
		for i, p := range paths {
			if i >= limit {
				sb.WriteString(fmt.Sprintf("  ... y %d rutas mas. (Consulta un proyecto directamente por su nombre)\n", len(paths)-limit))
				break
			}
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s -> %s\n", i+1, strings.ToUpper(string(p.Category)), p.Name, p.FullPath))
		}
		sb.WriteString("\nUsa /scan para forzar un re-escaneo del sistema de archivos en segundo plano.")
		return sb.String()

	case "/scan", "/escanear":
		reg := system.DefaultPathRegistry()
		if reg == nil && db.DB != nil {
			system.InitDefaultPathRegistry(db.DB)
			reg = system.DefaultPathRegistry()
		}
		if reg == nil {
			return "[RUTAS] Registro de rutas no inicializado."
		}
		go func() {
			_, _ = reg.RefreshScan(context.Background())
		}()
		return "[RUTAS] Escaneo universal del sistema anfitrion iniciado en segundo plano. Los proyectos se indexan en memoria RAM."

	case "/cancel", "/stop", "/abort", "/cancelar", "/parar":
		clearAll := len(parts) > 1 && strings.ToLower(parts[1]) == "all"
		if m.isBusy() {
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
				return fmt.Sprintf("[ALERTA] Petición activa cancelada y %d mensajes en cola descartados.", discarded)
			}
			if len(m.messageQueue) > 0 {
				return fmt.Sprintf("[ALERTA] Petición cancelada. (Quedan %d mensajes en cola. Usa /queue para verlos o /clearqueue para vaciarla).", len(m.messageQueue))
			}
			return "[ALERTA] Petición cancelada con éxito."
		}
		if clearAll || len(m.messageQueue) > 0 {
			discarded := m.ClearQueue()
			return fmt.Sprintf("[LIMPIEZA] Cola de mensajes vaciada (%d descartados).", discarded)
		}
		return "[INFO] No hay ninguna petición activa ni mensajes en cola."

	case "/queue", "/cola":
		if len(m.messageQueue) == 0 {
			return "[INFO] La cola de mensajes está vacía."
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("[COLA] Mensajes en cola de espera (%d):\n", len(m.messageQueue)))
		for i, q := range m.messageQueue {
			vTag := ""
			if q.IsVoice {
				vTag = "[VOZ] "
			}
			sb.WriteString(fmt.Sprintf("  %d. %s\"%s\"\n", i+1, vTag, q.Prompt))
		}
		sb.WriteString(">> Usa /clearqueue para vaciarla, /now <orden> para ejecutar de inmediato o /cancel para detener la tarea activa.")
		return sb.String()

	case "/clearqueue", "/dropqueue", "/vaciarcola":
		discarded := m.ClearQueue()
		return fmt.Sprintf("[LIMPIEZA] Cola de mensajes descartada con éxito (%d mensajes eliminados).", discarded)

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
				return fmt.Sprintf("[ALERTA] La clave '%s...' no parece ser válida de Groq (debe comenzar con 'gsk_').", targetKey[:minLen])
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
					voiceMsg = "\n[VOZ] Motor de voz 'Hey Ozy' ACTIVADO y escuchando."
				}
			}
			return fmt.Sprintf(">> Groq API Key configurada con éxito.\n[OK] Guardada en .env\n[OK] Proveedor activo: Groq (llama-3.3-70b-versatile @ 800 tokens/s)%s", voiceMsg)
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

		return fmt.Sprintf(">> Abriendo la consola de Groq en %s:\n"+
			"   URL: %s\n"+
			"1. Inicia sesión con 1 clic (Continuar con Google).\n"+
			"2. Haz clic en 'Create API Key' y copia la clave generada.\n"+
			"3. Vuelve a esta terminal y escribe '/groq' (o por voz: 'guarda mi clave'). Ozy la leerá del portapapeles.",
			chromeInfo, groqURL)

	case "/tools":
		var sb strings.Builder
		sb.WriteString(">> Herramientas de Control del SO disponibles:\n")
		for _, t := range agent.AgentTools {
			sb.WriteString(fmt.Sprintf("  • %-22s: %s\n", t.Name, t.Description))
		}
		mcpTools := mcp.DefaultRegistry.GetAllTools()
		if len(mcpTools) > 0 {
			sb.WriteString("\n>> Herramientas MCP Externas Conectadas:\n")
			for name, t := range mcpTools {
				sb.WriteString(fmt.Sprintf("  • %-26s: %s\n", name, t.Description))
			}
		}
		return sb.String()

	case "/thinking", "/thought":
		m.showThinking = !m.showThinking
		if m.showThinking {
			return "[PENSAMIENTO] Hilo de pensamiento: VISIBLE y desplegado. (Presiona Ctrl+T o /thinking para ocultar)"
		}
		return "[PENSAMIENTO] Hilo de pensamiento: PLEGADO y oculto. (Presiona Ctrl+T o /thinking para desplegar)"

	case "/mcp":
		if len(parts) > 1 && strings.ToLower(parts[1]) == "reload" {
			cfgPath := mcp.FindDefaultConfigFile()
			if cfgPath == "" {
				return "[ALERTA] No se encontró ningún archivo mcp_servers.json en las rutas canónicas (backend/mcp_servers.json o ~/.ozy/mcp_servers.json)."
			}
			configs, err := mcp.LoadConfigFile(cfgPath)
			if err != nil {
				return fmt.Sprintf("[ERROR] Error leyendo %s: %v", cfgPath, err)
			}
			if err := mcp.DefaultRegistry.Reload(context.Background(), configs); err != nil {
				return fmt.Sprintf("[ERROR] Error recargando servidores MCP: %v", err)
			}
			statuses := mcp.DefaultRegistry.GetServerStatus()
			totalTools := len(mcp.DefaultRegistry.GetAllTools())
			return fmt.Sprintf("[OK] Servidores MCP recargados desde %s (%d servidores, %d herramientas activas).", cfgPath, len(statuses), totalTools)
		}

		statuses := mcp.DefaultRegistry.GetServerStatus()
		if len(statuses) == 0 {
			cfgPath := mcp.FindDefaultConfigFile()
			hint := "Crea un archivo 'mcp_servers.json' para conectar servidores estándar (Claude Desktop / Cursor)."
			if cfgPath != "" {
				hint = fmt.Sprintf("Archivo detectado: %s (sin servidores activos o con errores).", cfgPath)
			}
			return fmt.Sprintf("[INFO] No hay servidores MCP conectados actualmente.\n>> %s\nUsa '/mcp reload' para recargar en caliente.", hint)
		}

		var sb strings.Builder
		sb.WriteString(">> Servidores MCP (Model Context Protocol) Conectados:\n")
		for _, s := range statuses {
			icon := "[OK]"
			if s.Status == "error" {
				icon = "[ERR]"
			} else if s.Status == "stopped" {
				icon = "[--]"
			}
			sb.WriteString(fmt.Sprintf("  %-5s %-14s [%s] — %d herramientas (%s)\n", icon, s.Name, s.Status, s.ToolCount, s.Command))
			if s.Error != "" {
				sb.WriteString(fmt.Sprintf("     [ERROR]: %s\n", s.Error))
			}
		}
		sb.WriteString("\n>> Usa '/mcp reload' para recargar en caliente tras editar mcp_servers.json.")
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
				return fmt.Sprintf("[ALERTA] Proveedor '%s' no reconocido o sin registrar.\n"+
					">> Usa '/provider' sin argumentos para ver los proveedores disponibles.", target)
			}

			key := providers.GetProviderKey(target)
			if key == "" && target != "lmstudio" && target != "ollama" && target != "local" {
				return fmt.Sprintf("[ALERTA] El proveedor '%s' no tiene una API key configurada.\n"+
					">> Configúrala escribiendo: /key %s <tu-api-key>", target, target)
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

			return fmt.Sprintf("[OK] Proveedor activo cambiado a: %s\n[OK] Modelo predeterminado: %s", target, defaultModel)
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
			{"kilocode", "KiloCode Gateway", "+500 modelos: Claude, GPT-5, Gemini, DeepSeek", "kilo/anthropic/claude-sonnet-4-5"},
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
		sb.WriteString(">> Proveedores de IA disponibles en OzyAssist:\n")
		for _, p := range allProviders {
			key := providers.GetProviderKey(p.id)
			hasConfig := key != "" || (p.id == "lmstudio" || p.id == "ollama")

			icon := "[--]"
			statusText := "Sin configurar"
			if hasConfig {
				icon = "[..]"
				statusText = "Clave Añadida (sin validar)"
			}
			if p.id == currProv {
				icon = "[*]"
				statusText = "ACTIVO"
			}

			sb.WriteString(fmt.Sprintf("  %-5s %-14s [%s] — %s (%s)\n", icon, p.label, statusText, p.desc, p.defModel))
		}

		sb.WriteString("\n>> Para cambiar de proveedor activo escribe:\n")
		sb.WriteString("   /provider cohere\n")
		sb.WriteString("   /provider groq\n")
		sb.WriteString("   /provider openai\n")
		sb.WriteString(">> Para configurar la clave de un proveedor usa:\n")
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
				return fmt.Sprintf("[OK] Proveedor: %s (Modelo: %s)", target, m.chat.Model)
			}
			// 2. Si es un nombre de modelo directo (ej: /model deepseek/deepseek-chat)
			if m.chat != nil {
				m.chat.Model = target
				return fmt.Sprintf("[OK] Modelo establecido a: %s", target)
			}
			return fmt.Sprintf("[ALERTA] No se pudo asignar el modelo: %s", target)
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
				return fmt.Sprintf("[OK] Nivel de permisos establecido en: %s", mode)
			}
			return "Modos válidos: autonomous | supervised | sandboxed"
		}
		return fmt.Sprintf("Nivel de permisos actual: %s", m.permissionLevel)

	case "/voice":
		if activeVoiceController == nil {
			m.voiceEnabled = !m.voiceEnabled
			if m.voiceEnabled {
				m.systemStatus = "Escuchando Wake Word ('Hey Ozy')..."
				return "[VOZ] Escucha nativa de Wake Word 'Hey Ozy' ACTIVADA."
			}
			m.systemStatus = "Listo para actuar"
			return "[SILENCIO] Escucha nativa de voz DESACTIVADA."
		}

		if m.voiceEnabled {
			activeVoiceController.Stop()
			m.voiceEnabled = false
			m.systemStatus = "Listo para actuar"
			return "[SILENCIO] Escucha nativa de voz DESACTIVADA."
		}

		if !activeVoiceController.HasSTT() {
			return `[ALERTA] Para activar la escucha por voz ("Hey Ozy"), necesitas un motor de transcripción (STT).
>> Configura una API Key gratuita de Groq (~150ms) escribiendo:
   /key groq <tu-api-key>
   (Obtén una gratis en https://console.groq.com/keys)
O con OpenAI Whisper:
   /key openai <tu-api-key>`
		}

		if err := activeVoiceController.Start(); err != nil {
			return fmt.Sprintf("[ERROR] Error al arrancar la escucha de voz: %v", err)
		}
		m.voiceEnabled = true
		m.systemStatus = "Escuchando Wake Word ('Hey Ozy')..."
		return "[VOZ] Escucha nativa de Wake Word 'Hey Ozy' ACTIVADA."

	case "/clear":
		m.entries = nil
		m.viewport.SetContent(m.renderConversation())
		m.viewport.GotoTop()
		return ""

	case "/key":
		if len(parts) < 3 {
			return `>> Uso de /key:
  /key cohere <tu-api-key>      (Command R+ de Cohere)
  /key groq <tu-api-key>        (Voz 'Hey Ozy' ultrarrápida gratis)
  /key openai <tu-api-key>
  /key openrouter <tu-api-key>
  /key deepseek <tu-api-key>
  /key anthropic <tu-api-key>
  /key mistral <tu-api-key>     (mistral-large-latest, codestral, etc.)
  /key kilocode <jwt-token>     (Gateway 500+ modelos: Claude, GPT, Gemini...)
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
			return fmt.Sprintf("[OK] Proveedor 'cohere' configurado y activado (Modelo: %s).", modelName)
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
					return "[OK] API Key de Groq guardada en .env. Motor de voz 'Hey Ozy' ACTIVADO y escuchando."
				}
			}
			return "[OK] API Key de Groq guardada en .env. Activa la voz con /voice."
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
				extraHint = "\n>> Nota: Esta clave parece de Cohere. Puedes activarla como proveedor nativo con: /key cohere " + keyVal + " o /provider cohere"
			}
			return fmt.Sprintf("[OK] Proveedor 'openai' configurado y guardado en .env (disponible para LLM y Whisper STT).%s", extraHint)
		}

		if targetProv == "local" || targetProv == "lmstudio" || targetProv == "ollama" && strings.HasPrefix(keyVal, "http") {
			providers.RegisterLocalHostURL(keyVal)
			_ = providers.SaveConfigKey("LMSTUDIO_URL", keyVal)
			if p, err := providers.Get("lmstudio"); err == nil {
				m.provider = p
			}
			return fmt.Sprintf("[OK] Endpoint local actualizado y guardado: %s", keyVal)
		}

		if targetProv == "kilocode" || targetProv == "kilo" {
			_ = providers.SaveConfigKey("KILOCODE_API_KEY", keyVal)
			_ = os.Setenv("KILOCODE_API_KEY", keyVal)
			providers.RegisterProviderKey("kilocode", keyVal)
			if p, err := providers.Get("kilocode"); err == nil {
				m.provider = p
				if m.chat != nil {
					m.chat.Provider = "kilocode"
					if len(p.Models()) > 0 {
						m.chat.Model = p.Models()[0]
					}
				}
			}
			modelName := "kilo/anthropic/claude-sonnet-4-5"
			if m.chat != nil && m.chat.Model != "" {
				modelName = m.chat.Model
			}
			return fmt.Sprintf(">> Proveedor 'KiloCode Gateway' configurado y activado (Modelo: %s).\n"+
				"[OK] Token JWT guardado en .env como KILOCODE_API_KEY\n"+
				">> Gateway unificado con +500 modelos (Claude, GPT, Gemini, DeepSeek, Mistral)\n"+
				"   Cambia modelo con: /model kilo/anthropic/claude-opus-4-5\n"+
				"   O: /model kilo/openai/gpt-4o", modelName)
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
			return fmt.Sprintf(">> Proveedor 'mistral' configurado y activado (Modelo: %s).\n"+
				"[OK] Clave guardada en .env\n"+
				">> Modelos disponibles: mistral-large-latest, codestral-latest, open-mixtral-8x22b\n"+
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
			return fmt.Sprintf("[OK] Proveedor '%s' configurado y guardado en .env (Modelo: %s).", targetProv, modelName)
		}
		return fmt.Sprintf("[ALERTA] Proveedor '%s' no reconocido. Disponibles: groq, openrouter, openai, deepseek, anthropic, mistral, kilocode, local", targetProv)

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

// handleNewChat crea una nueva sesión de chat en SQLite, limpia el historial
// en pantalla con un mensaje de bienvenida y activa el estado Idle listo para escribir.
// Si el chat anterior no contenía ningún mensaje, se elimina de la base de datos.
func (m Model) handleNewChat() (tea.Model, tea.Cmd) {
	// Limpiar sesión anterior si estaba vacía
	if m.chat != nil && m.chat.ID != "" {
		count, _ := db.CountMessagesByChat(m.chat.ID)
		if count == 0 {
			_ = db.DeleteChat(m.chat.ID)
		}
	}
	_ = db.CleanupEmptyChats("")

	newChat := &models.Chat{
		ID:        uuid.NewString(),
		UserID:    db.DefaultUserID(),
		Name:      "", // Inicialmente sin tema; el código (#shortID) se muestra hasta que se hable
		Mode:      "chat",
		CreatedAt: time.Now(),
	}
	if m.provider != nil {
		newChat.Provider = m.provider.Name()
		if len(m.provider.Models()) > 0 {
			newChat.Model = m.provider.Models()[0]
		}
	}
	_ = db.CreateChat(newChat)
	m.chat = newChat

	welcomeContent := "¡Hola! Soy OzyAssist, tu asistente autónomo de escritorio, código y cowork para Windows.\n\nEstoy conectado y listo con arquitectura Zero-Docker, memoria continua y herramientas de sistema.\nEscribe libremente tu instrucción o consulta para comenzar."

	m.entries = []ChatEntry{
		{
			Role:    "assistant",
			Content: welcomeContent,
		},
	}
	m.activeCard = nil
	m.promptHistory = nil
	m.promptHistoryIndex = -1
	m.state = StateIdle
	m.textarea.Focus()
	m.textarea.Reset()
	if m.ready {
		m.viewport.SetContent(m.renderConversation())
		m.viewport.GotoBottom()
	}
	return m, textarea.Blink
}

// extractChatTopic extrae un tema conciso del primer mensaje del usuario para titular la sesión
func extractChatTopic(prompt string) string {
	cleaned := strings.TrimSpace(prompt)
	if cleaned == "" {
		return ""
	}
	// Si es comando slash con argumento (ej: /now haz tal cosa), tomar el argumento
	if strings.HasPrefix(cleaned, "/") {
		fields := strings.Fields(cleaned)
		if len(fields) > 1 {
			cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, fields[0]))
		}
	}
	// Tomar la primera línea si hay saltos de línea
	if idx := strings.IndexByte(cleaned, '\n'); idx != -1 {
		cleaned = strings.TrimSpace(cleaned[:idx])
	}
	if len(cleaned) == 0 {
		return ""
	}
	// Capitalizar la primera letra
	runes := []rune(cleaned)
	runes[0] = unicode.ToUpper(runes[0])
	if len(runes) > 40 {
		return string(runes[:37]) + "..."
	}
	return string(runes)
}

// max retorna el mayor de dos enteros.
// Compatible con Go <1.21 que no tiene builtin max para int.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
