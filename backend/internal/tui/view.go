package tui

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
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

	if m.state == StateStartMenu {
		return m.renderStartMenuView()
	}
	if m.state == StateSettingsMenu {
		return m.renderSettingsMenuView()
	}
	if m.state == StateProviderMenu {
		return m.renderProviderSelectView()
	}
	if m.state == StateAPIKeySelect {
		return m.renderAPIKeySelectView()
	}
	if m.state == StateAPIKeyInput {
		return m.renderAPIKeyInputView()
	}
	if m.state == StateProfileEdit {
		return m.renderProfileEditView()
	}
	if m.state == StateChatHistory {
		return m.renderChatHistoryView()
	}
	if m.state == StateMemoryManager {
		return m.renderMemoryManagerView()
	}

	var sb strings.Builder

	// 1. Header superior
	sb.WriteString(m.renderHeader())
	sb.WriteString("\n")

	// 2. Historial de conversación (Viewport) o Vista Dividida con Sidebar (Zona Amarilla)
	if m.showSidebar && m.width >= 70 {
		sidebarW := 34
		if m.width < 95 {
			sidebarW = 26
		}
		chatW := m.width - sidebarW - 1
		if chatW < 35 {
			chatW = 35
		}

		vpView := m.viewport.View()
		sidebarView := m.renderSidebar(sidebarW, m.viewport.Height)
		sep := lipgloss.NewStyle().Foreground(ColorDim).Render("│")

		content := lipgloss.JoinHorizontal(lipgloss.Top, vpView, sep, sidebarView)
		sb.WriteString(content)
		sb.WriteString("\n")
	} else {
		sb.WriteString(m.viewport.View())
		sb.WriteString("\n")
	}

	// 3. Footer inferior (Status + Input box + Info modelo + Hints)
	sb.WriteString(m.renderFooter())

	result := sb.String()
	if m.height > 0 {
		lines := strings.Split(result, "\n")
		if len(lines) > m.height {
			result = strings.Join(lines[:m.height], "\n")
		}
	}
	return result
}

func (m Model) renderHeader() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}

	sessionHash := "d850815e"
	if m.chat != nil && len(m.chat.ID) >= 8 {
		sessionHash = m.chat.ID[:8]
	} else if m.loopSessionID != "" && len(m.loopSessionID) >= 8 {
		sessionHash = m.loopSessionID[:8]
	}

	codeTag := lipgloss.NewStyle().Foreground(ColorMuted).Bold(true).Render("#" + sessionHash)

	// Verificar si hay un tema real hablado en la conversación
	hasRealTopic := m.chat != nil && m.chat.Name != "" &&
		m.chat.Name != "TUI Session" &&
		m.chat.Name != "Nueva Sesion" &&
		!strings.HasPrefix(m.chat.Name, "Sesion ")

	var leftPart string
	if hasRealTopic {
		topicText := m.chat.Name
		if len(topicText) > 45 {
			topicText = topicText[:42] + "..."
		}
		topicLabel := lipgloss.NewStyle().Foreground(ColorText).Bold(false).Render(topicText)
		sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("·")
		leftPart = fmt.Sprintf(" %s %s %s", codeTag, sep, topicLabel)
	} else {
		leftPart = fmt.Sprintf(" %s", codeTag)
	}

	// Metadatos de la derecha: proveedor/modelo sutil
	rightInfo := ""
	if m.provider != nil {
		rightInfo = m.provider.Name()
		if m.chat != nil && m.chat.Model != "" {
			rightInfo += ":" + m.chat.Model
		}
	}
	rightPart := lipgloss.NewStyle().Foreground(ColorDim).Render(rightInfo + " ")

	spaceCount := convWidth - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
	if spaceCount < 1 {
		spaceCount = 1
	}
	topLine := fmt.Sprintf("%s%s%s", leftPart, strings.Repeat(" ", spaceCount), rightPart)

	// Línea divisoria gris/oscura continua
	divider := lipgloss.NewStyle().Foreground(lipgloss.Color("#2a2a2a")).Render(strings.Repeat("─", convWidth))

	return topLine + "\n" + divider
}

func (m Model) renderInteractiveCard(card *InteractiveCard, convWidth int) string {
	if card == nil {
		return ""
	}
	innerWidth := convWidth - 6
	if innerWidth < 30 {
		innerWidth = 30
	}

	var cardSb strings.Builder

	// 1. Tabs superiores (ej: Prioridad  Acciones  Configurar)
	if len(card.Tabs) > 0 {
		var tabParts []string
		for idx, tab := range card.Tabs {
			if idx == card.ActiveTab {
				tabParts = append(tabParts, CardTabActiveStyle.Render(tab))
			} else {
				tabParts = append(tabParts, CardTabInactiveStyle.Render(tab))
			}
		}
		cardSb.WriteString(strings.Join(tabParts, "   "))
		cardSb.WriteString("\n\n")
	}

	// 2. Pregunta o Título del paso
	if card.Question != "" {
		cardSb.WriteString(CardQuestionStyle.Render(card.Question))
		cardSb.WriteString("\n\n")
	}

	// 3. Opciones seleccionables con fila destacada
	for idx, opt := range card.Options {
		isSelected := idx == card.SelectedIndex && !card.IsAnswered && m.activeCard == card
		if isSelected {
			cursor := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(">> ")
			optText := CardOptionSelectedStyle.Render(fmt.Sprintf(" %s ", opt))
			cardSb.WriteString(fmt.Sprintf("%s%s\n", cursor, optText))
		} else if card.IsAnswered && opt == card.Answer {
			check := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render(" ✓ ")
			optText := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render(opt)
			cardSb.WriteString(fmt.Sprintf("%s%s\n", check, optText))
		} else {
			cursor := "   "
			cardSb.WriteString(fmt.Sprintf("%s%s\n", cursor, CardOptionNormalStyle.Render(opt)))
		}
	}
	cardSb.WriteString("\n")

	// 4. Barra inferior de atajos de la tarjeta
	if !card.IsAnswered && m.activeCard == card {
		hints := CardHintsStyle.Render(" [Tab] Fase  •  [↑ / ↓] Seleccionar  •  [Enter] Confirmar  •  [Esc] Omitir ")
		cardSb.WriteString(hints)
		cardSb.WriteString("\n")
	}

	return CardContainerStyle.Width(innerWidth).Render(cardSb.String())
}

