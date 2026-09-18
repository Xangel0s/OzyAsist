package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/system"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel(nil, nil, false)
	if m.state != StateIdle {
		t.Errorf("Expected initial state StateIdle, got %v", m.state)
	}
	if m.voiceEnabled {
		t.Errorf("Expected voiceEnabled false when passed false")
	}

	mActive := InitialModel(nil, nil, true)
	if !mActive.voiceEnabled {
		t.Errorf("Expected voiceEnabled true when passed true")
	}

	if len(m.entries) == 0 {
		t.Errorf("Expected welcome entry in entries")
	}

	// Test slash command /help
	out := m.handleSlashCommand("/help")
	if out == "" {
		t.Errorf("Expected non-empty output for /help")
	}

	// Test slash command /perm
	outPerm := m.handleSlashCommand("/perm autonomous")
	if m.permissionLevel != "autonomous" {
		t.Errorf("Expected permissionLevel autonomous, got %s (output: %s)", m.permissionLevel, outPerm)
	}

	// Test slash command /mcp
	outMCP := m.handleSlashCommand("/mcp")
	if outMCP == "" {
		t.Errorf("Expected non-empty output for /mcp")
	}

	// Test slash command /tools
	outTools := m.handleSlashCommand("/tools")
	if outTools == "" {
		t.Errorf("Expected non-empty output for /tools")
	}

	// Test slash command /voice without controller
	outVoice := m.handleSlashCommand("/voice")
	if outVoice == "" {
		t.Errorf("Expected output for /voice")
	}

	// Test slash command /key groq
	outKeyGroq := m.handleSlashCommand("/key groq gsk_test123")
	if outKeyGroq == "" {
		t.Errorf("Expected output for /key groq")
	}

	// Test slash command /provider
	outProvList := m.handleSlashCommand("/provider")
	if outProvList == "" {
		t.Errorf("Expected output for /provider")
	}

	// Test slash command /key cohere
	outKeyCohere := m.handleSlashCommand("/key cohere dummy_cohere_test_key_12345")
	if outKeyCohere == "" {
		t.Errorf("Expected output for /key cohere")
	}
	if m.provider == nil || m.provider.Name() != "cohere" {
		t.Errorf("Expected active provider to be cohere, got %v", m.provider)
	}

	// Test window resize message
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model := newM.(Model)
	if !model.ready {
		t.Errorf("Expected model to be ready after WindowSizeMsg")
	}
}

func TestCleanAssistantText(t *testing.T) {
	raw := "<thought>\nAnalizando el problema del usuario...\n</thought>\n\n\n\nHola, he organizado tus archivos.\n\n\nQue tengas buen día."
	cleaned := cleanAssistantText(raw)
	expected := "Hola, he organizado tus archivos.\n\nQue tengas buen día."
	if cleaned != expected {
		t.Errorf("Expected cleaned text:\n%q\nGot:\n%q", expected, cleaned)
	}
}

func TestWrapLine(t *testing.T) {
	longLine := "Get-Process : No se encuentra ningún proceso con el nombre \"notepad\". Compruebe el nombre del proceso y vuelva a ejecutar el cmdlet."
	maxWidth := 40
	wrapped := wrapLine(longLine, maxWidth)
	if len(wrapped) < 3 {
		t.Errorf("Expected at least 3 lines, got %d: %v", len(wrapped), wrapped)
	}
	for i, line := range wrapped {
		runes := []rune(line)
		if len(runes) > maxWidth {
			t.Errorf("Line %d exceeds maxWidth %d (len %d): %s", i, maxWidth, len(runes), line)
		}
	}
}

