package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	thoughtBlockRegex  = regexp.MustCompile(`(?s)<thought>.*?</thought>`)
	thinkBlockRegex    = regexp.MustCompile(`(?s)<think>.*?</think>`)
	jsonCodeBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*\\{[^{}]*?\"(?:tool|name|action)\"[^{}]*?\\}\\s*```")
	specialTokenRegex  = regexp.MustCompile(`<\|.*?\|>`)
	multiNewlineRegex  = regexp.MustCompile(`\n{3,}`)
)

// cleanAssistantText strips thought blocks, think tags, tokenizer artifacts and redundant json markdown
func cleanAssistantText(text string) string {
	cleaned := thoughtBlockRegex.ReplaceAllString(text, "")
	cleaned = thinkBlockRegex.ReplaceAllString(cleaned, "")
	cleaned = jsonCodeBlockRegex.ReplaceAllString(cleaned, "")
	cleaned = specialTokenRegex.ReplaceAllString(cleaned, "")

	// If streaming is currently inside an unclosed <thought> or <think>
	if idx := strings.LastIndex(cleaned, "<thought>"); idx != -1 {
		cleaned = cleaned[:idx]
	}
	if idx := strings.LastIndex(cleaned, "<think>"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	cleaned = multiNewlineRegex.ReplaceAllString(cleaned, "\n\n")

	return strings.TrimSpace(cleaned)
}

// wrapLine divide una línea de texto para que no exceda maxWidth columnas,
// preservando sangría inicial y dividiendo palabras largas si es necesario.
func wrapLine(line string, maxWidth int) []string {
	if maxWidth < 20 {
		maxWidth = 70
	}
	line = strings.TrimRight(line, "\r")
	runes := []rune(line)
	if len(runes) <= maxWidth {
		return []string{line}
	}

	// Detectar sangría de espacios o tabuladores iniciales
	indentCount := 0
	for _, r := range runes {
		if r == ' ' || r == '\t' {
			indentCount++
		} else {
			break
		}
	}
	indent := string(runes[:indentCount])
	if indentCount >= maxWidth-10 {
		indent = ""
	}

	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{line}
	}

	var result []string
	var current strings.Builder
	currentLen := 0

	for i, word := range words {
		wordRunes := []rune(word)
		wordLen := len(wordRunes)

		// Manejo de palabras continuas más largas que maxWidth (URLs, rutas, hashes)
		if wordLen > maxWidth {
			if currentLen > 0 {
				result = append(result, current.String())
				current.Reset()
				currentLen = 0
			}
			remRunes := wordRunes
			for len(remRunes) > maxWidth {
				result = append(result, string(remRunes[:maxWidth]))
				remRunes = remRunes[maxWidth:]
			}
			if len(remRunes) > 0 {
				current.WriteString(string(remRunes))
				currentLen = len(remRunes)
			}
			continue
		}

		if currentLen == 0 {
			if len(result) > 0 && indent != "" {
				current.WriteString(indent)
				currentLen += len([]rune(indent))
			} else if i == 0 && indent != "" {
				current.WriteString(indent)
				currentLen += len([]rune(indent))
			}
			current.WriteString(word)
			currentLen += wordLen
		} else if currentLen+1+wordLen <= maxWidth {
			current.WriteString(" ")
			current.WriteString(word)
			currentLen += 1 + wordLen
		} else {
			result = append(result, current.String())
			current.Reset()
			if indent != "" {
				current.WriteString(indent)
				currentLen = len([]rune(indent))
			} else {
				currentLen = 0
			}
			current.WriteString(word)
			currentLen += wordLen
		}
	}

	if currentLen > 0 {
		result = append(result, current.String())
	}

	return result
}