func (m Model) renderConversation() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}
	if m.showSidebar && m.width >= 70 {
		sidebarW := 34
		if m.width < 95 {
			sidebarW = 26
		}
		convWidth = m.width - sidebarW - 1
		if convWidth < 35 {
			convWidth = 35
		}
	}

	var sb strings.Builder

	boxStyle := lipgloss.NewStyle().
		Background(ColorMessageBg).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeftForeground(ColorPrimary).
		Padding(0, 1)

	for _, entry := range m.entries {
		switch entry.Role {
		case "user":
			maxUserW := convWidth - 12
			if maxUserW < 20 {
				maxUserW = 20
			}
			lines := wrapContent(entry.Content, maxUserW)
			rawUserText := strings.Join(lines, "\n")
			cardText := fmt.Sprintf("%s\n%s", UserStyle.Render("❯ TÚ:"), rawUserText)
			sb.WriteString(UserBoxStyle.Render(cardText))
			sb.WriteString("\n\n")

		case "assistant":
			// Renderizar pensamiento (Thinking / Chain of Thought) si existe y está desplegado
			if entry.Thinking != "" && m.showThinking {
				sb.WriteString(renderThinkingBlock(entry.Thinking, false, "", convWidth))
			}

			cleaned := cleanAssistantText(entry.Content)
			if cleaned != "" {
				maxAssistantW := convWidth - 12
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
					rendered = strings.TrimRight(rendered, "\r\n")
					cardText := fmt.Sprintf("%s\n%s", AssistantStyle.Render("O Ozy:"), rendered)
					sb.WriteString(boxStyle.Render(cardText))
					sb.WriteString("\n\n")
				} else {
					lines := wrapContent(cleaned, maxAssistantW)
					rawText := strings.Join(lines, "\n")
					cardText := fmt.Sprintf("%s\n%s", AssistantStyle.Render("O Ozy:"), rawText)
					sb.WriteString(boxStyle.Render(cardText))
					sb.WriteString("\n\n")
				}
			}

			// Renderizar tarjeta interactiva (Plan / Decisión con Tabs y Opciones) si existe
			if entry.Card != nil {
				renderedCard := m.renderInteractiveCard(entry.Card, convWidth)
				if renderedCard != "" {
					sb.WriteString(renderedCard)
					sb.WriteString("\n\n")
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
			// Solo renderizar el playground si el hilo de ejecución está desplegado
			if m.showThinking {
				sb.WriteString(renderPlaygroundTool(entry.ToolName, entry.ToolInput, entry.Content, entry.ToolSuccess, entry.DurationMs, convWidth))
			}
		}
	}

	// Si hay una herramienta ejecutándose en este momento (solo visible si el hilo está desplegado)
	if m.activeToolName != "" && m.showThinking {
		sb.WriteString(renderActiveToolProgress(m.activeToolName, m.activeToolInput, m.spinner.View(), convWidth))
	}

	// Si hay pensamiento en tiempo real mientras el modelo razona
	if m.currentThinking != "" && m.showThinking {
		sb.WriteString(renderThinkingBlock(m.currentThinking, true, m.spinner.View(), convWidth))
	}

	// Si hay streaming activo en este momento
	if len(m.currentStream) > 0 {
		cleaned := cleanAssistantText(m.currentStream)
		if cleaned != "" {
			maxStreamW := convWidth - 12
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
				rendered = strings.TrimRight(rendered, "\r\n") + "▌"
				cardText := fmt.Sprintf("%s\n%s", AssistantStyle.Render("O Ozy:"), rendered)
				sb.WriteString(boxStyle.Render(cardText))
				sb.WriteString("\n\n")
			} else {
				lines := wrapContent(cleaned, maxStreamW)
				rawText := strings.Join(lines, "\n") + "▌"
				cardText := fmt.Sprintf("%s\n%s", AssistantStyle.Render("O Ozy:"), rawText)
				sb.WriteString(boxStyle.Render(cardText))
				sb.WriteString("\n\n")
			}
		}
	}

	return sb.String()
}

// parseToolExecutionDisplay extrae si la llamada corresponde a un comando de terminal y genera un título descriptivo.
func parseToolExecutionDisplay(toolName, toolInput string) (isCommand bool, title string) {
	lowerName := strings.ToLower(toolName)
	if lowerName == "run_command" || lowerName == "os_run_command" || lowerName == "shell_exec" {
		isCommand = true
		var params struct {
			Command string `json:"command"`
			Cwd     string `json:"cwd"`
		}
		if err := json.Unmarshal([]byte(toolInput), &params); err == nil && params.Command != "" {
			cmdStr := strings.TrimSpace(params.Command)
			if params.Cwd != "" && params.Cwd != "." {
				title = fmt.Sprintf("❯ %s (en %s)", cmdStr, params.Cwd)
			} else {
				title = fmt.Sprintf("❯ %s", cmdStr)
			}
			return
		}
		cleanInput := strings.TrimSpace(toolInput)
		if cleanInput != "" {
			title = fmt.Sprintf("❯ %s", cleanInput)
			return
		}
		title = "❯ comando del sistema"
		return
	}

	isCommand = false
	cleanName := toolName
	if cleanName == "" {
		cleanName = "herramienta"
	}
	if strings.TrimSpace(toolInput) == "" {
		title = cleanName
		return
	}

	var genericMap map[string]interface{}
	if err := json.Unmarshal([]byte(toolInput), &genericMap); err == nil && len(genericMap) > 0 {
		if pathVal, ok := genericMap["path"].(string); ok && pathVal != "" {
			title = fmt.Sprintf("%s (path: %s)", cleanName, pathVal)
			return
		}
		if queryVal, ok := genericMap["query"].(string); ok && queryVal != "" {
			title = fmt.Sprintf("%s (query: %q)", cleanName, queryVal)
			return
		}
		if patternVal, ok := genericMap["pattern"].(string); ok && patternVal != "" {
			title = fmt.Sprintf("%s (patrón: %s)", cleanName, patternVal)
			return
		}
		if targetVal, ok := genericMap["target"].(string); ok && targetVal != "" {
			title = fmt.Sprintf("%s (%s)", cleanName, targetVal)
			return
		}
		if nameVal, ok := genericMap["name"].(string); ok && nameVal != "" {
			title = fmt.Sprintf("%s (%s)", cleanName, nameVal)
			return
		}
		var parts []string
		for k, v := range genericMap {
			if v == nil {
				continue
			}
			strVal := fmt.Sprintf("%v", v)
			if strVal == "<nil>" || strVal == "null" || strVal == "[]" || strVal == "" {
				continue
			}
			if len(strVal) > 30 {
				strVal = strVal[:27] + "..."
			}
			parts = append(parts, fmt.Sprintf("%s: %s", k, strVal))
			if len(parts) >= 2 {
				break
			}
		}
		if len(parts) > 0 {
			title = fmt.Sprintf("%s (%s)", cleanName, strings.Join(parts, ", "))
		} else {
			title = cleanName
		}
		return
	}

	rawInput := strings.TrimSpace(toolInput)
	if len(rawInput) > 40 {
		rawInput = rawInput[:37] + "..."
	}
	title = fmt.Sprintf("%s (%s)", cleanName, rawInput)
	return
}