func TestWrapLine_LongUnbrokenWord(t *testing.T) {
	longWord := "https://github.com/ozyassist/backend/internal/agent/very/deeply/nested/directory/path/file.go"
	maxWidth := 30
	wrapped := wrapLine(longWord, maxWidth)
	if len(wrapped) < 3 {
		t.Errorf("Expected unbroken word to be wrapped, got %d parts: %v", len(wrapped), wrapped)
	}
	for i, line := range wrapped {
		runes := []rune(line)
		if len(runes) > maxWidth {
			t.Errorf("Part %d exceeds maxWidth %d (len %d): %s", i, maxWidth, len(runes), line)
		}
	}
}

func TestWrapContent_PreservesIndentation(t *testing.T) {
	content := "  - Primer punto con una explicación bastante larga para probar el comportamiento del wrap\n    • Subpunto indentado"
	wrapped := wrapContent(content, 45)
	if len(wrapped) < 3 {
		t.Errorf("Expected wrapped lines >= 3, got %d: %v", len(wrapped), wrapped)
	}
}

func TestMessageQueue_EnqueueAndDequeue(t *testing.T) {
	m := InitialModel(nil, nil, false)
	if m.QueueLen() != 0 {
		t.Errorf("Expected empty queue initially, got %d", m.QueueLen())
	}

	m.EnqueuePrompt("primera tarea", false)
	m.EnqueuePrompt("segunda tarea", true)
	if m.QueueLen() != 2 {
		t.Fatalf("Expected queue length 2, got %d", m.QueueLen())
	}

	item1, ok1 := m.DequeuePrompt()
	if !ok1 || item1.Prompt != "primera tarea" || item1.IsVoice {
		t.Errorf("Unexpected item1: %+v", item1)
	}

	item2, ok2 := m.DequeuePrompt()
	if !ok2 || item2.Prompt != "segunda tarea" || !item2.IsVoice {
		t.Errorf("Unexpected item2: %+v", item2)
	}

	_, ok3 := m.DequeuePrompt()
	if ok3 {
		t.Errorf("Expected empty dequeue to return false")
	}

	m.EnqueuePrompt("tarea 3", false)
	m.EnqueuePrompt("tarea 4", false)
	cleared := m.ClearQueue()
	if cleared != 2 || m.QueueLen() != 0 {
		t.Errorf("Expected 2 cleared items, got %d (len %d)", cleared, m.QueueLen())
	}
}

func TestMessageQueue_SlashCommands(t *testing.T) {
	m := InitialModel(nil, nil, false)

	// Test /queue when empty
	outEmpty := m.handleSlashCommand("/queue")
	if outEmpty != "ℹ️ La cola de mensajes está vacía." {
		t.Errorf("Unexpected /queue empty output: %s", outEmpty)
	}

	// Enqueue items
	m.EnqueuePrompt("analizar logs", false)
	m.EnqueuePrompt("ejecutar backup", true)

	outList := m.handleSlashCommand("/queue")
	if outList == "" || len(m.messageQueue) != 2 {
		t.Errorf("Expected non-empty queue list with 2 items, got: %s", outList)
	}

	// Clear queue
	outClear := m.handleSlashCommand("/clearqueue")
	if m.QueueLen() != 0 {
		t.Errorf("Expected queue len 0 after /clearqueue, got %d (output: %s)", m.QueueLen(), outClear)
	}

	// Test /cancel when idle
	outCancelIdle := m.handleSlashCommand("/cancel")
	if outCancelIdle != "ℹ️ No hay ninguna petición activa ni mensajes en cola." {
		t.Errorf("Unexpected /cancel idle output: %s", outCancelIdle)
	}
}

func TestMessageQueue_KeyEscCancelsActiveState(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.state = StateExecutingTool
	m.loopSessionID = "dummy-session"

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newM.(Model)

	if model.state != StateIdle {
		t.Errorf("Expected state to be StateIdle after Esc, got %v", model.state)
	}
	if model.loopSessionID != "" {
		t.Errorf("Expected loopSessionID to be cleared, got %s", model.loopSessionID)
	}
}

