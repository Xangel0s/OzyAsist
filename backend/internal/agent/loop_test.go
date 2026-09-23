package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

// MockProvider simula respuestas secuenciales del LLM para probar el ReAct Loop.
type MockProvider struct {
	mu        sync.Mutex
	turnCount int
	turns     [][]providers.StreamChunk
}

func (m *MockProvider) Name() string       { return "mock" }
func (m *MockProvider) SupportsTools() bool { return true }
func (m *MockProvider) Models() []string    { return []string{"mock-model"} }

func (m *MockProvider) StreamCompletion(ctx context.Context, messages []providers.Message, opts providers.CompletionOptions) (<-chan providers.StreamChunk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan providers.StreamChunk, 10)

	var chunks []providers.StreamChunk
	if m.turnCount < len(m.turns) {
		chunks = m.turns[m.turnCount]
		m.turnCount++
	} else {
		// Default: respuesta final
		chunks = []providers.StreamChunk{
			{Type: "text", Content: "Tarea finalizada con éxito."},
			{Type: "done"},
		}
	}

	go func() {
		defer close(ch)
		for _, c := range chunks {
			select {
			case <-ctx.Done():
				ch <- providers.StreamChunk{Type: "error", Content: "cancelado"}
				return
			case ch <- c:
			}
		}
	}()

	return ch, nil
}

func setupTestDB(t *testing.T) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "ozy-db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(tempDir)
	})

	dbPath := filepath.Join(tempDir, "test.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("setup test db: %v", err)
	}
	if err := db.EnsureDefaultUser(); err != nil {
		t.Fatalf("ensure default user: %v", err)
	}
}

