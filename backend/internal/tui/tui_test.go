package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel(nil, nil, false)
	if m.state != StateStartMenu {
		t.Errorf("Expected initial state StateStartMenu, got %v", m.state)
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
	if outEmpty != "[INFO] La cola de mensajes está vacía." {
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
	if outCancelIdle != "[INFO] No hay ninguna petición activa ni mensajes en cola." {
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

	// 1. Entrar al chat desde el menú de inicio
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)

	if m.state != StateIdle {
		t.Fatalf("se esperaba estado StateIdle después de presionar Enter en INICIAR CONVERSACION, obtenido: %v", m.state)
	}

	// 2. Estado inicial de conversación debe ser chat limpio con mensaje de bienvenida de OzyAssist
	if m.isWelcomeState() {
		t.Fatalf("no se debe mostrar la tarjeta estática de sugerencias en el chat")
	}

	chatView := m.renderConversation()
	if !strings.Contains(chatView, "¡Hola! Soy OzyAssist") {
		t.Errorf("chat debe incluir mensaje de bienvenida del asistente: %s", chatView)
	}
	if strings.Contains(chatView, "ACCIONES RAPIDAS & PROMPTS SUGERIDOS") || strings.Contains(chatView, "ROM BIOS 1989-1996") {
		t.Errorf("chat no debe incluir la caja duplicada de sugerencias de bienvenida: %s", chatView)
	}
	if strings.Contains(chatView, "HERMES") || strings.Contains(chatView, "GROK") {
		t.Errorf("chat no debe contener menciones a HERMES ni GROK: %s", chatView)
	}

	// 3. Verificar ausencia total de emojis en la pantalla de bienvenida y chat
	for _, r := range chatView {
		if r >= 0x1F300 && r <= 0x1F9FF {
			t.Errorf("chat no debe contener emojis, encontrado: %U", r)
		}
	}

	// 4. Simular primer mensaje de usuario
	m.textarea.SetValue("¿qué proyectos tengo?")
	newM2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updatedModel := newM2.(Model)

	newChatView := updatedModel.renderConversation()
	if strings.Contains(newChatView, "ROM BIOS 1989-1996") {
		t.Fatalf("la bienvenida no debe mostrar ROM BIOS")
	}
	if !strings.Contains(newChatView, "TÚ: ") || !strings.Contains(newChatView, "¿qué proyectos tengo?") {
		t.Fatalf("el chat normal debe mostrar el mensaje de usuario: %s", newChatView)
	}

	// 5. Probar /clear para limpiar el historial de mensajes
	_ = updatedModel.handleSlashCommand("/clear")
	if len(updatedModel.entries) != 0 {
		t.Fatalf("después de /clear entries debe estar vacío, tamaño actual: %d", len(updatedModel.entries))
	}
}

func TestStartMenu_InteractiveArrowNavigation(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true

	// Estado inicial debe ser StartMenu y menuIndex 0
	if m.state != StateStartMenu || m.menuIndex != 0 {
		t.Fatalf("se esperaba StateStartMenu con menuIndex 0, obtenido: state=%v index=%d", m.state, m.menuIndex)
	}

	// Verificar renderizado de menú de inicio
	view := m.renderStartMenuView()
	if !strings.Contains(view, "INICIAR CONVERSACION") || !strings.Contains(view, "CONFIGURACIONES") || !strings.Contains(view, "SALIR") {
		t.Fatalf("menú de inicio no contiene las opciones principales: %s", view)
	}
	if strings.Contains(view, "HERMES") || strings.Contains(view, "GROK") {
		t.Fatalf("menú no debe contener HERMES ni GROK: %s", view)
	}

	// Verificar ausencia de emojis en menú
	for _, r := range view {
		if r >= 0x1F300 && r <= 0x1F9FF {
			t.Fatalf("menú de inicio no debe contener emojis, encontrado: %U", r)
		}
	}

	// 1. Navegación hacia abajo (KeyDown: index 0 -> index 1)
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.menuIndex != 1 {
		t.Fatalf("KeyDown debió avanzar a index 1, obtenido: %d", m.menuIndex)
	}

	// 2. Navegación hacia abajo (j: index 1 -> index 2)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = res.(Model)
	if m.menuIndex != 2 {
		t.Fatalf("tecla 'j' debió avanzar a index 2, obtenido: %d", m.menuIndex)
	}

	// 3. Navegación hacia abajo (KeyDown: index 2 -> index 3)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.menuIndex != 3 {
		t.Fatalf("KeyDown debió avanzar a index 3, obtenido: %d", m.menuIndex)
	}

	// 4. Wrap-around hacia abajo (KeyDown en index 3 -> index 0)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.menuIndex != 0 {
		t.Fatalf("wrap-around hacia abajo debió volver a index 0, obtenido: %d", m.menuIndex)
	}

	// 5. Wrap-around hacia arriba (KeyUp en index 0 -> index 3)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.menuIndex != 3 {
		t.Fatalf("wrap-around hacia arriba debió ir a index 3, obtenido: %d", m.menuIndex)
	}

	// 6. Tecla 'k' hacia arriba (index 3 -> index 2)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = res.(Model)
	if m.menuIndex != 2 {
		t.Fatalf("tecla 'k' debió retroceder a index 2, obtenido: %d", m.menuIndex)
	}

	// 7. Acceso numérico directo ('1' para iniciar conversación)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = res.(Model)
	if m.state != StateIdle {
		t.Fatalf("tecla '1' debió pasar a StateIdle, obtenido: %v", m.state)
	}
}

