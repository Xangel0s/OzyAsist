package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
)

const maxAgentTurns = 25

// AgentEvent es el protocolo de eventos WS que el loop emite hacia el frontend.
type AgentEvent struct {
	Type string `json:"type"`
	// state:sync
	State string `json:"state,omitempty"` // thinking | executing | awaiting | idle
	// message:delta
	Content string `json:"content,omitempty"`
	// message:done
	MessageID string `json:"message_id,omitempty"`
	// tool:call
	ToolID    string `json:"tool_id,omitempty"`
	ToolName  string `json:"tool_name,omitempty"`
	ToolInput string `json:"tool_input,omitempty"`
	// tool:result
	ToolOutput  string `json:"tool_output,omitempty"`
	ToolSuccess bool   `json:"tool_success,omitempty"`
	DurationMs  int64  `json:"duration_ms,omitempty"`
	// agent:completed
	TaskID string `json:"task_id,omitempty"`
	Turns  int    `json:"turns,omitempty"`
	// error
	Error string `json:"error,omitempty"`
}

// ApprovalRequest es una solicitud de aprobación pendiente.
type ApprovalRequest struct {
	ToolCall *providers.ToolCall
	Response chan bool // true=approved, false=denied
}

// AgentLoopParams contiene todos los parámetros para arrancar un loop agéntico.
type AgentLoopParams struct {
	Provider        providers.Provider
	Chat            *models.Chat
	Project         *models.Project
	UserMessage     string
	PermissionLevel string
	Emit            func(AgentEvent) // callback para emitir eventos WS
}

// LoopSession representa una sesión de agente activa, con su canal de cancelación
// y de respuestas de aprobación, indexada por sessionID.
type LoopSession struct {
	Cancel          context.CancelFunc
	ApprovalPending sync.Map // tool_id -> chan bool
}

var (
	activeSessions   sync.Map // sessionID -> *LoopSession
)

// StartAgentLoop arranca el ReAct Loop en una goroutine dedicada y devuelve sessionID.
// El loop corre de forma asíncrona y emite eventos via params.Emit.
func StartAgentLoop(ctx context.Context, params AgentLoopParams) string {
	sessionID := uuid.NewString()
	loopCtx, cancel := context.WithCancel(ctx)

	session := &LoopSession{Cancel: cancel}
	activeSessions.Store(sessionID, session)

	go func() {
		defer func() {
			activeSessions.Delete(sessionID)
			cancel()
		}()
		runReActLoop(loopCtx, sessionID, session, params)
	}()

	return sessionID
}

// CancelSession cancela un loop activo por sessionID.
func CancelSession(sessionID string) {
	if v, ok := activeSessions.Load(sessionID); ok {
		v.(*LoopSession).Cancel()
	}
}

// RespondApproval responde a una solicitud de aprobación de herramienta pendiente.
func RespondApproval(sessionID, toolID string, approved bool) {
	if v, ok := activeSessions.Load(sessionID); ok {
		session := v.(*LoopSession)
		if ch, ok := session.ApprovalPending.Load(toolID); ok {
			ch.(chan bool) <- approved
		}
	}
}