// renderPlaygroundTool formatea la ejecución de un comando o herramienta en un bloque interactivo estilo playground.
func renderPlaygroundTool(toolName, toolInput, content string, success bool, durationMs int64, convWidth int) string {
	isCommand, title := parseToolExecutionDisplay(toolName, toolInput)

	var sb strings.Builder
	borderStyle := lipgloss.NewStyle().Foreground(ColorDim)

	if isCommand {
		sb.WriteString(borderStyle.Render("┌─ ") + PlaygroundHeaderCmdStyle.Render("[COMANDO] ") + title + "\n")
	} else {
		sb.WriteString(borderStyle.Render("┌─ ") + PlaygroundHeaderToolStyle.Render("[PLAYGROUND] ") + title + "\n")
	}

	maxOutWidth := convWidth - 6
	if maxOutWidth < 25 {
		maxOutWidth = 25
	}

	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		sb.WriteString(borderStyle.Render("│ ") + MutedStyle.Render("(sin salida de consola)") + "\n")
	} else {
		lines := strings.Split(trimmedContent, "\n")
		const maxLinesToShow = 25
		if len(lines) > maxLinesToShow {
			head := lines[:15]
			tail := lines[len(lines)-5:]
			omitted := len(lines) - 20

			for _, l := range head {
				for _, wl := range wrapLine(l, maxOutWidth) {
					sb.WriteString(borderStyle.Render("│ ") + ToolContentStyle.Render(wl) + "\n")
				}
			}
			sb.WriteString(borderStyle.Render("│ ") + MutedStyle.Render(fmt.Sprintf("... (%d líneas omitidas) ...", omitted)) + "\n")
			for _, l := range tail {
				for _, wl := range wrapLine(l, maxOutWidth) {
					sb.WriteString(borderStyle.Render("│ ") + ToolContentStyle.Render(wl) + "\n")
				}
			}
		} else {
			for _, l := range lines {
				for _, wl := range wrapLine(l, maxOutWidth) {
					sb.WriteString(borderStyle.Render("│ ") + ToolContentStyle.Render(wl) + "\n")
				}
			}
		}
	}

	durStr := ""
	if durationMs > 0 {
		durStr = fmt.Sprintf(" (%dms)", durationMs)
	}

	if success {
		sb.WriteString(borderStyle.Render("└─ ") + PlaygroundSuccessStyle.Render("✓ Completado") + MutedStyle.Render(durStr) + "\n\n")
	} else {
		sb.WriteString(borderStyle.Render("└─ ") + PlaygroundErrorStyle.Render("✗ Error") + MutedStyle.Render(durStr) + "\n\n")
	}

	return sb.String()
}

// renderActiveToolProgress renderiza la herramienta o comando actualmente en progreso.
func renderActiveToolProgress(toolName, toolInput string, spinnerView string, convWidth int) string {
	isCommand, title := parseToolExecutionDisplay(toolName, toolInput)
	borderStyle := lipgloss.NewStyle().Foreground(ColorDim)

	var sb strings.Builder
	if isCommand {
		sb.WriteString(borderStyle.Render("┌─ ") + PlaygroundHeaderCmdStyle.Render("[COMANDO] ") + title + " " + spinnerView + "\n")
	} else {
		sb.WriteString(borderStyle.Render("┌─ ") + PlaygroundHeaderToolStyle.Render("[EJECUTANDO] ") + title + " " + spinnerView + "\n")
	}
	sb.WriteString(borderStyle.Render("└─ ") + MutedStyle.Render("En ejecución en segundo plano...") + "\n\n")
	return sb.String()
}

// renderCollapsedTool formatea una herramienta en una sola línea sutil cuando el hilo de ejecución está plegado.
func renderCollapsedTool(toolName, toolInput string, success bool, durationMs int64) string {
	isCommand, title := parseToolExecutionDisplay(toolName, toolInput)
	durStr := ""
	if durationMs > 0 {
		durStr = fmt.Sprintf(" (%dms)", durationMs)
	}
	status := PlaygroundSuccessStyle.Render("✓")
	if !success {
		status = PlaygroundErrorStyle.Render("✗")
	}

	prefix := "[PLAYGROUND]"
	if isCommand {
		prefix = "[COMANDO]"
	}

	return fmt.Sprintf("  %s %s %s %s%s\n", MutedStyle.Render("·"), PlaygroundHeaderCmdStyle.Render(prefix), MutedStyle.Render(title), status, MutedStyle.Render(durStr))
}

