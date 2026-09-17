package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ozyassist/backend/internal/agent"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

type UIState int

const (
	StateIdle UIState = iota
	StateThinking
	StateExecutingTool
	StateStreaming
	StateAwaitingApproval
)

type ChatEntry struct {
	Role        string // user, assistant, system, tool
	Content     string
	Thinking    string // hilo de pensamiento del modelo
	ToolName    string
	ToolInput   string
	ToolSuccess bool
	DurationMs  int64
}

// Custom Bubble Tea Messages
type agentEventMsg agent.AgentEvent
type loopFinishedMsg struct{}
type errMsg struct{ err error }

// VoiceCommandMsg transporta una orden reconocida por voz a la TUI
type VoiceCommandMsg struct {
	Command string
}

// VoiceStatusMsg actualiza el estado de escucha de voz
type VoiceStatusMsg struct {
	Listening bool
	Status    string
}

type Model struct {
	state           UIState
	viewport        viewport.Model
	textarea        textarea.Model
	spinner         spinner.Model
	entries         []ChatEntry
	promptHistory   []string
	historyIndex    int
	currentStream   string
	currentThinking string
	showThinking    bool
	activeToolName  string
	activeToolInput string

	provider        providers.Provider
	chat            *models.Chat
	permissionLevel string
	voiceEnabled    bool
	systemStatus    string

	width           int
	height          int
	ready           bool

	loopSessionID   string
	loopCancel      context.CancelFunc
}

func InitialModel(prov providers.Provider, chat *models.Chat, voiceActive bool) Model {
	ta := textarea.New()
	ta.Placeholder = "Escribe una instrucción (/help para comandos, o habla si la voz está activa)..."
	ta.Focus()
	ta.Prompt = "❯ "
	ta.CharLimit = 4096
	ta.SetWidth(80)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	welcomeText := "🚀 ¡Hola! Bienvenido a OzyAssist CLI.\nUn asistente agéntico listo para ayudarte con tu sistema y herramientas.\n💡 Escribe una instrucción, usa /help para comandos o /key groq para activar voz ('Hey Ozy')."
	initStatus := "Listo para actuar"
	if voiceActive {
		welcomeText = "🚀 ¡Hola! Bienvenido a OzyAssist CLI.\nControl del SO y escucha de voz 'Hey Ozy' activos.\n💡 Escribe una instrucción, usa /help o habla directamente 'Hey Ozy'."
		initStatus = "Escuchando Wake Word ('Hey Ozy')..."
	}

	m := Model{
		state:           StateIdle,
		textarea:        ta,
		spinner:         sp,
		provider:        prov,
		chat:            chat,
		permissionLevel: "autonomous",
		voiceEnabled:    voiceActive,
		systemStatus:    initStatus,
		historyIndex:    -1,
		entries: []ChatEntry{
			{
				Role:    "system",
				Content: welcomeText,
			},
		},
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}
