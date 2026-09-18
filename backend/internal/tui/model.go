package tui

import (
	"context"
	"strings"
	"time"

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
type loopStartedMsg struct {
	sessionID string
	cancel    context.CancelFunc
}
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

// QueuedPrompt representa una orden encolada mientras el agente está ocupado.
type QueuedPrompt struct {
	Prompt  string
	IsVoice bool
	AddedAt time.Time
}

type Model struct {
	state           UIState
	viewport        viewport.Model
	textarea        textarea.Model
	spinner         spinner.Model
	entries         []ChatEntry
	promptHistory   []string
	historyIndex    int
	messageQueue    []QueuedPrompt
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

	welcomeText := "[SISTEMA] OZYASIST KERNEL v2.6 INICIALIZADO"
	initStatus := "Listo para actuar"
	if voiceActive {
		welcomeText = "[SISTEMA] OZYASIST KERNEL v2.6 INICIALIZADO (VOZ ACTIVA)"
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

func (m Model) isWelcomeState() bool {
	if len(m.entries) == 0 {
		return true
	}
	if len(m.entries) == 1 && m.entries[0].Role == "system" && strings.HasPrefix(m.entries[0].Content, "[SISTEMA] OZYASIST") {
		return true
	}
	return false
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

// EnqueuePrompt añade un prompt a la cola de espera y retorna la nueva longitud.
func (m *Model) EnqueuePrompt(prompt string, isVoice bool) int {
	m.messageQueue = append(m.messageQueue, QueuedPrompt{
		Prompt:  prompt,
		IsVoice: isVoice,
		AddedAt: time.Now(),
	})
	return len(m.messageQueue)
}

// DequeuePrompt extrae el primer prompt en cola (FIFO).
func (m *Model) DequeuePrompt() (QueuedPrompt, bool) {
	if len(m.messageQueue) == 0 {
		return QueuedPrompt{}, false
	}
	item := m.messageQueue[0]
	m.messageQueue = m.messageQueue[1:]
	return item, true
}

// ClearQueue descarta todos los mensajes en cola y retorna la cantidad descartada.
func (m *Model) ClearQueue() int {
	count := len(m.messageQueue)
	m.messageQueue = nil
	return count
}

// QueueLen retorna la cantidad de mensajes pendientes en cola.
func (m *Model) QueueLen() int {
	return len(m.messageQueue)
}