// renderCollapsedActiveTool renderiza la herramienta activa en una sola línea discreta.
func renderCollapsedActiveTool(toolName string, spinnerView string) string {
	cleanName := toolName
	if cleanName == "" {
		cleanName = "herramienta"
	}
	return fmt.Sprintf("  %s %s %s %s\n", MutedStyle.Render("·"), PlaygroundHeaderCmdStyle.Render("[EJECUTANDO]"), MutedStyle.Render(cleanName), spinnerView)
}

// renderThinkingBlock renderiza el bloque de razonamiento (pensamiento / chain of thought).
func renderThinkingBlock(thinking string, isStreaming bool, spinnerView string, convWidth int) string {
	trimmed := strings.TrimSpace(thinking)
	if trimmed == "" {
		return ""
	}

	borderStyle := lipgloss.NewStyle().Foreground(ColorDim)
	var sb strings.Builder
	if isStreaming {
		sb.WriteString(borderStyle.Render("┌─ ") + MutedStyle.Render("Razonamiento ") + spinnerView + "\n")
	} else {
		sb.WriteString(borderStyle.Render("┌─ ") + MutedStyle.Render("Razonamiento") + "\n")
	}

	maxThinkW := convWidth - 6
	if maxThinkW < 20 {
		maxThinkW = 20
	}

	thinkLines := strings.Split(trimmed, "\n")
	for _, tl := range thinkLines {
		wrapped := wrapLine(tl, maxThinkW)
		for _, wl := range wrapped {
			sb.WriteString(borderStyle.Render("│ ") + MutedStyle.Render(wl) + "\n")
		}
	}
	sb.WriteString(borderStyle.Render("└─") + "\n\n")
	return sb.String()
}

func (m Model) renderFooter() string {
	var sb strings.Builder

	// Barra de estado: solo se muestra cuando el agente está procesando o hay cola en espera
	if m.state != StateIdle || len(m.messageQueue) > 0 {
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
	}

	// Caja de texto con badge de marca OZY
	borderStyle := InputBorderStyle
	if m.state != StateIdle {
		borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorQueue)
	}
	badge := InputBadgeStyle.Render(" OZY ")
	inputContent := fmt.Sprintf("%s %s", badge, m.textarea.View())
	sb.WriteString(borderStyle.Render(inputContent))
	sb.WriteString("\n")

	// Hints de atajos contextuales con indicador de voz discreto
	var hintsText string
	if m.state != StateIdle {
		hintsText = " [Enter] Encolar  •  [/now <orden>] Enviar directo  •  [/cancel | Esc] Cancelar  •  [/queue] Ver cola"
		if m.width > 0 && m.width < 95 {
			hintsText = " [Enter] Encolar  •  [/now <orden>] Directo  •  [Esc] Cancelar"
		}
	} else if len(m.messageQueue) > 0 {
		hintsText = fmt.Sprintf(" [Enter] Enviar  •  [COLA: %d] (/queue)  •  [/clearqueue] Vaciar  •  [Ctrl+C] Salir", len(m.messageQueue))
	} else if m.activeCard != nil {
		hintsText = " [↑ / ↓] Elegir opción  •  [Enter] Confirmar  •  [Esc] Escribir texto libre  •  [/help] Comandos"
	} else {
		hintsText = " [Enter] Enviar  •  [Ctrl+V] Voz  •  [Ctrl+B] Contexto  •  [/menu | Esc] Menú  •  [/help] Ayuda"
		if m.width > 0 && m.width < 95 {
			hintsText = " [Enter] Enviar  •  [Ctrl+V] Voz  •  [Esc] Menú  •  [/help] Ayuda"
		}
	}

	// Indicador discreto de voz si está activa
	if m.voiceEnabled {
		hintsText += "  •  voz: on ('Hey Ozy')"
	}

	hints := MutedStyle.Render(hintsText)
	if m.width > 0 {
		sb.WriteString(lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(hints))
	} else {
		sb.WriteString(hints)
	}

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
		Render("OZYASIST >> OS CONTROL & COWORK AGENT // ZERO-DOCKER KERNEL v7.0")

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

	// 3. Tarjeta de Acciones Rápidas & Prompts Sugeridos
	cardBorder := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	var cardSb strings.Builder
	cardHeader := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("── [ACCIONES RAPIDAS & PROMPTS SUGERIDOS] ──")
	cardSb.WriteString(cardHeader + "\n\n")

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

func (m Model) renderStartMenuView() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}
	innerWidth := convWidth - 6
	if innerWidth > 82 {
		innerWidth = 82
	}
	if innerWidth < 40 {
		innerWidth = 40
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
		Render("OZYASIST >> OS CONTROL & COWORK AGENT // ZERO-DOCKER KERNEL v7.0")

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

	// 3. Menú Interactivo de Inicio con flechitas
	menuBorder := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	type startMenuItem struct {
		num   string
		title string
		desc  string
	}

	items := []startMenuItem{
		{"1", "INICIAR CONVERSACION", "Abre la terminal de chat interactivo y control agéntico del sistema"},
		{"2", "HISTORIAL DE CONVERSACIONES", "Retomar o eliminar sesiones previas guardadas en SQLite"},
		{"3", "CONFIGURACIONES", "Ajusta proveedor LLM, permisos de seguridad, voz y diagnósticos"},
		{"4", "MODO VOZ EN VIVO", "Activa la consola interactiva con escucha ('Hey Ozy') y respuestas por voz"},
		{"5", "SALIR", "Cierra la sesión y apaga el asistente de consola"},
	}

	var menuSb strings.Builder
	menuHeader := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("── [MENU PRINCIPAL DE CONTROL] ──")
	menuSb.WriteString(menuHeader + "\n\n")

	for i, it := range items {
		selected := i == m.menuIndex
		var rowTitle string
		var rowDesc string
		if selected {
			cursor := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(">> ")
			numPart := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(fmt.Sprintf("[%s]", it.num))
			titlePart := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(" " + it.title + " ")
			rowTitle = fmt.Sprintf("%s%s%s", cursor, numPart, titlePart)
			rowDesc = lipgloss.NewStyle().Foreground(ColorText).Render(fmt.Sprintf("     %s", it.desc))
		} else {
			cursor := "   "
			numPart := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(fmt.Sprintf("[%s]", it.num))
			titlePart := lipgloss.NewStyle().Foreground(ColorText).Render(" " + it.title)
			rowTitle = fmt.Sprintf("%s%s%s", cursor, numPart, titlePart)
			rowDesc = lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("     %s", it.desc))
		}
		menuSb.WriteString(rowTitle + "\n" + rowDesc + "\n\n")
	}

	renderedMenu := menuBorder.Render(menuSb.String())
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(renderedMenu))
	sb.WriteString("\n")

	// 4. Atajos inferiores
	hintsText := " [↑ / ↓] Navegar  •  [Enter] Seleccionar  •  [1-4] Acceso directo  •  [q] Salir "
	renderedHints := lipgloss.NewStyle().Foreground(ColorMuted).Render(hintsText)
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(renderedHints))
	sb.WriteString("\n")

	return sb.String()
}

