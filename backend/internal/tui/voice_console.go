package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ozyassist/backend/internal/agent"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/voice"
)

type VoiceConsoleState int

const (
	VoiceStateIdle VoiceConsoleState = iota
	VoiceStateListening
	VoiceStateThinking
	VoiceStateSpeaking
)

type VoiceEntry struct {
	Speaker string // "user", "agent", "system"
	Text    string
}

type voiceTickMsg time.Time
type voiceAgentEventMsg agent.AgentEvent

var (
	listeningWaveFrames = []string{
		"  ▂ ▃ ▅ ▃ ▂    ▂ ▃ ▅ ▃ ▂  ",
		" ▂ ▃ ▅ ▆ ▅ ▃  ▂ ▃ ▅ ▆ ▅ ▃ ",
		"▃ ▅ ▆ ▇ ▆ ▅ ▃ ▅ ▆ ▇ ▆ ▅ ▃ ",
		" ▂ ▃ ▅ ▆ ▅ ▃  ▂ ▃ ▅ ▆ ▅ ▃ ",
	}

	speakingWaveFrames = []string{
		"█ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █",
		"▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇",
		"▃ ▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃ ▂ ▃ ▅",
		"▂ ▃ ▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃ ▂ ▃",
		"▃ ▂ ▃ ▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃ ▂",
		"▅ ▃ ▂ ▃ ▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃",
		"▇ ▅ ▃ ▂ ▃ ▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅",
	}

	styleLimeHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#181e00")).
			Background(lipgloss.Color("#d1f107")).
			Bold(true).
			Padding(0, 1)

	styleLimeText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d1f107")).
			Bold(true)

	styleDimText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	styleCardBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#333333")).
			Padding(1, 2)

	styleActiveCardBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#d1f107")).
				Padding(1, 2)
)

type VoiceConsoleModel struct {
	provider        providers.Provider
	chat            *models.Chat
	state           VoiceConsoleState
	waveFrame       int
	activeSessionID string
	activeUtterance string
	cancelTurn      context.CancelFunc
	history         []VoiceEntry
	input           textinput.Model
	width           int
	height          int
	program         *tea.Program
}

func NewVoiceConsoleModel(prov providers.Provider, chat *models.Chat) VoiceConsoleModel {
	ti := textinput.New()
	ti.Placeholder = "Escribe una orden o habla 'Hey Ozy'... [Esc para cancelar, Ctrl+C para salir]"
	ti.Focus()
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#d1f107")).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#eeeeee"))

	return VoiceConsoleModel{
		provider: prov,
		chat:     chat,
		state:    VoiceStateIdle,
		history:  make([]VoiceEntry, 0),
		input:    ti,
		width:    80,
		height:   24,
	}
}

func (m VoiceConsoleModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.tickCmd(),
	)
}

func (m VoiceConsoleModel) tickCmd() tea.Cmd {
	return tea.Tick(90*time.Millisecond, func(t time.Time) tea.Msg {
		return voiceTickMsg(t)
	})
}

func (m VoiceConsoleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = m.width - 6
		return m, nil

	case voiceTickMsg:
		m.waveFrame++
		cmds = append(cmds, m.tickCmd())
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlQ:
			if m.cancelTurn != nil {
				m.cancelTurn()
			}
			if m.activeSessionID != "" {
				agent.CancelSession(m.activeSessionID)
			}
			return m, tea.Quit

		case tea.KeyEsc:
			// Barge-in: Cancelar de inmediato cualquier tarea y silenciar el habla
			if m.activeSessionID != "" || m.state == VoiceStateSpeaking || m.state == VoiceStateThinking {
				if m.cancelTurn != nil {
					m.cancelTurn()
					m.cancelTurn = nil
				}
				if m.activeSessionID != "" {
					agent.CancelSession(m.activeSessionID)
					m.activeSessionID = ""
				}
				m.state = VoiceStateIdle
				m.activeUtterance = ""
				m.history = append(m.history, VoiceEntry{
					Speaker: "system",
					Text:    "[CANCELADO] Respuesta interrumpida y silenciada por el usuario.",
				})
			}
			return m, nil

		case tea.KeyEnter:
			val := strings.TrimSpace(m.input.Value())
			if val == "" {
				return m, nil
			}

			// Manejo de comando /cancel escrito
			if val == "/cancel" {
				m.input.SetValue("")
				if m.cancelTurn != nil {
					m.cancelTurn()
					m.cancelTurn = nil
				}
				if m.activeSessionID != "" {
					agent.CancelSession(m.activeSessionID)
					m.activeSessionID = ""
				}
				m.state = VoiceStateIdle
				m.activeUtterance = ""
				m.history = append(m.history, VoiceEntry{
					Speaker: "system",
					Text:    "[CANCELADO] Tarea cancelada por comando /cancel.",
				})
				return m, nil
			}

			m.input.SetValue("")
			m.startTurn(val)
			return m, nil
		}

	case voiceAgentEventMsg:
		evt := agent.AgentEvent(msg)
		switch evt.Type {
		case "voice:sentence":
			m.state = VoiceStateSpeaking
			m.activeUtterance = evt.Content

		case "state:sync":
			if evt.State == "thinking" && m.state != VoiceStateSpeaking {
				m.state = VoiceStateThinking
			}

		case "tool:call":
			m.state = VoiceStateThinking
			m.history = append(m.history, VoiceEntry{
				Speaker: "system",
				Text:    fmt.Sprintf("[HERRAMIENTA] Ejecutando: %s", evt.ToolName),
			})

		case "agent:completed":
			m.state = VoiceStateIdle
			m.activeUtterance = ""
			m.activeSessionID = ""
			m.cancelTurn = nil
			if evt.Content != "" {
				m.history = append(m.history, VoiceEntry{
					Speaker: "agent",
					Text:    evt.Content,
				})
			}

		case "error":
			m.state = VoiceStateIdle
			m.activeUtterance = ""
			m.activeSessionID = ""
			m.cancelTurn = nil
			m.history = append(m.history, VoiceEntry{
				Speaker: "system",
				Text:    fmt.Sprintf("[ERROR] %s", evt.Error),
			})
		}
		return m, nil
	}

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *VoiceConsoleModel) startTurn(prompt string) {
	// Cancelar cualquier turno previo
	if m.cancelTurn != nil {
		m.cancelTurn()
	}
	if m.activeSessionID != "" {
		agent.CancelSession(m.activeSessionID)
	}

	m.history = append(m.history, VoiceEntry{
		Speaker: "user",
		Text:    prompt,
	})
	m.state = VoiceStateThinking
	m.activeUtterance = ""

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelTurn = cancel

	prog := m.program

	params := agent.AgentLoopParams{
		Provider:        m.provider,
		Chat:            m.chat,
		UserMessage:     prompt,
		PermissionLevel: "autonomous",
		VoiceMode:       true,
		Emit: func(evt agent.AgentEvent) {
			if prog != nil {
				prog.Send(voiceAgentEventMsg(evt))
			}
		},
	}

	sessionID := agent.StartAgentLoop(ctx, params)
	m.activeSessionID = sessionID
}