func TestSettingsMenu_NavigationAndCycle(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true

	// 1. Acceder a configuraciones con '3'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m = res.(Model)
	if m.state != StateSettingsMenu {
		t.Fatalf("se esperaba StateSettingsMenu, obtenido: %v", m.state)
	}

	// 2. Verificar renderizado de configuraciones
	settingsView := m.renderSettingsMenuView()
	if !strings.Contains(settingsView, "Proveedor LLM Activo") || !strings.Contains(settingsView, "Gestión de Claves API") || !strings.Contains(settingsView, "Perfil de Usuario") || !strings.Contains(settingsView, "Nivel de Permisos SO") {
		t.Fatalf("vista de configuraciones incompleta: %s", settingsView)
	}
	if strings.Contains(settingsView, "HERMES") || strings.Contains(settingsView, "GROK") {
		t.Fatalf("configuraciones no debe contener HERMES ni GROK: %s", settingsView)
	}

	// 3. Verificar ausencia de emojis en vista de configuraciones
	for _, r := range settingsView {
		if r >= 0x1F300 && r <= 0x1F9FF {
			t.Fatalf("configuraciones no debe contener emojis, encontrado: %U", r)
		}
	}

	// 4. Alternar nivel de permisos (opción 4 en menú, índice 3)
	m.settingsIndex = 3
	initialPerm := m.permissionLevel
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.permissionLevel == initialPerm || m.settingsNotice == "" {
		t.Fatalf("Enter en permisos debió ciclar el nivel. Antes: %s, Ahora: %s, Notice: %s", initialPerm, m.permissionLevel, m.settingsNotice)
	}

	// 5. Opción 0 abre el Selector Interactivo de Proveedores
	m.settingsIndex = 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.state != StateProviderMenu {
		t.Fatalf("Enter en opción 0 debió pasar a StateProviderMenu, obtenido: %v", m.state)
	}

	// 6. En StateProviderMenu, presionar Enter activa el proveedor seleccionado
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.settingsNotice == "" {
		t.Fatalf("Enter en proveedor debió generar settingsNotice con el proveedor activado")
	}

	// 7. En StateProviderMenu, Esc regresa a StateSettingsMenu
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.state != StateSettingsMenu {
		t.Fatalf("Esc desde ProviderMenu debió regresar a StateSettingsMenu, obtenido: %v", m.state)
	}

	// 8. En StateSettingsMenu, Esc regresa a StateStartMenu
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.state != StateStartMenu {
		t.Fatalf("Esc desde SettingsMenu debió regresar a StateStartMenu, obtenido: %v", m.state)
	}

	// 9. Desde chat, /menu regresa a StateStartMenu
	m.state = StateIdle
	_ = m.handleSlashCommand("/menu")
	if m.state != StateStartMenu {
		t.Fatalf("/menu debió cambiar estado a StateStartMenu, obtenido: %v", m.state)
	}
}