func (m Model) renderSettingsMenuView() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}
	innerWidth := convWidth - 6
	if innerWidth > 82 {
		innerWidth = 82
	}
	if innerWidth < 40 {
		innerWidth = 40
	}

	var sb strings.Builder
	sb.WriteString("\n")

	// Título superior
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(
		titleStyle.Render("OZYASIST >> PANEL DE CONFIGURACIONES // ZERO-DOCKER KERNEL v7.0"),
	))
	sb.WriteString("\n\n")

	// Determinar valores actuales
	provName := "sin proveedor"
	if m.provider != nil {
		provName = m.provider.Name()
	} else if m.chat != nil && m.chat.Provider != "" {
		provName = m.chat.Provider
	}
	modelName := "defecto"
	if m.chat != nil && m.chat.Model != "" {
		modelName = m.chat.Model
	}

	voiceStatus := "DESACTIVADA (OFF)"
	if m.voiceEnabled {
		voiceStatus = "ACTIVADA (ON - Wake Word 'Hey Ozy')"
	}

	pathCount := 0
	if reg := system.DefaultPathRegistry(); reg != nil {
		pathCount = len(reg.ListAll())
	}

	userProfileSummary := "Sin definir (usa /profile para personalizar)"
	if db.DB != nil {
		if u, err := db.GetUser(db.DefaultUserID()); err == nil && u != nil && strings.TrimSpace(u.ProfileMd) != "" {
			lines := strings.Split(strings.TrimSpace(u.ProfileMd), "\n")
			userProfileSummary = lines[0]
			if len(userProfileSummary) > 38 {
				userProfileSummary = userProfileSummary[:35] + "..."
			}
		}
	}

	type settingRow struct {
		num   string
		label string
		val   string
		desc  string
	}

	items := []settingRow{
		{"1", "Proveedor LLM Activo", fmt.Sprintf("%s (%s)", provName, modelName), "Enter: Abrir selector interactivo de proveedores (flechas ↑/↓)"},
		{"2", "Gestión de Claves API", "Configurar / Actualizar tokens", "Enter: Seleccionar proveedor e ingresar clave interactiva"},
		{"3", "Perfil de Usuario", userProfileSummary, "Enter: Consultar o editar directivas y preferencias en SQLite"},
		{"4", "Nivel de Permisos SO", m.permissionLevel, "Enter: Alternar nivel (autonomous -> supervised -> sandboxed)"},
		{"5", "Escucha de Voz Continua", voiceStatus, "Enter: Alternar escucha continua de voz ('Hey Ozy')"},
		{"6", "Mapa de Rutas en RAM", fmt.Sprintf("%d ubicaciones indexadas", pathCount), "Enter: Inspeccionar detalles de rutas indexadas en RAM"},
		{"7", "Volver al Menú Principal", "[ <- REGRESAR ]", "Enter o Esc: Vuelve al menú principal de inicio"},
	}

	settingsBorder := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	var ssb strings.Builder
	ssbHeader := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("── [CONFIGURACIONES DE SISTEMA] ──")
	ssb.WriteString(ssbHeader + "\n\n")

	for i, it := range items {
		selected := i == m.settingsIndex
		var line string
		if selected {
			cursor := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(">> ")
			tag := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(fmt.Sprintf("[%s] %s:", it.num, it.label))
			val := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(" " + it.val)
			desc := lipgloss.NewStyle().Foreground(ColorText).Render(fmt.Sprintf("     %s", it.desc))
			line = fmt.Sprintf("%s%s%s\n%s\n", cursor, tag, val, desc)
		} else {
			cursor := "   "
			tag := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(fmt.Sprintf("[%s] %s:", it.num, it.label))
			val := lipgloss.NewStyle().Foreground(ColorText).Render(" " + it.val)
			desc := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("     %s", it.desc))
			line = fmt.Sprintf("%s%s%s\n%s\n", cursor, tag, val, desc)
		}
		ssb.WriteString(line)
	}

	renderedSettings := settingsBorder.Render(ssb.String())
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(renderedSettings))
	sb.WriteString("\n")

	// Cuadro de feedback si hay settingsNotice
	if m.settingsNotice != "" {
		noticeBox := lipgloss.NewStyle().
			Width(innerWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1).
			Render(m.settingsNotice)
		sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(noticeBox))
		sb.WriteString("\n")
	}

	// Hints
	hints := lipgloss.NewStyle().Foreground(ColorMuted).Render(" [↑ / ↓] Navegar  •  [Enter] Seleccionar / Abrir  •  [Esc / 7] Volver ")
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(hints))
	sb.WriteString("\n")

	return sb.String()
}