func (m VoiceConsoleModel) View() string {
	var sb strings.Builder

	// 1. Encabezado
	headerBadge := styleLimeHeader.Render("OZY VOICE")
	headerTitle := fmt.Sprintf(" %s · Asistente Autónomo de Voz en Tiempo Real", headerBadge)
	sb.WriteString("\n" + headerTitle + "\n")
	sb.WriteString(styleDimText.Render(strings.Repeat("─", max(40, m.width-4))) + "\n\n")

	// 2. Tarjeta central de estado y onda sonora
	var statusLabel string
	var waveText string
	var subtitleText string
	cardStyle := styleCardBox

	switch m.state {
	case VoiceStateSpeaking:
		cardStyle = styleActiveCardBox
		statusLabel = styleLimeText.Render("[OZY HABLANDO]")
		idx := m.waveFrame % len(speakingWaveFrames)
		waveText = styleLimeText.Render(speakingWaveFrames[idx])
		if m.activeUtterance != "" {
			subtitleText = fmt.Sprintf("\n%s\n", lipgloss.NewStyle().Italic(true).Render(fmt.Sprintf("\"%s\"", m.activeUtterance)))
		}

	case VoiceStateThinking:
		statusLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#f1c40f")).Bold(true).Render("[PROCESANDO ORDEN...]")
		spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		idx := m.waveFrame % len(spinnerFrames)
		waveText = fmt.Sprintf("%s Analizando y ejecutando acciones...", spinnerFrames[idx])

	default: // Idle / Listening
		statusLabel = styleDimText.Render("[ESCUCHANDO WAKE WORD · 'Hey Ozy']")
		idx := m.waveFrame % len(listeningWaveFrames)
		waveText = styleDimText.Render(listeningWaveFrames[idx])
		subtitleText = "\nPuedes hablar 'Hey Ozy' o escribir una orden directamente en la barra inferior.\n"
	}

	cardContent := fmt.Sprintf("%s\n\n  %s\n%s", statusLabel, waveText, subtitleText)
	sb.WriteString(cardStyle.Render(cardContent) + "\n\n")

	// 3. Registro de conversación reciente
	sb.WriteString(styleDimText.Render("HISTORIAL RECIENTE:") + "\n")
	startIdx := 0
	if len(m.history) > 6 {
		startIdx = len(m.history) - 6
	}
	for _, entry := range m.history[startIdx:] {
		switch entry.Speaker {
		case "user":
			sb.WriteString(fmt.Sprintf("  %s %s\n", styleDimText.Render("TU:"), entry.Text))
		case "agent":
			sb.WriteString(fmt.Sprintf("  %s %s\n", styleLimeText.Render("OZY:"), entry.Text))
		case "system":
			sb.WriteString(fmt.Sprintf("  %s\n", styleDimText.Render(entry.Text)))
		}
	}
	if len(m.history) == 0 {
		sb.WriteString(styleDimText.Render("  (Sin mensajes aun. Escribe o habla para comenzar)") + "\n")
	}

	// 4. Barra inferior de entrada y controles
	sb.WriteString("\n" + styleDimText.Render(strings.Repeat("─", max(40, m.width-4))) + "\n")
	sb.WriteString(m.input.View() + "\n")
	controls := styleDimText.Render("[Enter: Enviar]  ·  [Esc: Cancelar / Silenciar]  ·  [/cancel]  ·  [Ctrl+C: Salir]")
	sb.WriteString(controls + "\n")

	return sb.String()
}

// RunVoiceConsole arranca la interfaz TUI interactiva dedicada de voz
func RunVoiceConsole(prov providers.Provider, chat *models.Chat) error {
	m := NewVoiceConsoleModel(prov, chat)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.program = p

	// Iniciar detector de Wake Word pasivo si STT está disponible
	cfg := voice.AutoDetectConfig()
	if cfg.HasSTT() {
		engine := voice.NewNativeVoiceEngine(cfg, func(ctx context.Context, command string, onDelta func(string), onComplete func(string)) {
			p.Send(tea.KeyMsg{
				Type:  tea.KeyEnter,
				Runes: []rune(command),
			})
		})
		engineCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = engine.Start(engineCtx)
		defer engine.Stop()
	}

	_, err := p.Run()
	return err
}
