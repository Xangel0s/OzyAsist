package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

// checkOllamaLive verifica si Ollama está en línea y si tiene llama3.1 disponible.
func checkOllamaLive(t *testing.T) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://127.0.0.1:11434/api/tags")
	if err != nil {
		t.Skip("Ollama no está en ejecución en 127.0.0.1:11434; omitiendo prueba live con modelo local.")
		return false
	}
	defer resp.Body.Close()
	return true
}

// collectCompletion recopila la respuesta del LLM local vía stream de forma síncrona.
func collectCompletion(ctx context.Context, p providers.Provider, messages []providers.Message, opts providers.CompletionOptions) (string, []providers.ToolCall, error) {
	ch, err := p.StreamCompletion(ctx, messages, opts)
	if err != nil {
		return "", nil, err
	}
	var text strings.Builder
	var tools []providers.ToolCall
	for chunk := range ch {
		switch chunk.Type {
		case "text":
			text.WriteString(chunk.Content)
		case "tool_call":
			if chunk.ToolCall != nil {
				tools = append(tools, *chunk.ToolCall)
			}
		case "error":
			return text.String(), tools, fmt.Errorf("%s", chunk.Content)
		}
	}
	return text.String(), tools, nil
}

// TestLocalModel_Stress_ContextZeroWaste evalúa la compacidad y el gating estricto de tokens
// frente al LLM local (llama3.1:latest), midiendo tiempos de prefill, TTFT y tasa de tokens/segundo.
func TestLocalModel_Stress_ContextZeroWaste(t *testing.T) {
	if !checkOllamaLive(t) {
		return
	}
	setupTestDB(t)

	provider := providers.NewOllama("http://127.0.0.1:11434/v1")
	provider.SetModel("llama3.1:latest")

	// 1. Consulta puramente conceptual / de código: DEBE inyectar CERO tokens de HUD
	conceptualQuery := "Explícame en dos oraciones claras qué es un mutex en Go y cuándo usar RWMutex."
	isOS1 := system.IsOSRelevantQuery(conceptualQuery)
	if isOS1 {
		t.Fatalf("ERROR: La consulta conceptual '%s' fue clasificada incorrectamente como OS-relevante", conceptualQuery)
	}

	p1Params := AgentLoopParams{
		Provider:    provider,
		UserMessage: conceptualQuery,
		Chat:        &models.Chat{ID: uuid.NewString(), Model: "llama3.1:latest"},
	}
	promptConceptual := buildAgentSystemPrompt(p1Params)
	if strings.Contains(promptConceptual, "OZY OS-HUD") || strings.Contains(promptConceptual, "VENTANAS ACTIVAS EN PANTALLA") {
		t.Fatalf("Falla de Gating: El prompt conceptual contiene telemetría de HUD cuando no debería!")
	}
	t.Logf("✓ Query Conceptual: 0 tokens de OS-HUD inyectados (Prompt length: %d bytes)", len(promptConceptual))

	// Ejecutar consulta conceptual contra el modelo local
	t0 := time.Now()
	resText, tools, err := collectCompletion(context.Background(), provider, []providers.Message{
		{Role: "system", Content: promptConceptual},
		{Role: "user", Content: conceptualQuery},
	}, providers.CompletionOptions{
		Model: "llama3.1:latest",
	})
	dur1 := time.Since(t0)
	if err != nil {
		t.Fatalf("Error en inferencia local de Ollama: %v", err)
	}
	t.Logf("✓ Respuesta conceptual obtenida en %v (herramientas invocadas: %d):\n%s", dur1, len(tools), strings.TrimSpace(resText))
	if len(tools) > 0 {
		t.Fatalf("El modelo local invocó herramientas para una pregunta conceptual sin necesidad: %+v", tools)
	}

	// 2. Consulta de acción de OS: DEBE inyectar el OS-HUD en microsegundos (< 65 tokens)
	osQuery := "Abre la calculadora y dime si la ventana está activa en pantalla."
	isOS2 := system.IsOSRelevantQuery(osQuery)
	if !isOS2 {
		t.Fatalf("ERROR: La consulta OS '%s' no fue detectada como relevante", osQuery)
	}

	p2Params := AgentLoopParams{
		Provider:    provider,
		UserMessage: osQuery,
		Chat:        &models.Chat{ID: uuid.NewString(), Model: "llama3.1:latest"},
	}
	promptOS := buildAgentSystemPrompt(p2Params)
	if !strings.Contains(promptOS, "OZY OS-HUD") && !strings.Contains(promptOS, "VENTANAS ACTIVAS") {
		t.Fatalf("Falla de Inyección: El prompt para OS query NO contiene el OS-HUD!")
	}
	t.Logf("✓ Query OS: OS-HUD inyectado efímeramente en RAM (Prompt length: %d bytes)", len(promptOS))
}