func (m Model) renderProviderSelectView() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}
	innerWidth := convWidth - 6
	if innerWidth > 86 {
		innerWidth = 86
	}
	if innerWidth < 40 {
		innerWidth = 40
	}

	var sb strings.Builder
	sb.WriteString("\n")

	// Título superior
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(
		titleStyle.Render("OZYASIST >> SELECTOR INTERACTIVO DE PROVEEDORES LLM // ZERO-DOCKER"),
	))
	sb.WriteString("\n\n")

	// Obtener catálogo oficial
	catalog := providers.GetSupportedCatalog()

	// Proveedor actual activo en la sesión
	currentProv := ""
	if m.provider != nil {
		currentProv = strings.ToLower(m.provider.Name())
	} else if m.chat != nil && m.chat.Provider != "" {
		currentProv = strings.ToLower(m.chat.Provider)
	}

	var provSb strings.Builder
	header := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("── [PROVEEDORES DISPONIBLES EN EL SISTEMA] ──")
	provSb.WriteString(header + "\n\n")

	// Items: proveedores + opción final de volver
	totalItems := len(catalog) + 1

	for i := 0; i < totalItems; i++ {
		selected := i == m.providerIndex
		if i < len(catalog) {
			cat := catalog[i]
			isCurrentActive := strings.ToLower(cat.ID) == currentProv
			hasKey := providers.GetProviderKey(cat.ID) != ""

			// Tag de estado activo
			activeTag := ""
			if isCurrentActive {
				activeTag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(" [*] ACTIVO ")
			}

			// Tag de clave
			keyTag := ""
			if cat.IsLocal {
				keyTag = lipgloss.NewStyle().Foreground(ColorPrimary).Render("[LOCAL]")
			} else if hasKey {
				keyTag = lipgloss.NewStyle().Foreground(ColorPrimary).Render("[OK] CLAVE GUARDADA")
			} else {
				keyTag = lipgloss.NewStyle().Foreground(ColorMuted).Render("[--] SIN CLAVE")
			}

			var line string
			if selected {
				cursor := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(">> ")
				numPart := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(fmt.Sprintf("[%d]", i+1))
				namePart := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(fmt.Sprintf(" %s (%s) ", cat.DisplayName, cat.DefaultModel))
				titleLine := fmt.Sprintf("%s%s%s", cursor, numPart, namePart)
				if activeTag != "" {
					titleLine += " " + activeTag
				}
				if keyTag != "" {
					titleLine += " " + keyTag
				}
				descLine := lipgloss.NewStyle().Foreground(ColorText).Render(fmt.Sprintf("     %s", cat.Description))
				line = titleLine + "\n" + descLine + "\n\n"
			} else {
				cursor := "   "
				numPart := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(fmt.Sprintf("[%d]", i+1))
				namePart := lipgloss.NewStyle().Bold(true).Foreground(ColorText).Render(fmt.Sprintf(" %s (%s)", cat.DisplayName, cat.DefaultModel))
				titleLine := fmt.Sprintf("%s%s%s", cursor, numPart, namePart)
				if activeTag != "" {
					titleLine += " " + activeTag
				}
				if keyTag != "" {
					titleLine += " " + keyTag
				}
				descLine := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("     %s", cat.Description))
				line = titleLine + "\n" + descLine + "\n\n"
			}
			provSb.WriteString(line)
		} else {
			// Opción de retorno a configuraciones
			var line string
			if selected {
				cursor := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(">> ")
				tag := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(fmt.Sprintf("[%d] Volver a Configuraciones: [ <- REGRESAR ]", i+1))
				desc := lipgloss.NewStyle().Foreground(ColorText).Render("     Enter o Esc: Regresa al panel de configuraciones general")
				line = fmt.Sprintf("%s%s\n%s\n", cursor, tag, desc)
			} else {
				cursor := "   "
				tag := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(fmt.Sprintf("[%d] Volver a Configuraciones: [ <- REGRESAR ]", i+1))
				desc := lipgloss.NewStyle().Foreground(ColorMuted).Render("     Enter o Esc: Regresa al panel de configuraciones general")
				line = fmt.Sprintf("%s%s\n%s\n", cursor, tag, desc)
			}
			provSb.WriteString(line)
		}
	}

	provBorder := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	renderedProv := provBorder.Render(provSb.String())
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(renderedProv))
	sb.WriteString("\n")

	// Cuadro de feedback si hay settingsNotice
	if m.settingsNotice != "" {
		noticeBox := lipgloss.NewStyle().
			Width(innerWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1).
			Render(m.settingsNotice)
		sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(noticeBox))
		sb.WriteString("\n")
	}

	// Barra inferior de hints
	hints := lipgloss.NewStyle().Foreground(ColorMuted).Render(" [↑ / ↓] Navegar  •  [Enter] Activar Proveedor  •  [e] Modificar Clave  •  [Esc / 9] Volver ")
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(hints))
	sb.WriteString("\n")

	return sb.String()
}