// wrapContent procesa un bloque de texto multilínea y aplica wrapLine a cada línea.
func wrapContent(text string, maxWidth int) []string {
	if maxWidth < 20 {
		maxWidth = 70
	}
	var result []string
	lines := strings.Split(text, "\n")
	for _, l := range lines {
		wrapped := wrapLine(l, maxWidth)
		result = append(result, wrapped...)
	}
	return result
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Iniciando OzyAssist TUI..."
	}

	var sb strings.Builder

	// 1. Header superior
	sb.WriteString(m.renderHeader())
	sb.WriteString("\n")

	// 2. Historial de conversación (Viewport)
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	// 3. Footer inferior (Status + Input box + Hints)
	sb.WriteString(m.renderFooter())

	return sb.String()
}

func (m Model) renderHeader() string {
	asciiLogo := lipgloss.NewStyle().Foreground(lipgloss.Color("#d1f107")).Bold(true).Render("   .··'¯'··.   \n  :  .-.  :  OZY\n  :  '-'  :  ASSIST\n   '··._.··'   ")

	modelName := "desconocido"
	if m.chat != nil && m.chat.Model != "" {
		modelName = m.chat.Model
	} else if m.provider != nil {
		modelName = m.provider.Name()
	}

	displayModel := modelName
	if m.width > 0 && m.width < 95 && strings.Contains(displayModel, "/") {
		parts := strings.Split(displayModel, "/")
		displayModel = parts[len(parts)-1]
	}
	if len(displayModel) > 22 {
		displayModel = displayModel[:20] + ".."
	}

	provName := "sin proveedor"
	if m.provider != nil {
		provName = m.provider.Name()
	} else if m.chat != nil && m.chat.Provider != "" {
		provName = m.chat.Provider
	}

	modelTagText := fmt.Sprintf(" [LLM: %s: %s] ", provName, displayModel)
	if m.width > 0 && m.width < 90 {
		modelTagText = fmt.Sprintf(" %s:%s ", provName, displayModel)
	}
	modelTag := TagStyle.Render(modelTagText)

	permText := fmt.Sprintf(" Permisos: %s ", m.permissionLevel)
	if m.width > 0 && m.width < 85 {
		permText = fmt.Sprintf(" %s ", m.permissionLevel)
	}
	permTag := TagStyle.Render(permText)

	voiceText := " [VOZ: OFF] "
	voiceTag := TagStyle.Render(voiceText)
	if m.voiceEnabled {
		if m.width > 0 && m.width < 90 {
			voiceText = " [VOZ: ON] "
		} else {
			voiceText = " [VOZ: ACTIVO] "
		}
		voiceTag = TagActiveStyle.Render(voiceText)
	}

	tags := []string{modelTag, " ", permTag, " ", voiceTag}
	if len(m.messageQueue) > 0 {
		queueText := fmt.Sprintf(" [COLA: %d] ", len(m.messageQueue))
		tags = append(tags, " ", QueueBadgeStyle.Render(queueText))
	}
	statsLine := lipgloss.JoinHorizontal(lipgloss.Center, tags...)
	
	header := lipgloss.JoinVertical(lipgloss.Center, asciiLogo, statsLine)
	
	if m.width > 0 {
		return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(header)
	}
	return header
}