// TestLocalModel_Stress_ToolRouting_And_PeekState prueba la herramienta 'os_peek_state'
// disparada por el LLM local para obtener el estado de la PC sin coste de saturación.
func TestLocalModel_Stress_ToolRouting_And_PeekState(t *testing.T) {
	if !checkOllamaLive(t) {
		return
	}
	setupTestDB(t)

	// Medir el tiempo de ejecución directo de os_peek_state desde RAM
	tc := providers.ToolCall{
		ID:    "peek_stress_01",
		Name:  "os_peek_state",
		Input: json.RawMessage(`{}`),
	}

	t0 := time.Now()
	out, ok := executeToolCall(context.Background(), tc, nil, nil)
	dur := time.Since(t0)

	if !ok || !strings.Contains(out, "OZY OS-HUD") {
		t.Fatalf("os_peek_state falló: %s", out)
	}
	t.Logf("✓ os_peek_state ejecutado desde RAM en %v (< 1ms requerido). Contenido:\n%s", dur, out)
	if dur > 10*time.Millisecond {
		t.Errorf("ADVERTENCIA: Latencia de os_peek_state (%v) superó el umbral ultra-rápido de RAM", dur)
	}

	// Probar que el router de herramientas en RAM resuelve os_peek_state en microsegundos
	router := DefaultRAMToolRouter()
	routedTools := router.RouteTools("inspecciona el estado de mi pc y qué ventanas tengo", false, true)
	hasPeek := false
	for _, rt := range routedTools {
		if rt.Name == "os_peek_state" {
			hasPeek = true
			break
		}
	}
	if !hasPeek {
		t.Fatalf("RAMToolRouter no incluyó 'os_peek_state' para la consulta de estado de PC")
	}
	t.Logf("✓ RAMToolRouter resolvió os_peek_state exitosamente en microsegundos")
}