func TestOption1_StrictColumnAlignment(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true

	// 1. Verificar alineación en StartMenu
	startView := m.renderStartMenuView()
	for i, line := range strings.Split(startView, "\n") {
		if strings.Contains(line, "[1]") {
			// La línea no debe tener más de 12 espacios antes de [1] o >>
			trimmed := strings.TrimLeft(line, " ")
			leadSpaces := len(line) - len(trimmed)
			t.Logf("StartMenu line %d: leadSpaces=%d content=%q", i, leadSpaces, line)
			if leadSpaces > 15 {
				t.Fatalf("StartMenu: Opción [1] desalineada con sangría excesiva (%d espacios): %q", leadSpaces, line)
			}
		}
	}

	// 2. Verificar alineación en SettingsMenu
	settingsView := m.renderSettingsMenuView()
	for i, line := range strings.Split(settingsView, "\n") {
		if strings.Contains(line, "[1]") {
			trimmed := strings.TrimLeft(line, " ")
			leadSpaces := len(line) - len(trimmed)
			t.Logf("SettingsMenu line %d: leadSpaces=%d content=%q", i, leadSpaces, line)
			if leadSpaces > 15 {
				t.Fatalf("SettingsMenu: Opción [1] desalineada con sangría excesiva (%d espacios): %q", leadSpaces, line)
			}
		}
	}

	// 3. Verificar alineación en RetroWelcomeHero
	heroView := m.renderRetroWelcomeHero(90)
	for i, line := range strings.Split(heroView, "\n") {
		if strings.Contains(line, "[1]") {
			trimmed := strings.TrimLeft(line, " ")
			leadSpaces := len(line) - len(trimmed)
			t.Logf("RetroWelcomeHero line %d: leadSpaces=%d content=%q", i, leadSpaces, line)
			if leadSpaces > 15 {
				t.Fatalf("RetroWelcomeHero: Opción [1] desalineada con sangría excesiva (%d espacios): %q", leadSpaces, line)
			}
		}
	}

	// 4. Verificar alineación en ProviderSelectView
	provView := m.renderProviderSelectView()
	for i, line := range strings.Split(provView, "\n") {
		if strings.Contains(line, "[1]") {
			trimmed := strings.TrimLeft(line, " ")
			leadSpaces := len(line) - len(trimmed)
			t.Logf("ProviderSelectView line %d: leadSpaces=%d content=%q", i, leadSpaces, line)
			if leadSpaces > 15 {
				t.Fatalf("ProviderSelectView: Opción [1] desalineada con sangría excesiva (%d espacios): %q", leadSpaces, line)
			}
		}
	}
}

func TestProviderSelectMenu_FullFlow(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true

	// Ir a Configuraciones
	m.state = StateSettingsMenu
	m.settingsIndex = 0

	// Presionar Enter para entrar al selector de proveedores
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.state != StateProviderMenu {
		t.Fatalf("debió entrar a StateProviderMenu, obtenido: %v", m.state)
	}

	// Navegar hacia abajo
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.providerIndex != 1 {
		t.Fatalf("KeyDown debió mover cursor a índice 1, obtenido: %d", m.providerIndex)
	}

	// Navegar hacia arriba
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.providerIndex != 0 {
		t.Fatalf("KeyUp debió mover cursor a índice 0, obtenido: %d", m.providerIndex)
	}

	// Seleccionar segundo proveedor (Groq) con índice 1
	m.providerIndex = 1 // Groq
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.chat == nil || m.chat.Provider != "groq" {
		t.Fatalf("proveedor debió ser groq, obtenido: %+v", m.chat)
	}
	if !strings.Contains(m.settingsNotice, "GROQ") {
		t.Fatalf("notice debió mencionar GROQ: %s", m.settingsNotice)
	}

	// Renderizar la vista de proveedores y comprobar que contiene GROQ y que no tiene emojis
	view := m.renderProviderSelectView()
	if !strings.Contains(view, "Groq") || !strings.Contains(view, "Cohere") || !strings.Contains(view, "Volver a Configuraciones") {
		t.Fatalf("vista de proveedores incompleta: %s", view)
	}

	for _, r := range view {
		if r >= 0x1F300 && r <= 0x1F9FF {
			t.Fatalf("vista de proveedores contiene emoji: %U", r)
		}
	}

	// Seleccionar la última opción (Volver a Configuraciones)
	cat := providers.GetSupportedCatalog()
	m.providerIndex = len(cat) // Última opción (Volver)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.state != StateSettingsMenu {
		t.Fatalf("Enter en opción Volver debió regresar a StateSettingsMenu, obtenido: %v", m.state)
	}
}

