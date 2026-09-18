package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ozyassist/backend/internal/agent"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

type UIState int

const (
	StateStartMenu UIState = iota
	StateSettingsMenu
	StateProviderMenu
	StateAPIKeySelect
	StateAPIKeyInput
	StateProfileEdit
	StateChatHistory // Pantalla de historial de conversaciones
	StateIdle
	StateThinking
	StateExecutingTool
	StateStreaming
	StateAwaitingApproval
)

type InteractiveCard struct {
	Title         string
	Tabs          []string // e.g. ["Prioridad", "Tareas", "Riesgos", "Confirmar"]
	ActiveTab     int
	Question      string   // e.g. "¿Qué área o tarea deseas priorizar?"
	Options       []string // e.g. ["1. Analizar proyectos locales", "2. Escanear rutas en RAM", ...]
	SelectedIndex int      // Fila seleccionada actualmente
	IsAnswered    bool     // Si ya fue respondida
	Answer        string   // Respuesta seleccionada
}

type ChatEntry struct {
	Role        string // user, assistant, system, tool
	Content     string
	Thinking    string // hilo de pensamiento del modelo
	ToolName    string
	ToolInput   string
	ToolSuccess bool
	DurationMs  int64
	Card        *InteractiveCard // Tarjeta interactiva de decisión/pregunta
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
	menuIndex       int    // 0: Iniciar Conversacion, 1: Historial, 2: Configuraciones, 3: Salir
	settingsIndex   int    // Índice en menú de configuraciones
	providerIndex   int    // Índice en el selector interactivo de proveedores
	apiKeyIndex     int    // Índice en el selector de claves API
	apiKeyTarget    string // Proveedor seleccionado para ingresar clave (ej: "groq")
	apiKeyInput     textinput.Model // Input para escribir/pegar la clave
	profileInput    textinput.Model // Input para escribir directiva de perfil
	settingsNotice  string // Notificacion o resultado de accion en configuraciones/proveedores
	viewport        viewport.Model
	textarea        textarea.Model
	spinner         spinner.Model
	entries         []ChatEntry
	promptHistory       []string // Historial de prompts escritos por el usuario (para navegación con flechas)
	promptHistoryIndex  int      // Índice de navegación en el historial de prompts
	messageQueue    []QueuedPrompt
	currentStream   string
	currentThinking string
	showThinking    bool
	activeToolName  string
	activeToolInput string

	// Historial de conversaciones
	historyChats          []models.Chat // Lista de chats cargada desde SQLite
	chatHistoryIndex      int           // Fila seleccionada en la pantalla de historial
	historyConfirmDelete  string        // ID del chat pendiente de confirmación de borrado ("" = ninguno)
	historySearchInput    textinput.Model // Input para filtrar en tiempo real el historial

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
	activeCard      *InteractiveCard // Tarjeta interactiva actualmente en foco para selección (opcional)
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

	// Input interactivo para clave API
	keyTi := textinput.New()
	keyTi.Placeholder = "Pega o escribe tu API Key aquí..."
	keyTi.Prompt = "❯ "
	keyTi.CharLimit = 512
	keyTi.EchoMode = textinput.EchoPassword
	keyTi.EchoCharacter = '•'

	// Input interactivo para perfil de usuario
	profTi := textinput.New()
	profTi.Placeholder = "Escribe una directiva o preferencia (ej: stack Go/React, respuestas concisas)..."
	profTi.Prompt = "❯ "
	profTi.CharLimit = 1024

	// Input para búsqueda en historial de conversaciones
	searchTi := textinput.New()
	searchTi.Placeholder = "Buscar por tema, código o proveedor..."
	searchTi.Prompt = "❯ "
	searchTi.CharLimit = 128

	initStatus := "Listo para actuar"
	if voiceActive {
		initStatus = "Escuchando Wake Word ('Hey Ozy')..."
	}

	welcomeContent := "¡Hola! Soy OzyAssist, tu asistente autónomo de escritorio, código y cowork para Windows.\n\nEstoy conectado y listo con arquitectura Zero-Docker, memoria continua y herramientas de sistema.\nEscribe libremente tu instrucción o consulta para comenzar."

	m := Model{
		state:              StateStartMenu,
		menuIndex:          0,
		settingsIndex:      0,
		settingsNotice:     "",
		textarea:           ta,
		apiKeyInput:        keyTi,
		profileInput:       profTi,
		historySearchInput: searchTi,
		spinner:            sp,
		provider:           prov,
		chat:               chat,
		permissionLevel:    "autonomous",
		voiceEnabled:       voiceActive,
		systemStatus:       initStatus,
		promptHistoryIndex: -1,
		activeCard:         nil,
		entries: []ChatEntry{
			{
				Role:    "assistant",
				Content: welcomeContent,
			},
		},
	}

	return m
}

func (m Model) isWelcomeState() bool {
	return false
}

// isBusy retorna true si el modelo está procesando un turno agéntico o ejecutando herramientas.
func (m Model) isBusy() bool {
	return m.state == StateThinking || m.state == StateExecutingTool || m.state == StateStreaming || m.state == StateAwaitingApproval || m.loopSessionID != ""
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