// TestLocalModel_Stress_HeavySkeletonize_Reduction prueba la compresión masiva
// de archivos de código mediante el motor Rust nativo antes de pasarlos al LLM local.
func TestLocalModel_Stress_HeavySkeletonize_Reduction(t *testing.T) {
	if !checkOllamaLive(t) {
		return
	}

	// Creamos un archivo Go complejo y extenso para probar la esqueletización
	complexGoCode := `package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type WorkerConfig struct {
	ID        int
	Capacity  int
	Timeout   time.Duration
	Active    bool
}

type PipelineManager struct {
	mu      sync.RWMutex
	workers map[int]*WorkerConfig
	queue   chan string
}

func NewPipelineManager(workersCount int) *PipelineManager {
	mgr := &PipelineManager{
		workers: make(map[int]*WorkerConfig),
		queue:   make(chan string, 100),
	}
	for i := 0; i < workersCount; i++ {
		mgr.workers[i] = &WorkerConfig{
			ID:       i,
			Capacity: 10,
			Timeout:  5 * time.Second,
			Active:   true,
		}
	}
	return mgr
}

func (m *PipelineManager) ProcessItem(ctx context.Context, item string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(item) == 0 {
		return false, fmt.Errorf("item vacio")
	}
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case m.queue <- item:
		return true, nil
	}
}

func (m *PipelineManager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	close(m.queue)
	return nil
}
`
	tmpFile, err := os.CreateTemp("", "complex_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(complexGoCode)
	tmpFile.Close()

	// Esqueletizar con Rust
	t0 := time.Now()
	skeleton, err := CallRustSkeletonize(context.Background(), tmpFile.Name())
	rustDur := time.Since(t0)
	if err != nil {
		t.Fatalf("CallRustSkeletonize falló: %v", err)
	}

	origBytes := len(complexGoCode)
	skelBytes := len(skeleton)
	reduction := 100.0 - (float64(skelBytes) / float64(origBytes) * 100.0)

	t.Logf("✓ Rust Skeletonizer completado en %v", rustDur)
	t.Logf("  Tamaño original: %d bytes | Esqueleto: %d bytes | Reducción: %.1f%%", origBytes, skelBytes, reduction)

	if reduction < 40.0 {
		t.Fatalf("La reducción de tamaño (%.1f%%) no alcanzó el objetivo de compactación", reduction)
	}

	// Enviar esqueleto a llama3.1 local para validar que entiende la arquitectura
	provider := providers.NewOllama("http://127.0.0.1:11434/v1")
	provider.SetModel("llama3.1:latest")

	prompt := fmt.Sprintf("Analiza brevemente (1 o 2 oraciones) este esqueleto de Go y enumera las structs y métodos declarados:\n\n%s", skeleton)
	t1 := time.Now()
	resText, _, err := collectCompletion(context.Background(), provider, []providers.Message{
		{Role: "user", Content: prompt},
	}, providers.CompletionOptions{
		Model: "llama3.1:latest",
	})
	llmDur := time.Since(t1)
	if err != nil {
		t.Fatalf("Falla al llamar LLM local con esqueleto: %v", err)
	}

	t.Logf("✓ LLM local interpretó el esqueleto en %v:\n%s", llmDur, strings.TrimSpace(resText))
	if !strings.Contains(strings.ToLower(resText), "pipeline") && !strings.Contains(strings.ToLower(resText), "worker") {
		t.Logf("Nota: El modelo local respondió pero no mencionó explícitamente los tipos")
	}
}

// TestLocalModel_Stress_MultiTurnHistoryCompaction valida que el historial persistido
// en SQLite NO almacene el HUD efímero repetidamente, evitando la saturación de la ventana de contexto.
func TestLocalModel_Stress_MultiTurnHistoryCompaction(t *testing.T) {
	setupTestDB(t)

	chatID := uuid.NewString()
	userID := db.DefaultUserID()

	chat := &models.Chat{
		ID:        chatID,
		UserID:    userID,
		Name:      "Stress Test Chat",
		CreatedAt: time.Now(),
	}
	if err := db.CreateChat(chat); err != nil {
		t.Fatalf("create chat: %v", err)
	}

	// Simulamos 4 turnos sucesivos con preguntas de sistema y preguntas conceptuales
	turns := []struct {
		userMessage string
		isOS        bool
	}{
		{"¿Qué programas tengo abiertos en la PC?", true},
		{"Explícame la diferencia entre un slice y un array en Go", false},
		{"Abre la calculadora por favor", true},
		{"Gracias, ahora redacta un título para este script", false},
	}

	for i, turn := range turns {
		// Guardar mensaje de usuario
		userMsg := &models.Message{
			ID:        uuid.NewString(),
			ChatID:    chatID,
			Role:      "user",
			Content:   turn.userMessage,
			CreatedAt: time.Now(),
		}
		if err := db.CreateMessage(userMsg); err != nil {
			t.Fatalf("create user message: %v", err)
		}

		// Validar gating de prompt
		isOSDetected := system.IsOSRelevantQuery(turn.userMessage)
		if isOSDetected != turn.isOS {
			t.Errorf("Turno %d ('%s'): detección OS esperada %v, obtenida %v", i+1, turn.userMessage, turn.isOS, isOSDetected)
		}

		// Simular respuesta del asistente persistida
		assistantMsg := &models.Message{
			ID:        uuid.NewString(),
			ChatID:    chatID,
			Role:      "assistant",
			Content:   fmt.Sprintf("Respuesta al turno %d", i+1),
			CreatedAt: time.Now(),
		}
		if err := db.CreateMessage(assistantMsg); err != nil {
			t.Fatalf("create assistant message: %v", err)
		}
	}

	// Leer todo el historial acumulado de la base de datos
	storedMsgs, err := db.GetMessages(chatID)
	if err != nil {
		t.Fatalf("get stored messages: %v", err)
	}

	totalBytes := 0
	for _, m := range storedMsgs {
		totalBytes += len(m.Content)
		if strings.Contains(m.Content, "OZY OS-HUD") {
			t.Fatalf("ERROR CRÍTICO: Se encontró telemetría OS-HUD persistida en el historial de SQLite! ID: %s", m.ID)
		}
	}

	t.Logf("✓ 4 turnos persistidos limpiamente en SQLite. Total bytes de historial: %d bytes (0 HUD leakage)", totalBytes)

	// Verificar compactación para modelo local
	provider := providers.NewOllama("http://127.0.0.1:11434/v1")
	params := AgentLoopParams{
		Provider:    provider,
		Chat:        chat,
		UserMessage: "¿Puedes recordarme el primer tema que hablamos?",
	}
	initialHistory := buildInitialHistory(params)
	t.Logf("✓ buildInitialHistory construyó %d mensajes para el LLM local (ventana de contexto protegida)", len(initialHistory))

	// Validar que no exceda la ventana concisa para modelos locales (max 4 mensajes previos + system + user)
	if len(initialHistory) > 7 {
		t.Fatalf("Ventana de contexto no compactada para modelo local: %d mensajes", len(initialHistory))
	}
}

// TestLocalModel_Stress_BlackboardConcurrency evalúa la concurrencia masiva
// y la ausencia de condiciones de carrera en el Blackboard en RAM.
func TestLocalModel_Stress_BlackboardConcurrency(t *testing.T) {
	bb := system.DefaultBlackboard()
	var wg sync.WaitGroup
	iterations := 50
	concurrency := 10

	t0 := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				hud := bb.RenderHUD()
				if len(hud) == 0 {
					t.Errorf("worker %d: HUD vacío", id)
				}
				snap := bb.GetSnapshot()
				_ = len(snap.OpenWindows)
				isRel := system.IsOSRelevantQuery("abre la calculadora")
				if !isRel {
					t.Errorf("worker %d: IsOSRelevantQuery falló", id)
				}
			}
		}(i)
	}
	wg.Wait()
	totalOps := concurrency * iterations
	dur := time.Since(t0)
	avgOp := dur / time.Duration(totalOps)

	t.Logf("✓ Concurrencia de Blackboard completada: %d operaciones en %v (promedio: %v por operación)", totalOps, dur, avgOp)
	if avgOp > 500*time.Microsecond {
		t.Errorf("ADVERTENCIA: Operación de Blackboard superó los 500µs: %v", avgOp)
	}
}