func TestMessageQueue_KeyEnterEnqueuesWhenBusy(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.state = StateThinking
	m.textarea.SetValue("tarea secundaria mientras piensa")

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newM.(Model)

	if cmd != nil {
		t.Errorf("Expected no command dispatched immediately when enqueuing")
	}
	if model.QueueLen() != 1 {
		t.Fatalf("Expected 1 item in queue, got %d", model.QueueLen())
	}
	item, _ := model.DequeuePrompt()
	if item.Prompt != "tarea secundaria mientras piensa" {
		t.Errorf("Expected queued prompt 'tarea secundaria mientras piensa', got '%s'", item.Prompt)
	}
}

func TestMessageQueue_VoiceCommandEnqueuesWhenBusy(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.state = StateStreaming

	newM, _ := m.Update(VoiceCommandMsg{Command: "apaga la música"})
	model := newM.(Model)

	if model.QueueLen() != 1 {
		t.Fatalf("Expected 1 item in queue for voice command, got %d", model.QueueLen())
	}
	item, _ := model.DequeuePrompt()
	if item.Prompt != "apaga la música" || !item.IsVoice {
		t.Errorf("Expected queued voice item 'apaga la música' with IsVoice true, got %+v", item)
	}
}

func TestMessageQueue_DirectSendInterruptsAndStarts(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.state = StateExecutingTool
	m.textarea.SetValue("/now detén todo y revisa la memoria")

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newM.(Model)

	if cmd == nil {
		t.Errorf("Expected cmd to be non-nil when /now starts a turn immediately")
	}
	if model.state != StateThinking {
		t.Errorf("Expected state to be StateThinking after /now, got %v", model.state)
	}
}

func TestSlashCommands_ProfileAndMemories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ozy-tui-mem-test-*")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "tui_test.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("error initializing test db: %v", err)
	}
	defer db.Close()
	_ = db.EnsureDefaultUser()
	memory.InitContinuousMemory(db.DB, nil)

	m := InitialModel(nil, nil, false)

	// 1. Probar /profile actualización
	updateOut := m.handleSlashCommand("/profile # Lead Developer crmgeofal")
	if !strings.Contains(updateOut, "[PERFIL]") || !strings.Contains(updateOut, "actualizado") {
		t.Errorf("expected profile update confirmation, got: %s", updateOut)
	}

	// 2. Probar /profile consulta
	viewOut := m.handleSlashCommand("/profile")
	if !strings.Contains(viewOut, "Lead Developer crmgeofal") {
		t.Errorf("expected profile content in /profile, got: %s", viewOut)
	}

	// 3. Probar /remember
	rememberOut := m.handleSlashCommand("/remember El backend corre en puerto 4000")
	if !strings.Contains(rememberOut, "[MEMORIA]") || !strings.Contains(rememberOut, "Recordado") {
		t.Errorf("expected memory confirmation, got: %s", rememberOut)
	}

	// 4. Probar /memories consulta
	memoriesOut := m.handleSlashCommand("/memories")
	if !strings.Contains(memoriesOut, "[MEMORIA CONTINUA]") || !strings.Contains(memoriesOut, "puerto 4000") {
		t.Errorf("expected remembered fact in /memories, got: %s", memoriesOut)
	}

	// 5. Verificar que no contenga emojis en las salidas
	for _, out := range []string{updateOut, viewOut, rememberOut, memoriesOut} {
		for _, r := range out {
			if r >= 0x1F300 && r <= 0x1F9FF {
				t.Errorf("output should not contain emojis, found: %U in '%s'", r, out)
			}
		}
	}
}