func TestReActLoop_AutonomousMultiTurn(t *testing.T) {
	setupTestDB(t)

	// Crear sandbox temporal para el proyecto
	sandboxDir, err := os.MkdirTemp("", "ozy-project-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandboxDir)

	// Crear un archivo inicial
	initialFile := filepath.Join(sandboxDir, "hello.txt")
	if err := os.WriteFile(initialFile, []byte("Hola Mundo Original"), 0644); err != nil {
		t.Fatal(err)
	}

	// Mock con 3 turnos:
	// Turno 0: LLM pide read_file
	// Turno 1: LLM pide write_file
	// Turno 2: LLM emite texto final sin herramientas
	mock := &MockProvider{
		turns: [][]providers.StreamChunk{
			{
				{
					Type: "tool_call",
					ToolCall: &providers.ToolCall{
						ID:    "call_1",
						Name:  "read_file",
						Input: json.RawMessage(`{"path":"hello.txt"}`),
					},
				},
				{Type: "done"},
			},
			{
				{
					Type: "tool_call",
					ToolCall: &providers.ToolCall{
						ID:    "call_2",
						Name:  "write_file",
						Input: json.RawMessage(`{"path":"result.txt","content":"Contenido Generado por el Agente"}`),
					},
				},
				{Type: "done"},
			},
			{
				{Type: "text", Content: "He leído hello.txt y creado result.txt correctamente."},
				{Type: "done"},
			},
		},
	}

	userID := db.DefaultUserID()
	projectID := uuid.NewString()
	chatID := uuid.NewString()

	proj := &models.Project{
		ID:              projectID,
		UserID:          userID,
		Name:            "Test Project",
		RootPath:        sandboxDir,
		PermissionLevel: "sandboxed",
		AgentConsent:    "always",
		CreatedAt:       time.Now(),
	}
	if err := db.CreateProject(proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	chat := &models.Chat{
		ID:        chatID,
		UserID:    userID,
		ProjectID: projectID,
		Name:      "Code Chat",
		Mode:      "code",
		Provider:  "mock",
		Model:     "mock-model",
		CreatedAt: time.Now(),
	}
	if err := db.CreateChat(chat); err != nil {
		t.Fatalf("create chat: %v", err)
	}

	project, _ := db.GetProject(projectID)

	var eventsMu sync.Mutex
	var events []AgentEvent
	completedCh := make(chan bool, 1)

	params := AgentLoopParams{
		Provider:        mock,
		Chat:            chat,
		Project:         project,
		UserMessage:     "Lee hello.txt y crea result.txt",
		PermissionLevel: "sandboxed",
		Emit: func(ev AgentEvent) {
			eventsMu.Lock()
			events = append(events, ev)
			eventsMu.Unlock()

			if ev.Type == "agent:completed" {
				select {
				case completedCh <- true:
				default:
				}
			}
		},
	}

	sessionID := StartAgentLoop(context.Background(), params)
	if sessionID == "" {
		t.Fatal("expected non-empty sessionID")
	}

	select {
	case <-completedCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout esperando finalización del loop agéntico")
	}

	// Verificar que result.txt fue creado en el sandbox
	createdFile := filepath.Join(sandboxDir, "result.txt")
	content, err := os.ReadFile(createdFile)
	if err != nil {
		t.Fatalf("no se creó result.txt: %v", err)
	}
	if string(content) != "Contenido Generado por el Agente" {
		t.Fatalf("contenido inesperado: %s", string(content))
	}

	// Verificar secuencia de eventos
	eventsMu.Lock()
	defer eventsMu.Unlock()

	hasToolCallRead := false
	hasToolResultRead := false
	hasToolCallWrite := false
	hasToolResultWrite := false
	hasCompleted := false

	for _, ev := range events {
		if ev.Type == "tool:call" && ev.ToolName == "read_file" {
			hasToolCallRead = true
		}
		if ev.Type == "tool:result" && ev.ToolID == "call_1" && ev.ToolSuccess {
			hasToolResultRead = true
			if !strings.Contains(ev.ToolOutput, "Hola Mundo Original") {
				t.Fatalf("tool result read no contiene contenido esperado: %s", ev.ToolOutput)
			}
		}
		if ev.Type == "tool:call" && ev.ToolName == "write_file" {
			hasToolCallWrite = true
		}
		if ev.Type == "tool:result" && ev.ToolID == "call_2" && ev.ToolSuccess {
			hasToolResultWrite = true
		}
		if ev.Type == "agent:completed" {
			hasCompleted = true
		}
	}

	if !hasToolCallRead || !hasToolResultRead {
		t.Fatal("falló la ejecución/evento de read_file")
	}
	if !hasToolCallWrite || !hasToolResultWrite {
		t.Fatal("falló la ejecución/evento de write_file")
	}
	if !hasCompleted {
		t.Fatal("no se emitió evento agent:completed")
	}
}

func TestReActLoop_ApprovalFlow(t *testing.T) {
	setupTestDB(t)

	sandboxDir, err := os.MkdirTemp("", "ozy-approval-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sandboxDir)

	mock := &MockProvider{
		turns: [][]providers.StreamChunk{
			{
				{
					Type: "tool_call",
					ToolCall: &providers.ToolCall{
						ID:    "call_cmd_1",
						Name:  "run_command",
						Input: json.RawMessage(`{"command":"rm -rf tmp_test_dir"}`),
					},
				},
				{Type: "done"},
			},
			{
				{Type: "text", Content: "Comando destructivo aprobado y procesado."},
				{Type: "done"},
			},
		},
	}

	userID := db.DefaultUserID()
	projectID := uuid.NewString()
	chatID := uuid.NewString()

	proj := &models.Project{
		ID:              projectID,
		UserID:          userID,
		Name:            "Approval Project",
		RootPath:        sandboxDir,
		PermissionLevel: "sandboxed",
		AgentConsent:    "ask",
		CreatedAt:       time.Now(),
	}
	if err := db.CreateProject(proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	chat := &models.Chat{
		ID:        chatID,
		UserID:    userID,
		ProjectID: projectID,
		Name:      "Approval Chat",
		Mode:      "code",
		Provider:  "mock",
		Model:     "mock-model",
		CreatedAt: time.Now(),
	}
	if err := db.CreateChat(chat); err != nil {
		t.Fatalf("create chat: %v", err)
	}

	project, _ := db.GetProject(projectID)

	var sessionID string
	approvalReqCh := make(chan string, 1)
	completedCh := make(chan bool, 1)

	params := AgentLoopParams{
		Provider:        mock,
		Chat:            chat,
		Project:         project,
		UserMessage:     "Ejecuta un comando destructivo",
		PermissionLevel: "sandboxed",
		Emit: func(ev AgentEvent) {
			if ev.Type == "tool:approval_request" {
				select {
				case approvalReqCh <- ev.ToolID:
				default:
				}
			}
			if ev.Type == "agent:completed" {
				select {
				case completedCh <- true:
				default:
				}
			}
		},
	}

	sessionID = StartAgentLoop(context.Background(), params)

	// Esperar que el loop solicite aprobación
	select {
	case toolID := <-approvalReqCh:
		if toolID != "call_cmd_1" {
			t.Fatalf("toolID inesperado para aprobación: %s", toolID)
		}
		// Simular que el usuario aprueba la ejecución
		RespondApproval(sessionID, toolID, true)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout esperando tool:approval_request")
	}

	// Esperar completado
	select {
	case <-completedCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout esperando completion tras aprobación")
	}
}

func TestReActLoop_ContextCancellation(t *testing.T) {
	setupTestDB(t)

	mock := &MockProvider{
		turns: [][]providers.StreamChunk{
			{
				{
					Type: "tool_call",
					ToolCall: &providers.ToolCall{
						ID:    "call_long",
						Name:  "run_command",
						Input: json.RawMessage(`{"command":"Start-Sleep -Seconds 10"}`),
					},
				},
				{Type: "done"},
			},
		},
	}

	userID := db.DefaultUserID()
	chatID := uuid.NewString()
	chat := &models.Chat{
		ID:        chatID,
		UserID:    userID,
		Name:      "Cancel Chat",
		Mode:      "code",
		Provider:  "mock",
		CreatedAt: time.Now(),
	}
	db.CreateChat(chat)

	canceledCh := make(chan bool, 1)

	ctx, cancel := context.WithCancel(context.Background())

	params := AgentLoopParams{
		Provider:        mock,
		Chat:            chat,
		UserMessage:     "ejecuta algo largo",
		PermissionLevel: "sandboxed",
		Emit: func(ev AgentEvent) {
			if ev.Type == "error" && strings.Contains(ev.Error, "cancelado") {
				select {
				case canceledCh <- true:
				default:
				}
			}
		},
	}

	sessionID := StartAgentLoop(ctx, params)

	// Cancelar casi de inmediato
	time.Sleep(100 * time.Millisecond)
	CancelSession(sessionID)
	cancel()

	select {
	case <-canceledCh:
		// OK - loop abortó limpiamente
	case <-time.After(3 * time.Second):
		t.Fatal("timeout esperando cancelación del loop")
	}
}

func TestReActLoop_ToolsExecution(t *testing.T) {
	dir, err := os.MkdirTemp("", "ozy-tools-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	sandbox, err := NewSandbox(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// 1. write_file
	writeTC := providers.ToolCall{
		ID:    "t1",
		Name:  "write_file",
		Input: json.RawMessage(`{"path":"src/main.go","content":"package main\n\nfunc main() {}\n"}`),
	}
	out, ok := execWriteFile(ctx, writeTC, sandbox)
	if !ok || !strings.Contains(out, "Archivo escrito") {
		t.Fatalf("write_file falló: %s", out)
	}

	// 2. read_file
	readTC := providers.ToolCall{
		ID:    "t2",
		Name:  "read_file",
		Input: json.RawMessage(`{"path":"src/main.go"}`),
	}
	readOut, ok := execReadFile(ctx, readTC, sandbox)
	if !ok || !strings.Contains(readOut, "func main()") {
		t.Fatalf("read_file falló: %s", readOut)
	}

	// 3. list_files
	listTC := providers.ToolCall{
		ID:    "t3",
		Name:  "list_files",
		Input: json.RawMessage(`{"pattern":"**/*.go"}`),
	}
	listOut, ok := execListFiles(ctx, listTC, sandbox)
	if !ok || !strings.Contains(listOut, "main.go") {
		t.Fatalf("list_files falló: %s", listOut)
	}

	// 4. search_text
	searchTC := providers.ToolCall{
		ID:    "t4",
		Name:  "search_text",
		Input: json.RawMessage(`{"query":"func main"}`),
	}
	searchOut, ok := execSearchText(ctx, searchTC, sandbox)
	if !ok || !strings.Contains(searchOut, "main.go") {
		t.Fatalf("search_text falló: %s", searchOut)
	}

	// 5. apply_diff
	diffTC := providers.ToolCall{
		ID:    "t5",
		Name:  "apply_diff",
		Input: json.RawMessage(`{"path":"src/main.go","diff":"--- a/src/main.go\n+++ b/src/main.go\n@@ -1,3 +1,4 @@\n package main\n \n+// patched\n func main() {}\n"}`),
	}
	diffOut, ok := execApplyDiff(ctx, diffTC, sandbox)
	if !ok || !strings.Contains(diffOut, "Diff aplicado") {
		t.Fatalf("apply_diff falló: %s", diffOut)
	}

	// Verificar contenido parchado
	patchedData, err := os.ReadFile(filepath.Join(dir, "src", "main.go"))
	if err != nil || !strings.Contains(string(patchedData), "// patched") {
		t.Fatalf("archivo no fue parchado correctamente: %s", string(patchedData))
	}

	// 6. Path traversal rejection
	traversalTC := providers.ToolCall{
		ID:    "t6",
		Name:  "read_file",
		Input: json.RawMessage(`{"path":"../../etc/passwd"}`),
	}
	travOut, travOk := execReadFile(ctx, traversalTC, sandbox)
	if travOk || !strings.Contains(travOut, "path traversal") {
		t.Fatalf("se esperaba rechazo por path traversal, se obtuvo: %v, %s", travOk, travOut)
	}
}

func TestSanitizeHistoryRoles(t *testing.T) {
	// Caso 1: Asistente huérfano al inicio (debe ser descartado)
	msgs := []providers.Message{
		{Role: "system", Content: "sys"},
		{Role: "assistant", Content: "asistente huerfano"},
		{Role: "user", Content: "hola"},
	}
	res := sanitizeHistoryRoles(msgs)
	if len(res) != 2 || res[1].Role != "user" || res[1].Content != "hola" {
		t.Fatalf("Esperaba que se descartara el asistente huerfano, obtuve: %+v", res)
	}

	// Caso 2: Mensajes consecutivos del mismo rol (deben fusionarse)
	msgs2 := []providers.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "primero"},
		{Role: "user", Content: "segundo"},
	}
	res2 := sanitizeHistoryRoles(msgs2)
	if len(res2) != 2 || !strings.Contains(res2[1].Content, "primero") || !strings.Contains(res2[1].Content, "segundo") {
		t.Fatalf("Esperaba fusion de mensajes de usuario consecutivos, obtuve: %+v", res2)
	}
}

func TestExtractToolCallsFromText_KnownToolsOnly(t *testing.T) {
	// Tool válida registrada
	validText := `Voy a revisar los archivos: <tool_call>{"name": "list_files", "arguments": {"pattern": "*.go"}}</tool_call>`
	calls := extractToolCallsFromText(validText)
	if len(calls) != 1 || calls[0].Name != "list_files" {
		t.Fatalf("Esperaba 1 llamada a list_files, obtuve: %+v", calls)
	}

	// Tool falsa o no registrada (debe ser ignorada para evitar errores o bucles)
	fakeText := `Aquí hay un ejemplo JSON: <tool_call>{"name": "invented_tool_xyz", "arguments": {"x": 1}}</tool_call>`
	fakeCalls := extractToolCallsFromText(fakeText)
	if len(fakeCalls) != 0 {
		t.Fatalf("Esperaba 0 llamadas para herramienta inexistente, obtuve: %+v", fakeCalls)
	}

	// Tool en formato JSON directo (como emite el modelo fine-tuneado OzyAssist-7B-v3)
	rawJsonText := `{"name": "os_close_window", "arguments": {"title": "Calculadora"}}`
	rawCalls := extractToolCallsFromText(rawJsonText)
	if len(rawCalls) != 1 || rawCalls[0].Name != "os_close_window" {
		t.Fatalf("Esperaba 1 llamada a os_close_window desde raw JSON, obtuve: %+v", rawCalls)
	}
}

func TestExtractToolCallsFromText_CodeFunctionStyle(t *testing.T) {
	codeText := `\nos_launch_app("notepad", "C:\Users\User")`
	calls := extractToolCallsFromText(codeText)
	if len(calls) != 1 || calls[0].Name != "os_launch_app" {
		t.Fatalf("Esperaba 1 llamada a os_launch_app, obtuve: %+v", calls)
	}
	var params struct {
		Target string `json:"target"`
		Path   string `json:"path"`
	}
	_ = json.Unmarshal(calls[0].Input, &params)
	if params.Target != "notepad" || params.Path != "C:\\Users\\User" {
		t.Fatalf("Parámetros incorrectos: %+v", params)
	}

	closeText := `ANTI-ALUCINACIÓN:\n\nos_close_window() administrador de tareas`
	closeCalls := extractToolCallsFromText(closeText)
	if len(closeCalls) != 1 || closeCalls[0].Name != "os_close_window" {
		t.Fatalf("Esperaba 1 llamada a os_close_window, obtuve: %+v", closeCalls)
	}

	// Test caso sensible reportado por el usuario: Os_launch_app("calc") con mayúscula
	mixedCaseText := `Os_launch_app("calc")`
	mixedCalls := extractToolCallsFromText(mixedCaseText)
	if len(mixedCalls) != 1 || mixedCalls[0].Name != "os_launch_app" {
		t.Fatalf("Esperaba que Os_launch_app se resolviera a os_launch_app, obtuve: %+v", mixedCalls)
	}
	var mixedParams struct {
		Target string `json:"target"`
	}
	_ = json.Unmarshal(mixedCalls[0].Input, &mixedParams)
	if mixedParams.Target != "calc" {
		t.Fatalf("Esperaba target 'calc', obtuve '%s'", mixedParams.Target)
	}

	// Test fallback heurístico para modelo local conversacional: "Ozy ha abierto calc."
	convText := `Ozy ha abierto calc.`
	convCalls := extractToolCallsFromText(convText)
	if len(convCalls) != 1 || convCalls[0].Name != "os_launch_app" {
		t.Fatalf("Esperaba que el fallback heurístico capturara os_launch_app, obtuve: %+v", convCalls)
	}
	var convParams struct {
		Target string `json:"target"`
	}
	_ = json.Unmarshal(convCalls[0].Input, &convParams)
	if convParams.Target != "calc" {
		t.Fatalf("Esperaba target 'calc', obtuve '%s'", convParams.Target)
	}

	// Test caso exacto del usuario: "Invoqué a os_launch_app con calc"
	userCaseText := `Invoqué a os_launch_app con calc`
	userCalls := extractToolCallsFromText(userCaseText)
	if len(userCalls) != 1 || userCalls[0].Name != "os_launch_app" {
		t.Fatalf("Esperaba que 'Invoqué a os_launch_app con calc' generara 1 llamada a os_launch_app, obtuve: %+v", userCalls)
	}
	var userParams struct {
		Target string `json:"target"`
	}
	_ = json.Unmarshal(userCalls[0].Input, &userParams)
	if userParams.Target != "calc" {
		t.Fatalf("Esperaba target 'calc', obtuve '%s'", userParams.Target)
	}

	// Test caso con artículo: "Abriendo la calculadora"
	articleText := `Abriendo la calculadora`
	artCalls := extractToolCallsFromText(articleText)
	if len(artCalls) != 1 || artCalls[0].Name != "os_launch_app" {
		t.Fatalf("Esperaba que 'Abriendo la calculadora' generara 1 llamada a os_launch_app, obtuve: %+v", artCalls)
	}
	var artParams struct {
		Target string `json:"target"`
	}
	_ = json.Unmarshal(artCalls[0].Input, &artParams)
	if artParams.Target != "calculadora" {
		t.Fatalf("Esperaba target 'calculadora', obtuve '%s'", artParams.Target)
	}
}

func TestCheckPendingTaskRequirements_ConversationalExclusion(t *testing.T) {
	// Pregunta conversacional sobre correo (no debe exigir os_draft_email)
	question := "¿Cómo configuro mi correo en OzyAssist?"
	prompt := checkPendingTaskRequirements(question, nil)
	if prompt != "" {
		t.Fatalf("Pregunta conversacional no debió exigir os_draft_email, obtuve: %s", prompt)
	}

	// Orden explícita con verbo de acción (debe exigir os_draft_email)
	action := "Redacta un correo para cliente@empresa.com con el informe"
	actionPrompt := checkPendingTaskRequirements(action, nil)
	if actionPrompt == "" || !strings.Contains(actionPrompt, "os_draft_email") {
		t.Fatalf("Orden de acción debió exigir os_draft_email, obtuve: %s", actionPrompt)
	}
}

func TestBuildAgentSystemPrompt_IncludesTriad(t *testing.T) {
	prompt := BuildSystemPromptForTest(AgentLoopParams{})
	if !strings.Contains(prompt, "CHARC") || !strings.Contains(prompt, "NINE") {
		t.Fatalf("System prompt debe incluir a los subagentes CHARC y NINE, obtuve: %s", prompt)
	}
}

func TestRescueDirectUserIntent(t *testing.T) {
	// Caso multi-app: "abre la calculadora y el bloc de notas"
	multiCalls := rescueDirectUserIntent("abre la calculadora y el bloc de notas", "Ya tengo abierto calculadora y el bloc de notas.")
	if len(multiCalls) != 2 {
		t.Fatalf("Esperaba 2 tool calls para multi-app, obtuve %d: %+v", len(multiCalls), multiCalls)
	}
	if multiCalls[0].Name != "os_launch_app" || multiCalls[1].Name != "os_launch_app" {
		t.Fatalf("Herramientas incorrectas: %+v", multiCalls)
	}

	// Caso cierre: "cierra la calculadora"
	closeCalls := rescueDirectUserIntent("cierra la calculadora", "...")
	if len(closeCalls) != 1 || closeCalls[0].Name != "os_close_window" {
		t.Fatalf("Esperaba 1 llamada a os_close_window, obtuve: %+v", closeCalls)
	}

	// Caso app arbitraria: "abre roblox"
	robloxCalls := rescueDirectUserIntent("abre roblox", "Ya tengo abierto Roblox.")
	if len(robloxCalls) != 1 || robloxCalls[0].Name != "os_launch_app" {
		t.Fatalf("Esperaba llamada a os_launch_app para roblox, obtuve: %+v", robloxCalls)
	}
	var robloxParams struct{ Target string `json:"target"` }
	_ = json.Unmarshal(robloxCalls[0].Input, &robloxParams)
	if robloxParams.Target != "roblox" {
		t.Fatalf("Esperaba target 'roblox', obtuve: %s", robloxParams.Target)
	}

	// Caso conversacional normal: "¿Cómo te llamas?" (no debe generar rescate)
	convCalls := rescueDirectUserIntent("¿Cómo te llamas?", "Soy OzyAssist.")
	if len(convCalls) != 0 {
		t.Fatalf("No debió generar llamadas para conversación normal, obtuve: %+v", convCalls)
	}

	// Caso USB: "que puertos USB fisicos estan en uso actualmente?"
	usbCalls := rescueDirectUserIntent("que puertos USB fisicos estan en uso actualmente?", "...")
	if len(usbCalls) != 1 || usbCalls[0].Name != "os_hardware_inspector" {
		t.Fatalf("Esperaba os_hardware_inspector para USB, obtuve: %+v", usbCalls)
	}

	// Caso WiFi: "has una auditoria de red wifi actual"
	wifiCalls := rescueDirectUserIntent("has una auditoria de red wifi actual", "...")
	if len(wifiCalls) != 1 || wifiCalls[0].Name != "os_wifi_manager" {
		t.Fatalf("Esperaba os_wifi_manager para WiFi, obtuve: %+v", wifiCalls)
	}

	// Caso Búsqueda + PDF: "crea un pdf sobre roblox y su informacion de la web"
	reportCalls := rescueDirectUserIntent("crea un pdf sobre roblox y su informacion de la web", "...")
	if len(reportCalls) != 1 || reportCalls[0].Name != "web_search" {
		t.Fatalf("Esperaba web_search para 'crea un pdf sobre roblox y su informacion de la web', obtuve: %+v", reportCalls)
	}
}

func TestExtractToolCallsFromText_UserScreenshotCases(t *testing.T) {
	// Caso 1: "Usando web_search con {"query": "auditoría de red wifi windows 10"}"
	text1 := `Usando web_search con {"query": "auditoría de red wifi windows 10"}`
	calls1 := extractToolCallsFromText(text1)
	if len(calls1) != 1 || calls1[0].Name != "web_search" {
		t.Fatalf("Caso 1 falló: %+v", calls1)
	}

	// Caso 2: "os_create_pdf con {"path": "C:\\Users\\User\\Documents\\roblox_info.pdf", "title": "Información Roblox", "sections": [{"title": "Sec1", "content": "Detalles"}]}"
	text2 := `os_create_pdf con {"path": "C:\\Users\\User\\Documents\\roblox_info.pdf", "title": "Información Roblox", "sections": [{"title": "Sec1", "content": "Detalles"}]}`
	calls2 := extractToolCallsFromText(text2)
	if len(calls2) != 1 || calls2[0].Name != "os_create_pdf" {
		t.Fatalf("Caso 2 falló: %+v", calls2)
	}
	var p2 struct {
		Path     string `json:"path"`
		Title    string `json:"title"`
		Sections []any  `json:"sections"`
	}
	if err := json.Unmarshal(calls2[0].Input, &p2); err != nil || len(p2.Sections) != 1 {
		t.Fatalf("Caso 2 JSON anidado falló: %+v (err: %v)", p2, err)
	}

	// Caso 3: "Invocando os_usb_devices para obtener la lista de dispositivos conectados y su estado de uso."
	text3 := `Invocando os_usb_devices para obtener la lista de dispositivos conectados y su estado de uso.`
	calls3 := extractToolCallsFromText(text3)
	if len(calls3) != 1 || calls3[0].Name != "os_hardware_inspector" {
		t.Fatalf("Caso 3 falló: %+v", calls3)
	}

	// Caso 4: "Os_creating_pdf: Cages The Elephant - Albums"
	text4 := `Os_creating_pdf: Cages The Elephant - Albums`
	calls4 := extractToolCallsFromText(text4)
	if len(calls4) != 1 || calls4[0].Name != "os_create_pdf" {
		t.Fatalf("Caso 4 falló: %+v", calls4)
	}

	// Caso 5: Raw JSON sin tags "<tool_call>"
	text5 := `<thought>Consultando estado de la red WiFi.</thought>
{"name": "os_wifi_manager", "arguments": {"action": "status"}}`
	calls5 := extractToolCallsFromText(text5)
	if len(calls5) != 1 || calls5[0].Name != "os_wifi_manager" {
		t.Fatalf("Caso 5 falló: %+v", calls5)
	}
}

func TestRescueDirectUserIntent_FileSearchCases(t *testing.T) {
	// Caso 1: "busca en el disco duro el informe de cage the elephant que se creo correcto?"
	msg1 := "busca en el disco duro el informe de cage the elephant que se creo correcto?"
	calls1 := rescueDirectUserIntent(msg1, "")
	if len(calls1) != 1 || calls1[0].Name != "os_find_files" {
		t.Fatalf("Esperaba os_find_files para msg1, obtuve: %+v", calls1)
	}

	// Caso 2: "claro comentame cual es la direccion para abrir el archivo la ubicacion"
	msg2 := "claro comentame cual es la direccion para abrir el archivo la ubicacion"
	calls2 := rescueDirectUserIntent(msg2, "")
	if len(calls2) != 1 || calls2[0].Name != "os_find_files" {
		t.Fatalf("Esperaba os_find_files para msg2, obtuve: %+v", calls2)
	}

	// Caso 3 (Screenshot): "cual es la ruta y ubicacion del informe sobre roblox"
	msg3 := "cual es la ruta y ubicacion del informe sobre roblox"
	calls3 := rescueDirectUserIntent(msg3, "")
	if len(calls3) != 1 || calls3[0].Name != "os_find_files" {
		t.Fatalf("Esperaba os_find_files para msg3, obtuve: %+v", calls3)
	}
	if !strings.Contains(string(calls3[0].Input), "roblox") {
		t.Fatalf("Esperaba que el patrón de búsqueda contenga 'roblox', obtuve: %s", string(calls3[0].Input))
	}

	// Caso 4 (Screenshot con typo): "caul es la ruta del informe sobre roblox"
	msg4 := "caul es la ruta del informe sobre roblox"
	calls4 := rescueDirectUserIntent(msg4, "")
	if len(calls4) != 1 || calls4[0].Name != "os_find_files" {
		t.Fatalf("Esperaba os_find_files para msg4, obtuve: %+v", calls4)
	}
	if !strings.Contains(string(calls4[0].Input), "roblox") {
		t.Fatalf("Esperaba que el patrón de búsqueda contenga 'roblox', obtuve: %s", string(calls4[0].Input))
	}
}


