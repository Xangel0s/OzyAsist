package agent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

// ZeroCallMockProvider falla si el LLM es invocado, demostrando bypass total.
type ZeroCallMockProvider struct {
	mu         sync.Mutex
	callCount  int
	calledWith []providers.CompletionOptions
}

func (m *ZeroCallMockProvider) Name() string        { return "llamacpp" }
func (m *ZeroCallMockProvider) SupportsTools() bool { return true }
func (m *ZeroCallMockProvider) Models() []string    { return []string{"qwen2.5-3b-gguf"} }

func (m *ZeroCallMockProvider) StreamCompletion(ctx context.Context, messages []providers.Message, opts providers.CompletionOptions) (<-chan providers.StreamChunk, error) {
	m.mu.Lock()
	m.callCount++
	m.calledWith = append(m.calledWith, opts)
	m.mu.Unlock()

	ch := make(chan providers.StreamChunk, 2)
	ch <- providers.StreamChunk{Type: "text", Content: "LLM fue llamado"}
	ch <- providers.StreamChunk{Type: "done"}
	close(ch)
	return ch, nil
}

func TestRunAgentLoop_FastTrackDirectBypass(t *testing.T) {
	setupTestDB(t)
	mockProv := &ZeroCallMockProvider{}

	completedCh := make(chan bool, 1)
	var events []AgentEvent
	var evMu sync.Mutex
	emit := func(e AgentEvent) {
		evMu.Lock()
		events = append(events, e)
		evMu.Unlock()
		if e.Type == "agent:completed" {
			select {
			case completedCh <- true:
			default:
			}
		}
	}

	params := AgentLoopParams{
		Provider:        mockProv,
		Chat:            &models.Chat{ID: "test-chat-ft", Model: "qwen2.5-3b-gguf"},
		UserMessage:     "apaga la bulla",
		PermissionLevel: string(Autonomous),
		Emit:            emit,
		VoiceMode:       false,
	}

	sessionID := StartAgentLoop(context.Background(), params)
	if sessionID == "" {
		t.Fatal("StartAgentLoop retornó sessionID vacío")
	}

	select {
	case <-completedCh:
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout esperando agent:completed en Fast-Track")
	}

	mockProv.mu.Lock()
	calls := mockProv.callCount
	mockProv.mu.Unlock()

	// 1. Verificar que el LLM NUNCA fue llamado (0 tokens, 0 llamadas)
	if calls != 0 {
		t.Fatalf("FALLA FAST-TRACK: El LLM fue llamado %d veces cuando debía haber sido bypass directo (0 llamadas)!", calls)
	}

	// 2. Verificar eventos emitidos
	evMu.Lock()
	defer evMu.Unlock()

	var hasToolCall, hasToolResult, hasCompleted bool
	for _, e := range events {
		if e.Type == "tool:call" && e.ToolName == "os_audio_device" {
			hasToolCall = true
		}
		if e.Type == "tool:result" && e.ToolName == "os_audio_device" {
			hasToolResult = true
		}
		if e.Type == "agent:completed" {
			hasCompleted = true
		}
	}

	if !hasToolCall {
		t.Errorf("Esperaba evento tool:call para os_audio_device en Fast-Track")
	}
	if !hasToolResult {
		t.Errorf("Esperaba evento tool:result para os_audio_device en Fast-Track")
	}
	if !hasCompleted {
		t.Errorf("Esperaba evento agent:completed tras ejecución Fast-Track")
	}
}

func TestRunAgentLoop_RollbackExecution(t *testing.T) {
	setupTestDB(t)
	mockProv := &ZeroCallMockProvider{}

	completed1 := make(chan bool, 1)
	var events []AgentEvent
	var evMu sync.Mutex
	emit1 := func(e AgentEvent) {
		evMu.Lock()
		events = append(events, e)
		evMu.Unlock()
		if e.Type == "agent:completed" {
			select {
			case completed1 <- true:
			default:
			}
		}
	}

	// Primero disparar un Fast-Track para registrar el rollback
	params1 := AgentLoopParams{
		Provider:        mockProv,
		Chat:            &models.Chat{ID: "test-chat-rb", Model: "qwen2.5-3b-gguf"},
		UserMessage:     "silencia la pc",
		PermissionLevel: string(Autonomous),
		Emit:            emit1,
		VoiceMode:       false,
	}
	StartAgentLoop(context.Background(), params1)
	select {
	case <-completed1:
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout esperando completed1")
	}

	// Luego enviar la orden de Rollback ("deshazlo")
	completed2 := make(chan bool, 1)
	evMu.Lock()
	events = nil
	evMu.Unlock()

	emit2 := func(e AgentEvent) {
		evMu.Lock()
		events = append(events, e)
		evMu.Unlock()
		if e.Type == "agent:completed" {
			select {
			case completed2 <- true:
			default:
			}
		}
	}

	params2 := AgentLoopParams{
		Provider:        mockProv,
		Chat:            &models.Chat{ID: "test-chat-rb", Model: "qwen2.5-3b-gguf"},
		UserMessage:     "deshazlo por favor",
		PermissionLevel: string(Autonomous),
		Emit:            emit2,
		VoiceMode:       false,
	}
	StartAgentLoop(context.Background(), params2)
	select {
	case <-completed2:
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout esperando completed2")
	}

	mockProv.mu.Lock()
	calls := mockProv.callCount
	mockProv.mu.Unlock()

	if calls != 0 {
		t.Fatalf("FALLA ROLLBACK: El LLM fue llamado %d veces cuando debía haber sido reversión directa (0 llamadas)!", calls)
	}

	evMu.Lock()
	defer evMu.Unlock()

	var hasRollbackTool, hasCompleted bool
	for _, e := range events {
		if e.Type == "tool:call" && strings.HasPrefix(e.ToolID, "rollback_") {
			hasRollbackTool = true
		}
		if e.Type == "agent:completed" {
			hasCompleted = true
		}
	}

	if !hasRollbackTool {
		t.Errorf("Esperaba llamada a herramienta de rollback con prefijo 'rollback_'")
	}
	if !hasCompleted {
		t.Errorf("Esperaba que el rollback emitiera agent:completed")
	}
}

func TestDynamicToolPruning_LocalModel(t *testing.T) {
	// Verificar que para un modelo local y una consulta específica, las herramientas se podan de 48 a <= 3
	allTools := GetActiveTools(false)
	if len(allTools) < 20 {
		t.Fatalf("Esperaba catálogo completo de herramientas > 20, obtuvo: %d", len(allTools))
	}

	// Consulta sobre música/volumen para modelo local
	pruned := GetActiveToolsForQuery("bájale a la bulla", false, true)
	if len(pruned) > 5 {
		t.Errorf("FALLA PODA DINÁMICA: Esperaba <= 5 herramientas podadas, obtuvo %d herramientas: %v", len(pruned), toolNames(pruned))
	}

	hasAudioDevice := false
	for _, pt := range pruned {
		if pt.Name == "os_audio_device" {
			hasAudioDevice = true
			break
		}
	}
	if !hasAudioDevice {
		t.Errorf("Poda dinámica debía incluir os_audio_device para 'bájale a la bulla'")
	}
}

func toolNames(tools []providers.ToolDef) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}