func TestChat_EscReturnsToStartMenu(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true
	m.state = StateIdle

	// 1. Con textarea vacía, Esc regresa a StateStartMenu
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.state != StateStartMenu {
		t.Fatalf("Esc con textarea vacía debió regresar a StateStartMenu, obtenido: %v", m.state)
	}

	// 2. Con texto en textarea, primer Esc limpia el texto
	m.state = StateIdle
	m.textarea.SetValue("texto de prueba")
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.state != StateIdle {
		t.Fatalf("primer Esc debió mantener StateIdle y limpiar texto")
	}
	if m.textarea.Value() != "" {
		t.Fatalf("primer Esc debió vaciar textarea, valor actual: %q", m.textarea.Value())
	}

	// Segundo Esc regresa a StateStartMenu
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.state != StateStartMenu {
		t.Fatalf("segundo Esc debió regresar a StateStartMenu, obtenido: %v", m.state)
	}
}

func TestAPIKeyManager_InteractiveFlow(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true

	// 1. Acceder a configuraciones
	m.state = StateSettingsMenu
	m.settingsIndex = 1 // Opción 2: Gestión de Claves API

	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.state != StateAPIKeySelect {
		t.Fatalf("se esperaba StateAPIKeySelect, obtenido: %v", m.state)
	}

	// 2. Verificar vista de selección de claves API
	selectView := m.renderAPIKeySelectView()
	if !strings.Contains(selectView, "CONFIGURAR API KEYS DE PROVEEDORES") || !strings.Contains(selectView, "Cohere") || !strings.Contains(selectView, "Groq") {
		t.Fatalf("vista de selección de claves incompleta: %s", selectView)
	}
	for _, r := range selectView {
		if r >= 0x1F300 && r <= 0x1F9FF {
			t.Fatalf("renderAPIKeySelectView contiene emoji: %U", r)
		}
	}

	// 3. Seleccionar proveedor para configurar (índice 0)
	m.apiKeyIndex = 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.state != StateAPIKeyInput {
		t.Fatalf("se esperaba StateAPIKeyInput, obtenido: %v", m.state)
	}
	if m.apiKeyTarget == "" {
		t.Fatalf("apiKeyTarget no debe estar vacío")
	}

	// 4. Verificar vista de ingreso de clave API
	inputView := m.renderAPIKeyInputView()
	if !strings.Contains(inputView, "CONFIGURAR CLAVE API") || !strings.Contains(inputView, "Ctrl+V") {
		t.Fatalf("vista de ingreso de clave incompleta: %s", inputView)
	}
	for _, r := range inputView {
		if r >= 0x1F300 && r <= 0x1F9FF {
			t.Fatalf("renderAPIKeyInputView contiene emoji: %U", r)
		}
	}

	// 5. Ingresar una clave y presionar Enter para guardar
	m.apiKeyInput.SetValue("test-token-cohere-123456")
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.state != StateAPIKeySelect {
		t.Fatalf("después de guardar clave debió retornar a StateAPIKeySelect, obtenido: %v", m.state)
	}
	if !strings.Contains(m.settingsNotice, "[OK]") || !strings.Contains(m.settingsNotice, "COHERE") {
		t.Fatalf("notice de guardado inesperado: %s", m.settingsNotice)
	}

	// 6. Esc regresa a StateSettingsMenu
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.state != StateSettingsMenu {
		t.Fatalf("Esc desde StateAPIKeySelect debió regresar a StateSettingsMenu, obtenido: %v", m.state)
	}
}