// runReActLoop implementa el bucle ReAct autónomo:
// 1. Llama al LLM con el historial + tool definitions
// 2. Si el LLM emite texto → streamearlo via Emit(message:delta)
// 3. Si el LLM solicita tools → ejecutarlas y añadir resultados al historial
// 4. Repetir hasta que el LLM no solicite más tools (respuesta final) o se alcance maxTurns
func runReActLoop(ctx context.Context, sessionID string, session *LoopSession, params AgentLoopParams) {
	emit := params.Emit
	sandbox := buildSandbox(params.Project)
	permLevel := PermissionLevel(params.PermissionLevel)
	if permLevel == "" {
		permLevel = Sandboxed
	}

	// --- Construir historial inicial ---
	history := buildInitialHistory(params)

	taskID := uuid.NewString()
	emit(AgentEvent{Type: "agent:start", TaskID: taskID})

	var finalContent string
	var allToolCalls []providers.ToolCall

	for turn := 0; turn < maxAgentTurns; turn++ {
		if ctx.Err() != nil {
			emit(AgentEvent{Type: "error", Error: "loop cancelado por el usuario"})
			return
		}

		emit(AgentEvent{Type: "state:sync", State: "thinking"})

		// --- Llamada al LLM con tool definitions ---
		chunkCh, err := params.Provider.StreamCompletion(ctx, history, providers.CompletionOptions{
			Stream: true,
			Model:  params.Chat.Model,
			Tools:  AgentTools,
		})
		if err != nil {
			emit(AgentEvent{Type: "error", Error: "LLM error: " + err.Error()})
			return
		}

		// --- Consumir el stream del LLM en este turno ---
		var turnText string
		var turnToolCalls []providers.ToolCall

		for chunk := range chunkCh {
			if ctx.Err() != nil {
				emit(AgentEvent{Type: "error", Error: "cancelado"})
				return
			}
			switch chunk.Type {
			case "text":
				turnText += chunk.Content
				emit(AgentEvent{Type: "message:delta", Content: chunk.Content})
			case "tool_call":
				if chunk.ToolCall != nil {
					turnToolCalls = append(turnToolCalls, *chunk.ToolCall)
				}
			case "error":
				emit(AgentEvent{Type: "error", Error: chunk.Content})
				return
			}
		}

		finalContent += turnText

		// --- Añadir turno del assistant al historial ---
		assistantMsg := providers.Message{
			Role:    "assistant",
			Content: turnText,
		}
		if len(turnToolCalls) > 0 {
			assistantMsg.ToolCalls = turnToolCalls
		}
		history = append(history, assistantMsg)

		// --- Si no hay tool calls → el LLM terminó ---
		if len(turnToolCalls) == 0 {
			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1})
			return
		}

		allToolCalls = append(allToolCalls, turnToolCalls...)

		// --- Ejecutar cada tool call y añadir resultados ---
		for _, tc := range turnToolCalls {
			if ctx.Err() != nil {
				emit(AgentEvent{Type: "error", Error: "cancelado"})
				return
			}

			inputJSON, _ := json.Marshal(tc.Input)
			emit(AgentEvent{
				Type:      "tool:call",
				ToolID:    tc.ID,
				ToolName:  tc.Name,
				ToolInput: string(inputJSON),
			})

			// --- Verificar autorización / permisos ---
			step := toolCallToPlanStep(tc)
			auth, err := ValidateAndAuthorize(step, permLevel, sandbox)
			if err != nil {
				toolResult := fmt.Sprintf("ERROR DE AUTORIZACIÓN: %v", err)
				emit(AgentEvent{Type: "tool:result", ToolID: tc.ID, ToolOutput: toolResult, ToolSuccess: false})
				history = appendToolResult(history, tc.ID, toolResult)
				continue
			}

			// --- Si requiere aprobación del usuario → suspender loop ---
			if auth.RequiresConfirmation {
				emit(AgentEvent{
					Type:      "tool:approval_request",
					ToolID:    tc.ID,
					ToolName:  tc.Name,
					ToolInput: string(inputJSON),
				})
				emit(AgentEvent{Type: "state:sync", State: "awaiting"})

				approved := waitForApproval(ctx, session, tc.ID)
				if !approved {
					toolResult := "Herramienta denegada por el usuario."
					emit(AgentEvent{Type: "tool:result", ToolID: tc.ID, ToolOutput: toolResult, ToolSuccess: false})
					history = appendToolResult(history, tc.ID, toolResult)
					continue
				}
			}

			// --- Ejecutar herramienta ---
			emit(AgentEvent{Type: "state:sync", State: "executing"})
			start := time.Now()
			output, success := executeToolCall(ctx, tc, auth, sandbox)
			durationMs := time.Since(start).Milliseconds()

			emit(AgentEvent{
				Type:        "tool:result",
				ToolID:      tc.ID,
				ToolOutput:  output,
				ToolSuccess: success,
				DurationMs:  durationMs,
			})

			// --- Añadir resultado al historial para el siguiente turno ---
			history = appendToolResult(history, tc.ID, output)
		}
	}

	// Si llegamos aquí, se superó maxTurns
	emit(AgentEvent{Type: "error", Error: fmt.Sprintf("máximo de turnos alcanzado (%d)", maxAgentTurns)})
}