func (m Model) renderAPIKeySelectView() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}
	innerWidth := convWidth - 6
	if innerWidth > 86 {
		innerWidth = 86
	}
	if innerWidth < 40 {
		innerWidth = 40
	}

	var sb strings.Builder
	sb.WriteString("\n")

	// Título superior
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(
		titleStyle.Render("OZYASIST >> GESTION INTERACTIVA DE CLAVES API // ZERO-DOCKER"),
	))
	sb.WriteString("\n\n")

	catalog := providers.GetSupportedCatalog()

	var ksb strings.Builder
	header := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("── [CONFIGURAR API KEYS DE PROVEEDORES] ──")
	ksb.WriteString(header + "\n\n")

	totalItems := len(catalog) + 1

	for i := 0; i < totalItems; i++ {
		selected := i == m.apiKeyIndex
		if i < len(catalog) {
			cat := catalog[i]
			key := providers.GetProviderKey(cat.ID)

			statusTag := ""
			if cat.IsLocal {
				statusTag = lipgloss.NewStyle().Foreground(ColorPrimary).Render("[LOCAL] No requiere clave")
			} else if key != "" {
				masked := key
				if len(masked) > 8 {
					masked = masked[:4] + "..." + masked[len(masked)-4:]
				}
				statusTag = lipgloss.NewStyle().Foreground(ColorPrimary).Render(fmt.Sprintf("[OK] Guardada: %s", masked))
			} else {
				statusTag = lipgloss.NewStyle().Foreground(ColorMuted).Render("[--] Sin clave configurada")
			}

			var line string
			if selected {
				cursor := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(">> ")
				numPart := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(fmt.Sprintf("[%d]", i+1))
				namePart := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(fmt.Sprintf(" %s ", cat.DisplayName))
				descLine := lipgloss.NewStyle().Foreground(ColorText).Render(fmt.Sprintf("     Enter: Ingresar o actualizar clave para %s", cat.DisplayName))
				line = fmt.Sprintf("%s%s%s %s\n%s\n\n", cursor, numPart, namePart, statusTag, descLine)
			} else {
				cursor := "   "
				numPart := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(fmt.Sprintf("[%d]", i+1))
				namePart := lipgloss.NewStyle().Bold(true).Foreground(ColorText).Render(fmt.Sprintf(" %s", cat.DisplayName))
				descLine := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("     Enter: Ingresar o actualizar clave para %s", cat.DisplayName))
				line = fmt.Sprintf("%s%s%s %s\n%s\n\n", cursor, numPart, namePart, statusTag, descLine)
			}
			ksb.WriteString(line)
		} else {
			// Opción volver a configuraciones
			var line string
			if selected {
				cursor := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(">> ")
				tag := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(fmt.Sprintf("[%d] Volver a Configuraciones: [ <- REGRESAR ]", i+1))
				desc := lipgloss.NewStyle().Foreground(ColorText).Render("     Enter o Esc: Regresa al menú general de configuraciones")
				line = fmt.Sprintf("%s%s\n%s\n", cursor, tag, desc)
			} else {
				cursor := "   "
				tag := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(fmt.Sprintf("[%d] Volver a Configuraciones: [ <- REGRESAR ]", i+1))
				desc := lipgloss.NewStyle().Foreground(ColorMuted).Render("     Enter o Esc: Regresa al menú general de configuraciones")
				line = fmt.Sprintf("%s%s\n%s\n", cursor, tag, desc)
			}
			ksb.WriteString(line)
		}
	}

	keyBorder := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	renderedKeys := keyBorder.Render(ksb.String())
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(renderedKeys))
	sb.WriteString("\n")

	if m.settingsNotice != "" {
		noticeBox := lipgloss.NewStyle().
			Width(innerWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1).
			Render(m.settingsNotice)
		sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(noticeBox))
		sb.WriteString("\n")
	}

	hints := lipgloss.NewStyle().Foreground(ColorMuted).Render(" [↑ / ↓] Navegar  •  [Enter] Seleccionar Proveedor  •  [Esc] Volver a Configuraciones ")
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(hints))
	sb.WriteString("\n")

	return sb.String()
}

func (m Model) renderAPIKeyInputView() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}
	innerWidth := convWidth - 6
	if innerWidth > 82 {
		innerWidth = 82
	}
	if innerWidth < 40 {
		innerWidth = 40
	}

	var sb strings.Builder
	sb.WriteString("\n")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(
		titleStyle.Render(fmt.Sprintf("OZYASIST >> CONFIGURAR CLAVE API: %s", strings.ToUpper(m.apiKeyTarget))),
	))
	sb.WriteString("\n\n")

	var formSb strings.Builder
	header := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("── [INGRESO INTERACTIVO DE CLAVE API] ──")
	formSb.WriteString(header + "\n\n")

	promptInfo := lipgloss.NewStyle().Foreground(ColorText).Render(
		fmt.Sprintf("Escribe o pega tu clave para %s (Ctrl+V soportado):\nSe registrará en memoria y se guardará de forma persistente.", strings.ToUpper(m.apiKeyTarget)),
	)
	formSb.WriteString(promptInfo + "\n\n")

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Width(innerWidth - 6).
		Render(m.apiKeyInput.View())

	formSb.WriteString(inputBox + "\n\n")

	infoText := lipgloss.NewStyle().Foreground(ColorMuted).Render("Nota: Los caracteres se ofuscan con viñetas por seguridad.")
	formSb.WriteString(infoText + "\n")

	border := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2)

	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(border.Render(formSb.String())))
	sb.WriteString("\n")

	hints := lipgloss.NewStyle().Foreground(ColorMuted).Render(" [Enter] Guardar Clave  •  [Esc] Cancelar y Regresar ")
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(hints))
	sb.WriteString("\n")

	return sb.String()
}

func (m Model) renderProfileEditView() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}
	innerWidth := convWidth - 6
	if innerWidth > 82 {
		innerWidth = 82
	}
	if innerWidth < 40 {
		innerWidth = 40
	}

	var sb strings.Builder
	sb.WriteString("\n")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(
		titleStyle.Render("OZYASIST >> EDITOR INTERACTIVO DE PERFIL DE USUARIO // SQLite WAL"),
	))
	sb.WriteString("\n\n")

	// Obtener perfil actual de SQLite
	currProfile := "Sin perfil registrado aún."
	if db.DB != nil {
		if u, err := db.GetUser(db.DefaultUserID()); err == nil && u != nil && strings.TrimSpace(u.ProfileMd) != "" {
			currProfile = strings.TrimSpace(u.ProfileMd)
		}
	}

	var formSb strings.Builder
	header := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("── [DIRECTIVAS Y PREFERENCIAS PERSISTENTES] ──")
	formSb.WriteString(header + "\n\n")

	profHeader := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent).Render("Perfil actual en SQLite:")
	formSb.WriteString(profHeader + "\n")

	// Mostrar perfil actual en caja sutil
	profileCard := lipgloss.NewStyle().
		Background(ColorMessageBg).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeftForeground(ColorPrimary).
		Padding(0, 1).
		Width(innerWidth - 8).
		Render(currProfile)
	formSb.WriteString(profileCard + "\n\n")

	promptInfo := lipgloss.NewStyle().Foreground(ColorText).Render(
		"Escribe una nueva directiva o preferencia para tu perfil:\n(ej: 'Trabajo en Go y React, respuestas breves y código modular')",
	)
	formSb.WriteString(promptInfo + "\n\n")

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Width(innerWidth - 6).
		Render(m.profileInput.View())

	formSb.WriteString(inputBox + "\n")

	border := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2)

	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(border.Render(formSb.String())))
	sb.WriteString("\n")

	hints := lipgloss.NewStyle().Foreground(ColorMuted).Render(" [Enter] Agregar Directiva y Guardar en SQLite  •  [Esc] Volver a Configuraciones ")
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(hints))
	sb.WriteString("\n")

	return sb.String()
}