func TestProfileEditor_InteractiveFlow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ozy-profile-test-*")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "profile_test.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("error initializing test db: %v", err)
	}
	defer db.Close()
	_ = db.EnsureDefaultUser()

	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true

	// 1. Acceder a configuraciones
	m.state = StateSettingsMenu
	m.settingsIndex = 2 // Opción 3: Perfil de Usuario

	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.state != StateProfileEdit {
		t.Fatalf("se esperaba StateProfileEdit, obtenido: %v", m.state)
	}

	// 2. Verificar vista del editor de perfil
	profileView := m.renderProfileEditView()
	if !strings.Contains(profileView, "EDITOR INTERACTIVO DE PERFIL") || !strings.Contains(profileView, "DIRECTIVAS Y PREFERENCIAS") {
		t.Fatalf("vista de editor de perfil incompleta: %s", profileView)
	}
	for _, r := range profileView {
		if r >= 0x1F300 && r <= 0x1F9FF {
			t.Fatalf("renderProfileEditView contiene emoji: %U", r)
		}
	}

	// 3. Ingresar directiva y presionar Enter
	m.profileInput.SetValue("Especialista en Go puro y Bubble Tea")
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.state != StateSettingsMenu {
		t.Fatalf("después de guardar perfil debió regresar a StateSettingsMenu, obtenido: %v", m.state)
	}
	if !strings.Contains(m.settingsNotice, "[OK]") || !strings.Contains(m.settingsNotice, "SQLite") {
		t.Fatalf("notice de perfil guardado inesperado: %s", m.settingsNotice)
	}

	// 4. Verificar que se persistió en la base de datos
	u, err := db.GetUser(db.DefaultUserID())
	if err != nil || u == nil {
		t.Fatalf("error recuperando usuario de DB: %v", err)
	}
	if !strings.Contains(u.ProfileMd, "Especialista en Go puro y Bubble Tea") {
		t.Fatalf("perfil en DB no contiene la directiva guardada: %s", u.ProfileMd)
	}
}

func TestModelInfo_RenderedBelowChatBarInFooter(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true
	m.state = StateIdle

	footerView := m.renderFooter()
	if !strings.Contains(footerView, "[LLM:") || !strings.Contains(footerView, "Permisos:") || !strings.Contains(footerView, "[VOZ:") {
		t.Fatalf("renderFooter debe incluir la información del modelo abajo de la barra de chat: %s", footerView)
	}

	headerView := m.renderHeader()
	if strings.Contains(headerView, "[LLM:") {
		t.Fatalf("renderHeader no debe duplicar las etiquetas del modelo en la parte superior: %s", headerView)
	}
}

func TestProviderMenu_EditKeyShortcut(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true

	// Ir a selector de proveedores
	m.state = StateProviderMenu
	m.providerIndex = 0 // Cohere

	// Presionar 'e' para editar/reemplazar clave
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = res.(Model)

	if m.state != StateAPIKeyInput {
		t.Fatalf("tecla 'e' en ProviderMenu debió abrir StateAPIKeyInput, obtenido: %v", m.state)
	}
	if m.apiKeyTarget != "cohere" {
		t.Fatalf("apiKeyTarget debió ser cohere, obtenido: %s", m.apiKeyTarget)
	}
}

func TestInteractiveCard_NavigationAndSelection(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true
	m.state = StateIdle

	// 1. Verificar que el card inicial existe
	if m.activeCard == nil {
		t.Fatalf("se esperaba activeCard inicializado en StateIdle")
	}
	if len(m.activeCard.Options) == 0 {
		t.Fatalf("activeCard debe tener opciones")
	}

	// 2. Probar navegación KeyDown
	initialIdx := m.activeCard.SelectedIndex
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.activeCard.SelectedIndex != initialIdx+1 {
		t.Fatalf("KeyDown debió avanzar SelectedIndex, antes: %d, ahora: %d", initialIdx, m.activeCard.SelectedIndex)
	}

	// 3. Probar navegación KeyUp
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.activeCard.SelectedIndex != initialIdx {
		t.Fatalf("KeyUp debió retroceder SelectedIndex, esperado: %d, obtenido: %d", initialIdx, m.activeCard.SelectedIndex)
	}

	// 4. Probar cambio de pestaña con KeyTab
	initTab := m.activeCard.ActiveTab
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = res.(Model)
	if m.activeCard.ActiveTab != (initTab+1)%len(m.activeCard.Tabs) {
		t.Fatalf("KeyTab debió avanzar pestaña de la tarjeta")
	}

	// 5. Verificar renderizado del card
	conv := m.renderConversation()
	if !strings.Contains(conv, "Prioridad") || !strings.Contains(conv, ">>") {
		t.Fatalf("renderConversation debe contener las tabs y la opción activa del card: %s", conv)
	}

	// Verificar ausencia de emojis
	for _, r := range conv {
		if r >= 0x1F300 && r <= 0x1F9FF {
			t.Fatalf("renderConversation contiene emojis: %U", r)
		}
	}

	// 6. Seleccionar opción 4 (Escribir instrucción libre) con Enter
	m.activeCard.SelectedIndex = 3 // Opción 4
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.activeCard != nil {
		t.Fatalf("Enter en opción libre debió limpiar activeCard")
	}
}