// buildInitialHistory construye el historial de mensajes inicial para el loop,
// incluyendo system prompt, contexto de proyecto, memoria episódica, e historial previo del chat.
func buildInitialHistory(params AgentLoopParams) []providers.Message {
	systemPrompt := buildAgentSystemPrompt(params)
	history := []providers.Message{{Role: "system", Content: systemPrompt}}

	// Historial previo del chat
	prevMessages, err := db.GetMessages(params.Chat.ID)
	if err != nil {
		log.Printf("agent loop: error cargando historial: %v", err)
	}
	for _, m := range prevMessages {
		history = append(history, providers.Message{Role: m.Role, Content: m.Content})
	}

	// Añadir el mensaje del usuario actual
	history = append(history, providers.Message{Role: "user", Content: params.UserMessage})
	return history
}

func buildAgentSystemPrompt(params AgentLoopParams) string {
	base := `Eres Ozy, un agente de código autónomo con acceso a herramientas reales del sistema de archivos.

MODO DE OPERACIÓN:
- Tienes acceso a herramientas para leer/escribir archivos, ejecutar comandos, buscar texto y explorar el proyecto.
- Usa las herramientas de forma iterativa: lee primero para entender, luego modifica con precisión.
- Para modificaciones quirúrgicas usa apply_diff. Para crear archivos nuevos o reescrituras completas usa write_file.
- Después de ejecutar comandos, verifica el resultado antes de continuar.
- Cuando termines, responde con un resumen claro de lo que hiciste.

REGLAS:
- No inventes resultados. Si una herramienta falla, reporta el error y ajusta tu estrategia.
- No ejecutes comandos destructivos sin confirmación explícita del usuario.
- Usa search_text y list_files antes de asumir la estructura del proyecto.
- Responde siempre en español.`

	if params.Project != nil {
		if params.Project.InstructionsMd != "" {
			base = fmt.Sprintf("[Instrucciones del proyecto \"%s\"]:\n%s\n\n---\n\n%s",
				params.Project.Name, params.Project.InstructionsMd, base)
		}
		// Contexto del grafo de dependencias
		graphCtx := memory.BuildGraphContext(params.Project.ID, params.UserMessage)
		if graphCtx != "" {
			base += "\n\n[Contexto de dependencias del proyecto]:\n" + graphCtx
		}
	}

	return base
}

// appendToolResult añade el resultado de una herramienta al historial.
func appendToolResult(history []providers.Message, toolCallID, content string) []providers.Message {
	return append(history, providers.Message{
		Role: "tool",
		ToolResult: &providers.ToolResult{
			ToolCallID: toolCallID,
			Content:    content,
		},
	})
}

// waitForApproval suspende el loop hasta recibir la respuesta de aprobación del usuario.
// Retorna true si aprobado, false si denegado o si el context fue cancelado.
func waitForApproval(ctx context.Context, session *LoopSession, toolID string) bool {
	ch := make(chan bool, 1)
	session.ApprovalPending.Store(toolID, ch)
	defer session.ApprovalPending.Delete(toolID)

	select {
	case approved := <-ch:
		return approved
	case <-ctx.Done():
		return false
	case <-time.After(5 * time.Minute): // timeout de aprobación
		return false
	}
}

// persistAgentMessage guarda el mensaje final del agente en SQLite y en memoria episódica.
func persistAgentMessage(params AgentLoopParams, taskID, content string, toolCalls []providers.ToolCall) string {
	msgID := taskID
	msg := &models.Message{
		ID:        msgID,
		ChatID:    params.Chat.ID,
		Role:      "assistant",
		Content:   content,
		CreatedAt: time.Now(),
	}
	if len(toolCalls) > 0 {
		tcs := make([]map[string]any, len(toolCalls))
		for i, tc := range toolCalls {
			tcs[i] = map[string]any{"id": tc.ID, "name": tc.Name, "input": string(tc.Input)}
		}
		tcJSON, _ := json.Marshal(tcs)
		msg.ToolCallsJSON = string(tcJSON)
	}
	if err := db.CreateMessage(msg); err != nil {
		log.Printf("agent loop: error guardando mensaje: %v", err)
	}
	memory.StoreChatMessage(params.Chat.UserID, params.Chat.ProjectID, params.Chat.ID, "assistant", content)
	return msgID
}

// buildSandbox crea el sandbox del proyecto si está disponible.
func buildSandbox(project *models.Project) *Sandbox {
	if project == nil || project.RootPath == "" {
		return nil
	}
	sb, err := NewSandbox(project.RootPath)
	if err != nil {
		log.Printf("agent loop: no se pudo crear sandbox: %v", err)
		return nil
	}
	return sb
}