func TestSlashCommands_PathsScanDream(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "paths_test.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("error initializing test db: %v", err)
	}
	defer db.Close()
	_ = db.EnsureDefaultUser()

	// Registrar una ruta de prueba
	system.InitDefaultPathRegistry(db.DB)
	system.DefaultPathRegistry().Register(system.IndexedPath{
		ID:       "p1",
		Name:     "crmgeofal",
		FullPath: "C:\\Proyectos\\crmgeofal",
		Category: system.CategoryProject,
	}, false)

	// Inicializar subagente DREAMER
	memory.InitContinuousMemory(db.DB, nil)
	memory.InitDreamerSubagent(nil, memory.DefaultStore(), db.DB, memory.DefaultCache())

	m := InitialModel(nil, nil, false)

	// 1. Probar /paths
	pathsOut := m.handleSlashCommand("/paths")
	if !strings.Contains(pathsOut, "[RUTAS INDEXADAS EN RAM]") || !strings.Contains(pathsOut, "crmgeofal") {
		t.Errorf("expected crmgeofal in /paths, got: %s", pathsOut)
	}

	// 2. Probar /scan
	scanOut := m.handleSlashCommand("/scan")
	if !strings.Contains(scanOut, "[RUTAS]") || !strings.Contains(scanOut, "iniciado") {
		t.Errorf("expected scan confirmation in /scan, got: %s", scanOut)
	}

	// 3. Probar /dream
	dreamOut := m.handleSlashCommand("/dream")
	if !strings.Contains(dreamOut, "[DREAMING]") || !strings.Contains(dreamOut, "iniciado") {
		t.Errorf("expected dreamer confirmation in /dream, got: %s", dreamOut)
	}

	// 4. Verificar ausencia total de emojis
	for _, out := range []string{pathsOut, scanOut, dreamOut} {
		for _, r := range out {
			if r >= 0x1F300 && r <= 0x1F9FF {
				t.Errorf("output should not contain emojis, found: %U in '%s'", r, out)
			}
		}
	}
}

func TestRetroWelcomeHero_TransitionToChat(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true

	// 1. Estado inicial debe ser pantalla de bienvenida retro
	if !m.isWelcomeState() {
		t.Fatalf("se esperaba estado de bienvenida al iniciar")
	}

	hero := m.renderConversation()
	if !strings.Contains(hero, "OZYASIST") || !strings.Contains(hero, "ZERO-DOCKER") {
		t.Errorf("hero debe incluir nombre y arquitectura: %s", hero)
	}
	if !strings.Contains(hero, "/paths") || !strings.Contains(hero, "GROK") {
		t.Errorf("hero debe incluir sugerencias tipo Grok: %s", hero)
	}

	// 2. Verificar ausencia total de emojis en la pantalla de bienvenida
	for _, r := range hero {
		if r >= 0x1F300 && r <= 0x1F9FF {
			t.Errorf("hero no debe contener emojis, encontrado: %U", r)
		}
	}

	// 3. Simular primer mensaje de usuario
	m.textarea.SetValue("¿qué proyectos tengo?")
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updatedModel := newM.(Model)

	// Ya no debe estar en estado de bienvenida
	if updatedModel.isWelcomeState() {
		t.Fatalf("después de enviar mensaje ya no debe ser estado de bienvenida")
	}

	chatView := updatedModel.renderConversation()
	if strings.Contains(chatView, "ROM BIOS 1989-1996") {
		t.Fatalf("la bienvenida retro debió dar paso al chat normal")
	}
	if !strings.Contains(chatView, "TÚ: ") || !strings.Contains(chatView, "¿qué proyectos tengo?") {
		t.Fatalf("el chat normal debe mostrar el mensaje de usuario: %s", chatView)
	}

	// 4. Probar /clear para regresar a la pantalla de bienvenida
	_ = updatedModel.handleSlashCommand("/clear")
	if !updatedModel.isWelcomeState() {
		t.Fatalf("después de /clear debe retornar a estado de bienvenida")
	}
	clearHero := updatedModel.renderConversation()
	if !strings.Contains(clearHero, "OZYASIST") {
		t.Fatalf("/clear debió restaurar la pantalla retro de bienvenida: %s", clearHero)
	}
}