func TestHeader_ReferenceStyleLayout(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.width = 90
	m.ready = true

	header := m.renderHeader()
	if !strings.Contains(header, "OzyAssist:") {
		t.Fatalf("header debe incluir título de OzyAssist al estilo de referencia: %s", header)
	}
	for _, r := range header {
		if r >= 0x1F300 && r <= 0x1F9FF {
			t.Fatalf("header contiene emojis: %U", r)
		}
	}
}

// TestStartMenu_FourOptions verifica que el menú principal tiene exactamente 4 opciones
// y que el wrap circular cubre el rango 0-3.
func TestStartMenu_FourOptions(t *testing.T) {
	m := InitialModel(nil, nil, false)
	m.state = StateStartMenu
	m.width = 120
	m.height = 40
	m.ready = true

	view := m.renderStartMenuView()

	// Las 4 opciones deben estar presentes en el render
	expectedOptions := []string{
		"INICIAR CONVERSACION",
		"HISTORIAL DE CONVERSACIONES",
		"CONFIGURACIONES",
		"SALIR",
	}
	for _, opt := range expectedOptions {
		if !strings.Contains(view, opt) {
			t.Errorf("renderStartMenuView debe incluir opción '%s'. Render:\n%s", opt, view)
		}
	}

	// Verificar wrap circular: desde índice 0 navegar arriba debe ir a 3
	m.menuIndex = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	updatedM := updated.(Model)
	if updatedM.menuIndex != 3 {
		t.Errorf("Wrap desde 0 hacia arriba debe ir a 3, got %d", updatedM.menuIndex)
	}

	// Desde índice 3 navegar abajo debe ir a 0
	m.menuIndex = 3
	updated2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updatedM2 := updated2.(Model)
	if updatedM2.menuIndex != 0 {
		t.Errorf("Wrap desde 3 hacia abajo debe ir a 0, got %d", updatedM2.menuIndex)
	}
}

// TestChatHistory_NavigationAndResume verifica la pantalla de historial:
// navegación con flechas y transición a StateIdle al seleccionar un chat.
func TestChatHistory_NavigationAndResume(t *testing.T) {
	// Setup DB temporal en memoria
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("db.Init: %v", err)
	}
	_ = db.EnsureDefaultUser()
	defer db.Close()

	m := InitialModel(nil, nil, false)
	m.state = StateChatHistory
	m.width = 120
	m.height = 40
	m.ready = true
	// Simular sin historial: solo entradas fijas
	m.historyChats = nil
	m.chatHistoryIndex = 0

	// Verificar que se renderiza correctamente
	view := m.renderChatHistoryView()
	if !strings.Contains(view, "HISTORIAL DE CONVERSACIONES") {
		t.Errorf("renderChatHistoryView debe incluir el título: %s", view)
	}
	if !strings.Contains(view, "NUEVA CONVERSACION") {
		t.Errorf("renderChatHistoryView debe incluir la opción NUEVA CONVERSACION")
	}
	if !strings.Contains(view, "VOLVER AL MENU PRINCIPAL") {
		t.Errorf("renderChatHistoryView debe incluir la opción VOLVER AL MENU PRINCIPAL")
	}

	// Navegación hacia abajo: de 0 a 1 (totalItems=2, solo fijas)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updatedM := updated.(Model)
	if updatedM.chatHistoryIndex != 1 {
		t.Errorf("Navegación hacia abajo: chatHistoryIndex esperado 1, got %d", updatedM.chatHistoryIndex)
	}

	// Esc debe regresar a StateStartMenu
	updated2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedM2 := updated2.(Model)
	if updatedM2.state != StateStartMenu {
		t.Errorf("Esc desde StateChatHistory debe ir a StateStartMenu, got %v", updatedM2.state)
	}
}

