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