// getFilteredHistoryChats retorna las sesiones filtradas por el término de búsqueda actual
func (m Model) getFilteredHistoryChats() []models.Chat {
	term := strings.ToLower(strings.TrimSpace(m.historySearchInput.Value()))
	if term == "" {
		return m.historyChats
	}
	var filtered []models.Chat
	for _, c := range m.historyChats {
		matchID := strings.Contains(strings.ToLower(c.ID), term)
		matchName := strings.Contains(strings.ToLower(c.Name), term)
		matchProv := strings.Contains(strings.ToLower(c.Provider), term)
		matchModel := strings.Contains(strings.ToLower(c.Model), term)
		if matchID || matchName || matchProv || matchModel {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// renderChatHistoryView renderiza la pantalla de historial de conversaciones navegable con buscador interactivo.
// Muestra sesiones guardadas en SQLite con fecha, código y tema si existe.
// Permite filtrar en tiempo real, reanudar, eliminar o crear una nueva sesión.
func (m Model) renderChatHistoryView() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}
	innerWidth := convWidth - 6
	if innerWidth > 86 {
		innerWidth = 86
	}
	if innerWidth < 40 {
		innerWidth = 40
	}

	var sb strings.Builder
	sb.WriteString("\n")

	// Título superior minimalista
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(
		titleStyle.Render("OZYASIST >> HISTORIAL DE CONVERSACIONES // ZERO-DOCKER KERNEL v7.0"),
	))
	sb.WriteString("\n\n")

	border := lipgloss.NewStyle().
		Width(innerWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	var listSb strings.Builder
	listHeader := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("── [SESIONES GUARDADAS] ──")
	listSb.WriteString(listHeader + "\n\n")

	// Barra de búsqueda en vivo
	searchBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Render(" BUSCAR ")
	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1).
		Width(innerWidth - 8).
		Render(fmt.Sprintf("%s %s", searchBadge, m.historySearchInput.View()))
	listSb.WriteString(searchBox + "\n\n")

	filtered := m.getFilteredHistoryChats()

	// Entradas fijas al final de la lista de chats filtrados
	const extraNew = "[ NUEVA CONVERSACION ]"
	const extraBack = "[ VOLVER AL MENU PRINCIPAL ]"

	totalItems := len(filtered) + 2 // chats filtrados + nueva + volver

	for i := 0; i < totalItems; i++ {
		selected := i == m.chatHistoryIndex

		var label string
		var sublabel string

		if i < len(filtered) {
			c := filtered[i]
			dateStr := c.CreatedAt.Format("02 Jan 15:04")
			shortID := c.ID
			if len(shortID) >= 8 {
				shortID = shortID[:8]
			}
			hasRealTopic := c.Name != "" && c.Name != "TUI Session" && c.Name != "Nueva Sesion" && !strings.HasPrefix(c.Name, "Sesion ")

			if hasRealTopic {
				name := c.Name
				if len(name) > 36 {
					name = name[:33] + "..."
				}
				label = fmt.Sprintf("[%s]  #%s · %s", dateStr, shortID, name)
			} else {
				label = fmt.Sprintf("[%s]  #%s", dateStr, shortID)
			}

			sublabel = fmt.Sprintf("     ID: %s  |  Proveedor: %s", shortID, c.Provider)
			if c.Model != "" {
				sublabel += "  |  Modelo: " + c.Model
			}
		} else if i == len(filtered) {
			label = extraNew
			sublabel = "     Inicia una nueva conversacion limpia"
		} else {
			label = extraBack
			sublabel = "     Regresa al menu principal"
		}

		if selected {
			// Verificar si este item está pendiente de confirmación de borrado
			isConfirmingDelete := i < len(filtered) &&
				m.historyConfirmDelete == filtered[i].ID

			cursor := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(">> ")
			labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary)
			rowTitle := fmt.Sprintf("%s%s", cursor, labelStyle.Render(" "+label+" "))
			rowSub := lipgloss.NewStyle().Foreground(ColorText).Render(sublabel)

			listSb.WriteString(rowTitle + "\n" + rowSub + "\n")

			if isConfirmingDelete {
				confirmBox := lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#181e00")).
					Background(lipgloss.Color("#f1c40f")).
					Padding(0, 2).
					Render("  CONFIRMAR ELIMINACION  [s] Eliminar  [n / Esc] Cancelar  ")
				listSb.WriteString("     " + confirmBox + "\n")
			}
			listSb.WriteString("\n")
		} else {
			cursor := "   "
			labelStyleNormal := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
			rowTitle := fmt.Sprintf("%s%s", cursor, labelStyleNormal.Render(label))
			rowSub := lipgloss.NewStyle().Foreground(ColorMuted).Render(sublabel)
			listSb.WriteString(rowTitle + "\n" + rowSub + "\n\n")
		}
	}

	if len(filtered) == 0 {
		if len(m.historyChats) == 0 {
			emptyMsg := lipgloss.NewStyle().Foreground(ColorMuted).Render("   Sin sesiones previas registradas en la base de datos.")
			listSb.WriteString(emptyMsg + "\n\n")
		} else {
			emptyMsg := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("   No se encontraron conversaciones que coincidan con \"%s\".", m.historySearchInput.Value()))
			listSb.WriteString(emptyMsg + "\n\n")
		}
	}

	rendered := border.Render(listSb.String())
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(rendered))
	sb.WriteString("\n")

	// Atajos
	var hintStr string
	if m.historyConfirmDelete != "" {
		hintStr = " [s] Confirmar Eliminacion  •  [n / Esc] Cancelar "
	} else {
		hintStr = " [Escribe] Filtrar  •  [↑ / ↓] Navegar  •  [Enter] Abrir  •  [d] Eliminar  •  [Esc] Limpiar / Volver "
	}
	hints2 := lipgloss.NewStyle().Foreground(ColorMuted).Render(hintStr)
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(hints2))
	sb.WriteString("\n")

	return sb.String()
}