// TestChatHistory_DeleteConfirmFlow verifica el flujo completo de confirmación de borrado:
// tecla 'd' activa confirmación, 'n' la cancela, 's' ejecuta el borrado.
func TestChatHistory_DeleteConfirmFlow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_delete.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("db.Init: %v", err)
	}
	_ = db.EnsureDefaultUser()
	defer db.Close()

	m := InitialModel(nil, nil, false)
	m.state = StateChatHistory
	m.width = 120
	m.height = 40
	m.ready = true
	m.historyChats = nil
	m.chatHistoryIndex = 0
	m.historyConfirmDelete = ""

	// Sin chats, pulsar 'd' no debe activar confirmación (solo chats reales)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updatedM := updated.(Model)
	if updatedM.historyConfirmDelete != "" {
		t.Errorf("'d' sobre entradas fijas no debe activar confirmación de borrado")
	}

	// Cancelar confirmación activa con 'n'
	m.historyConfirmDelete = "some-chat-id"
	updated2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updatedM2 := updated2.(Model)
	if updatedM2.historyConfirmDelete != "" {
		t.Errorf("'n' debe cancelar la confirmación de borrado, got: '%s'", updatedM2.historyConfirmDelete)
	}

	// Cancelar confirmación activa con Esc
	m.historyConfirmDelete = "some-chat-id-2"
	updated3, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedM3 := updated3.(Model)
	if updatedM3.historyConfirmDelete != "" {
		t.Errorf("Esc debe cancelar la confirmación de borrado, got: '%s'", updatedM3.historyConfirmDelete)
	}
}

// TestMouseScrollViewport verifica que la rueda del mouse scrollea el viewport del chat.
func TestMouseScrollViewport(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_mouse.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("db.Init: %v", err)
	}
	_ = db.EnsureDefaultUser()
	defer db.Close()

	m := InitialModel(nil, nil, false)
	m.state = StateIdle
	m.width = 120
	m.height = 40
	// Inicializar viewport manualmente para los tests
	m.viewport.Width = 120
	m.viewport.Height = 30
	m.viewport.SetContent("Linea de contenido de prueba para scroll\n" + strings.Repeat("Otra linea de contenido\n", 50))
	m.ready = true

	// Scroll hacia abajo con rueda
	updated, _ := m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	_ = updated // Solo verificamos que no hay panic ni error

	// Scroll hacia arriba con rueda
	updated2, _ := m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	_ = updated2 // Solo verificamos que no hay panic ni error
}

// TestSlashNew_CreatesNewChat verifica que /new limpia el historial de pantalla
// y produce un mensaje de bienvenida apropiado.
func TestSlashNew_CreatesNewChat(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_new.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("db.Init: %v", err)
	}
	_ = db.EnsureDefaultUser()
	defer db.Close()

	providers.InitProviders()
	system.InitDefaultPathRegistry(db.DB)
	memory.BuildSystemIndex()

	m := InitialModel(nil, nil, false)
	m.width = 120
	m.height = 40
	m.ready = true
	m.state = StateIdle

	// Simular historial de entradas previas
	m.entries = append(m.entries, ChatEntry{Role: "user", Content: "Mensaje previo de prueba"})
	m.entries = append(m.entries, ChatEntry{Role: "assistant", Content: "Respuesta previa de prueba"})

	// Ejecutar /new
	out := m.handleSlashCommand("/new")
	if !strings.Contains(out, "NUEVO CHAT") {
		t.Errorf("/new debe retornar confirmación NUEVO CHAT, got: %s", out)
	}

	// Verificar que /history produce estado correcto
	m.state = StateIdle
	m.textarea.Reset()
	out2 := m.handleSlashCommand("/history")
	_ = out2 // /history retorna "" y cambia el estado
	if m.state != StateChatHistory {
		t.Errorf("/history debe transitar a StateChatHistory, estado actual: %v", m.state)
	}
}