func (m Model) renderConversation() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}

	if m.isWelcomeState() {
		return m.renderRetroWelcomeHero(convWidth)
	}

	var sb strings.Builder

	for _, entry := range m.entries {
		switch entry.Role {
		case "user":
			sb.WriteString(UserStyle.Render("❯ TÚ: "))
			maxUserW := convWidth - 10
			if maxUserW < 20 {
				maxUserW = 20
			}
			lines := wrapContent(entry.Content, maxUserW)
			for i, l := range lines {
				if i > 0 {
					sb.WriteString("      ")
				}
				sb.WriteString(l)
				sb.WriteString("\n")
			}
			sb.WriteString("\n")

		case "assistant":
			// Renderizar pensamiento (Thinking / Chain of Thought) si existe
			if entry.Thinking != "" {
				if m.showThinking {
					sb.WriteString(MutedStyle.Render("[PENSAMIENTO / RAZONAMIENTO — Presiona Ctrl+T para plegar]:\n"))
					thinkLines := strings.Split(strings.TrimSpace(entry.Thinking), "\n")
					maxThinkW := convWidth - 8
					if maxThinkW < 20 {
						maxThinkW = 20
					}
					for _, tl := range thinkLines {
						wrapped := wrapLine(tl, maxThinkW)
						for _, wl := range wrapped {
							sb.WriteString(MutedStyle.Render("│ " + wl))
							sb.WriteString("\n")
						}
					}
					sb.WriteString("\n")
				} else {
					sb.WriteString(MutedStyle.Render("[PENSAMIENTO OCULTO — Presiona Ctrl+T para desplegar]\n\n"))
				}
			}

			cleaned := cleanAssistantText(entry.Content)
			if cleaned != "" {
				sb.WriteString(AssistantStyle.Render("OZY: "))
				maxAssistantW := convWidth - 10
				if maxAssistantW < 20 {
					maxAssistantW = 20
				}
				
				// Renderizar Markdown con Glamour
				r, _ := glamour.NewTermRenderer(
					glamour.WithAutoStyle(),
					glamour.WithWordWrap(maxAssistantW),
				)
				rendered, err := r.Render(cleaned)
				if err == nil {
					// Eliminar salto de línea final extra de glamour
					rendered = strings.TrimRight(rendered, "\r\n")
					
					// Añadir sangría para alinear con "⚡ OZY: "
					lines := strings.Split(rendered, "\n")
					for i, l := range lines {
						if i > 0 {
							sb.WriteString("       ")
						}
						sb.WriteString(l)
						sb.WriteString("\n")
					}
					sb.WriteString("\n")
				} else {
					lines := wrapContent(cleaned, maxAssistantW)
					for i, l := range lines {
						if i > 0 {
							sb.WriteString("       ")
						}
						sb.WriteString(l)
						sb.WriteString("\n")
					}
					sb.WriteString("\n")
				}
			}

		case "system":
			maxSysW := convWidth - 4
			if maxSysW < 20 {
				maxSysW = 20
			}
			lines := wrapContent(entry.Content, maxSysW)
			for _, l := range lines {
				sb.WriteString(SystemStyle.Render(l))
				sb.WriteString("\n")
			}
			sb.WriteString("\n")

		case "tool":
			badge := ToolResultSuccessStyle.Render(fmt.Sprintf(" ✓ %s ", entry.ToolName))
			if !entry.ToolSuccess {
				badge = ToolResultErrorStyle.Render(fmt.Sprintf(" ✗ %s (Error) ", entry.ToolName))
			}
			dur := ""
			if entry.DurationMs > 0 {
				dur = fmt.Sprintf(" (%dms)", entry.DurationMs)
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", badge, MutedStyle.Render(dur)))

			// Formatear salida con sangría y word-wrapping garantizado
			maxToolWidth := convWidth - 6
			if maxToolWidth < 25 {
				maxToolWidth = 25
			}
			rawLines := strings.Split(strings.TrimSpace(entry.Content), "\n")
			for _, rl := range rawLines {
				wrapped := wrapLine(rl, maxToolWidth)
				for _, wl := range wrapped {
					sb.WriteString(ToolContentStyle.Render("│ " + wl))
					sb.WriteString("\n")
				}
			}
			sb.WriteString("\n")
		}
	}

	// Si hay streaming activo en este momento
	if len(m.currentStream) > 0 {
		cleaned := cleanAssistantText(m.currentStream)
		if cleaned != "" {
			sb.WriteString(AssistantStyle.Render("OZY: "))
			maxStreamW := convWidth - 10
			if maxStreamW < 20 {
				maxStreamW = 20
			}
			
			// Renderizar Markdown con Glamour
			r, _ := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(maxStreamW),
			)
			rendered, err := r.Render(cleaned)
			if err == nil {
				rendered = strings.TrimRight(rendered, "\r\n")
				lines := strings.Split(rendered, "\n")
				for i, l := range lines {
					if i > 0 {
						sb.WriteString("       ")
					}
					sb.WriteString(l)
					if i == len(lines)-1 {
						sb.WriteString("▌")
					}
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			} else {
				lines := wrapContent(cleaned, maxStreamW)
				for i, l := range lines {
					if i > 0 {
						sb.WriteString("       ")
					}
					sb.WriteString(l)
					if i == len(lines)-1 {
						sb.WriteString("▌")
					}
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString(MutedStyle.Render("⚡ OZY: (Razonando...) ▌\n\n"))
		}
	}

	// Si hay una herramienta ejecutándose en este momento
	if m.activeToolName != "" {
		badge := ToolBadgeStyle.Render(fmt.Sprintf(" ⚙️ Ejecutando %s... ", m.activeToolName))
		sb.WriteString(fmt.Sprintf("%s %s\n", badge, m.spinner.View()))
		if m.activeToolInput != "" {
			maxInputW := convWidth - 14
			if maxInputW < 20 {
				maxInputW = 20
			}
			wrapped := wrapLine(m.activeToolInput, maxInputW)
			for i, wl := range wrapped {
				if i == 0 {
					sb.WriteString(ToolContentStyle.Render("│ Params: " + wl))
				} else {
					sb.WriteString(ToolContentStyle.Render("│         " + wl))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m Model) renderFooter() string {
	var sb strings.Builder

	// Línea de estado con spinner y contador de cola si aplica
	statusLine := m.systemStatus
	if m.state != StateIdle {
		statusLine = fmt.Sprintf("%s %s", m.spinner.View(), m.systemStatus)
	}
	if len(m.messageQueue) > 0 {
		statusLine = fmt.Sprintf("%s  •  [COLA: %d en espera]", statusLine, len(m.messageQueue))
	}
	maxStatusW := m.width - 4
	if maxStatusW > 10 && len([]rune(statusLine)) > maxStatusW {
		runes := []rune(statusLine)
		statusLine = string(runes[:maxStatusW-3]) + "..."
	}
	sb.WriteString(StatusBarStyle.Render(statusLine))
	sb.WriteString("\n")

	// Caja de texto con borde neon o ámbar (si está encolando órdenes)
	borderStyle := InputBorderStyle
	if m.state != StateIdle {
		borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorQueue)
	}
	sb.WriteString(borderStyle.Render(m.textarea.View()))
	sb.WriteString("\n")

	// Hints de atajos contextuales
	var hintsText string
	if m.state != StateIdle {
		hintsText = "  [Enter] Encolar  •  [/now <orden>] Enviar directo  •  [/cancel | Esc] Cancelar  •  [/queue] Ver cola"
		if m.width > 0 && m.width < 95 {
			hintsText = "  [Enter] Encolar  •  [/now <orden>] Directo  •  [Esc] Cancelar"
		}
	} else if len(m.messageQueue) > 0 {
		hintsText = fmt.Sprintf("  [Enter] Enviar  •  [COLA: %d] (/queue)  •  [/clearqueue] Vaciar  •  [Ctrl+C] Salir", len(m.messageQueue))
	} else {
		hintsText = "  [Enter] Enviar  •  [Ctrl+C] Salir  •  [Ctrl+T] Pensamiento  •  [Ctrl+L] Limpiar  •  [/help] Comandos"
		if m.width > 0 && m.width < 85 {
			hintsText = "  [Enter] Enviar  •  [Ctrl+T] Pensamiento  •  [/help] Ayuda"
		}
	}
	hints := MutedStyle.Render(hintsText)
	sb.WriteString(hints)

	return sb.String()
}

func (m Model) renderRetroWelcomeHero(convWidth int) string {
	if convWidth < 40 {
		convWidth = 40
	}
	innerWidth := convWidth - 6
	if innerWidth > 82 {
		innerWidth = 82
	}

	var sb strings.Builder
	sb.WriteString("\n")

	// 1. Banner Retro ASCII de OZYASIST
	logoAscii := `   ___  _______   _____   _   ___ ___ ____ _____ 
  / _ \/ _  /\ \ / / _ | /_\ / __|_ _/ ___|_   _|
 | | | \// /  \ V / __ |/ _ \\__ \| |\___ \ | |  
 | |_| |/ //\  | / /_/ / ___ \__) | | ___) || |  
  \___//___/   |_| \__,_/_/   \_\___/___|____/ |_|`

	renderedLogo := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render(logoAscii)

	subHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Render("OZYASIST >> OS CONTROL & COWORK AGENT // HERMES CORE v2.6")

	headerBlock := lipgloss.JoinVertical(lipgloss.Center, renderedLogo, "", subHeader)
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(headerBlock))
	sb.WriteString("\n\n")

	// 2. Panel de Diagnóstico Estilo BIOS / 80s-90s Terminal
	diagBorder := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorPrimary).
		Background(ColorContainer).
		Padding(0, 1)

	var diagLines []string
	tagStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary)

	diagLines = append(diagLines, fmt.Sprintf("%s ROM BIOS 1989-1996 • ZERO-DOCKER VIRTUAL WORKSPACE", tagStyle.Render(" SISTEMA ")))
	diagLines = append(diagLines, fmt.Sprintf("%s 640KB BASE OK • HYBRID KV-CACHE EN RAM: ACTIVO", tagStyle.Render(" MEMORIA ")))
	diagLines = append(diagLines, fmt.Sprintf("%s MAPA DE RUTAS Y PROYECTOS DEL HOST INDEXADO EN RAM", tagStyle.Render(" DISCO   ")))
	diagLines = append(diagLines, fmt.Sprintf("%s OZY (EXEC) • CHARC (SEGURIDAD) • NINE (ESTRATEGA) • DREAMER", tagStyle.Render(" TRIADA  ")))

	diagBox := diagBorder.Render(strings.Join(diagLines, "\n"))
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(diagBox))
	sb.WriteString("\n\n")

	// 3. Tarjeta de Acciones Rápidas & Prompts Sugeridos (Estilo Grok Central)
	cardBorder := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	var cardSb strings.Builder
	cardSb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("── [ACCIONES RAPIDAS & PROMPTS DISPONIBLES (TIPO GROK)] ──\n\n"))

	suggestions := []struct {
		cmd  string
		desc string
	}{
		{"/paths", "Inspeccionar mapa de proyectos del host indexados en RAM"},
		{"/profile", "Consultar o refinar tu perfil permanente y directivas"},
		{"/dream", "Ejecutar consolidacion cognitiva de memoria con DREAMER"},
		{"\"Abre con antigravity ide el proyecto crmgeofal\"", "Lanza tu editor en el repo"},
		{"\"Crea un Excel de presupuesto en el escritorio\"", "Genera hoja .xlsx con formato"},
		{"\"Monitorea el puerto 8080 con watchdog\"", "Vigila tus servicios en background"},
	}

	for i, s := range suggestions {
		numTag := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(fmt.Sprintf("[%d] %s", i+1, s.cmd))
		cardSb.WriteString(fmt.Sprintf("  %s\n      %s\n", numTag, MutedStyle.Render(s.desc)))
	}

	renderedCards := cardBorder.Render(cardSb.String())
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(renderedCards))
	sb.WriteString("\n\n")

	// 4. Prompt hint inferior
	hint := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render(">> Escribe tu orden abajo para comenzar la sesion de chat, o usa /help")
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(hint))
	sb.WriteString("\n")

	return sb.String()
}
