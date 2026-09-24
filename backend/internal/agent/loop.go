package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/voice"
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
	Thinking    string `json:"thinking,omitempty"`
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
	UserID          string           // opcional: usuario de la sesión
	// VoiceMode reduce el system prompt y las tools al mínimo para minimizar
	// el tiempo de prefill y conseguir respuestas rápidas (~2-3s vs ~15s).
	VoiceMode bool
}

// LoopSession representa una sesión de agente activa, con su canal de cancelación
// y de respuestas de aprobación, indexada por sessionID.
type LoopSession struct {
	Cancel          context.CancelFunc
	ApprovalPending sync.Map // tool_id -> chan bool
}

var (
	activeSessions     sync.Map // sessionID -> *LoopSession
	activeChatSessions sync.Map // chatID -> sessionID
)

// StartAgentLoop arranca el ReAct Loop en una goroutine dedicada y devuelve sessionID.
// El loop corre de forma asíncrona y emite eventos via params.Emit.
func StartAgentLoop(ctx context.Context, params AgentLoopParams) string {
	sessionID := uuid.NewString()
	loopCtx, cancel := context.WithCancel(ctx)

	session := &LoopSession{Cancel: cancel}
	activeSessions.Store(sessionID, session)
	if params.Chat != nil && params.Chat.ID != "" {
		activeChatSessions.Store(params.Chat.ID, sessionID)
	}

	go func() {
		defer func() {
			activeSessions.Delete(sessionID)
			if params.Chat != nil && params.Chat.ID != "" {
				activeChatSessions.Delete(params.Chat.ID)
			}
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

// CancelChat cancela cualquier sesión agéntica activa asociada a un chatID.
func CancelChat(chatID string) bool {
	if sessID, ok := activeChatSessions.Load(chatID); ok {
		CancelSession(sessID.(string))
		return true
	}
	return false
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

type ContextKey string

const UserMessageContextKey ContextKey = "user_message"

func UserMessageFromContext(ctx context.Context) string {
	if v := ctx.Value(UserMessageContextKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// runReActLoop implementa el bucle ReAct autónomo:
// 1. Llama al LLM con el historial + tool definitions
// 2. Si el LLM emite texto → streamearlo via Emit(message:delta)
// 3. Si el LLM solicita tools → ejecutarlas y añadir resultados al historial
// 4. Repetir hasta que el LLM no solicite más tools (respuesta final) o se alcance maxTurns
func runReActLoop(ctx context.Context, sessionID string, session *LoopSession, params AgentLoopParams) {
	ctx = context.WithValue(ctx, UserMessageContextKey, params.UserMessage)
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
	var pendingRequirementPrompted bool
	charcAuditor := NewCharcAuditor(nil)

	var speakerQueue *voice.LocalSpeakerQueue
	if params.VoiceMode {
		speakerQueue = voice.NewLocalSpeakerQueue(ctx)
		defer speakerQueue.Close()
	}

	graph := memory.GetSystemGraph()

	// --- 1. REFLEJO DE ROLLBACK (Deshacer instantáneo en 1ms, 0 tokens) ---
	if graph != nil && graph.IsRollbackQuery(params.UserMessage) {
		if revertState, ok := graph.PopRollback(); ok {
			emit(AgentEvent{Type: "state:sync", State: "executing"})

			revInput, _ := json.Marshal(revertState.RevertArgs)
			revTC := providers.ToolCall{
				ID:    "rollback_" + uuid.NewString()[:8],
				Name:  revertState.RevertToolName,
				Input: json.RawMessage(revInput),
			}
			emit(AgentEvent{
				Type:      "tool:call",
				ToolID:    revTC.ID,
				ToolName:  revTC.Name,
				ToolInput: string(revInput),
			})

			step := toolCallToPlanStep(revTC)
			auth := &AuthorizedAction{Step: step}
			res, success := executeToolCall(ctx, revTC, auth, sandbox)
			emit(AgentEvent{
				Type:        "tool:result",
				ToolID:      revTC.ID,
				ToolName:    revTC.Name,
				ToolInput:   string(revInput),
				ToolOutput:  res,
				ToolSuccess: success,
			})

			rollbackReply := fmt.Sprintf("Listo, deshecho: %s.", revertState.Description)
			emit(AgentEvent{Type: "message:delta", Content: rollbackReply})
			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, rollbackReply, []providers.ToolCall{revTC})
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: 1, Content: rollbackReply})
			return
		}
	}

	// --- 2. REFLEJO FAST-TRACK (Bypass de LLM para órdenes deterministas en ~1ms, 0 tokens) ---
	if graph != nil {
		if matches, ok := graph.ResolveMultiIntent(params.UserMessage); ok && len(matches) > 0 {
			emit(AgentEvent{Type: "state:sync", State: "executing"})

			var executedToolCalls []providers.ToolCall
			allSuccess := true

			for _, match := range matches {
				// Registrar estado previo para rollback si aplica a audio
				if match.Engram.ToolName == "os_audio_device" {
					if act, ok := match.ExtractedArgs["action"].(string); ok && act == "mute" {
						isMute, _ := match.ExtractedArgs["mute"].(bool)
						graph.RecordRollback(memory.RollbackState{
							EngramID:       match.Engram.ID,
							ToolName:       "os_audio_device",
							ActionTaken:    "mute",
							RevertToolName: "os_audio_device",
							RevertArgs:     map[string]any{"action": "mute", "mute": !isMute},
							Description:    "Restaurar silencio de audio",
						})
					}
				}

				ftInput, _ := json.Marshal(match.ExtractedArgs)
				ftTC := providers.ToolCall{
					ID:    "fasttrack_" + uuid.NewString()[:8],
					Name:  match.Engram.ToolName,
					Input: json.RawMessage(ftInput),
				}

				emit(AgentEvent{
					Type:      "tool:call",
					ToolID:    ftTC.ID,
					ToolName:  ftTC.Name,
					ToolInput: string(ftInput),
				})

				step := toolCallToPlanStep(ftTC)
				auth := &AuthorizedAction{Step: step}
				res, success := executeToolCall(ctx, ftTC, auth, sandbox)
				emit(AgentEvent{
					Type:        "tool:result",
					ToolID:      ftTC.ID,
					ToolName:    ftTC.Name,
					ToolInput:   string(ftInput),
					ToolOutput:  res,
					ToolSuccess: success,
				})

				if success {
					graph.PromoteEngram(match.Engram.ID)
				} else {
					allSuccess = false
				}
				executedToolCalls = append(executedToolCalls, ftTC)
			}

			reply := memory.SynthesizeMultiFeedback(matches)
			if !allSuccess && reply == "" {
				reply = "Listo, he ejecutado las acciones solicitadas."
			}

			emit(AgentEvent{Type: "message:delta", Content: reply})
			if params.VoiceMode && speakerQueue != nil {
				speakerQueue.Enqueue(reply)
			}
			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, reply, executedToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: 1, Content: reply})
			return
		}
	}

	for turn := 0; turn < maxAgentTurns; turn++ {
		if ctx.Err() != nil {
			emit(AgentEvent{Type: "error", Error: "loop cancelado por el usuario"})
			return
		}

		emit(AgentEvent{Type: "state:sync", State: "thinking"})

		// --- Llamada al LLM con tool definitions (Poda Dinámica para modelos locales / voz) ---
		isLocal := params.Provider != nil && (params.Provider.Name() == "llamacpp" || params.Provider.Name() == "ollama" || params.Provider.Name() == "lmstudio")
		tools := GetActiveToolsForQuery(params.UserMessage, params.VoiceMode, isLocal)
		chunkCh, err := params.Provider.StreamCompletion(ctx, history, providers.CompletionOptions{
			Stream: true,
			Model:  params.Chat.Model,
			Tools:  tools,
		})
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "connectex") || strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "dial tcp") {
				provName := "modelo"
				if params.Provider != nil {
					provName = params.Provider.Name()
				}
				emit(AgentEvent{
					Type: "error",
					Error: fmt.Sprintf("No se pudo conectar con el servidor LLM (%s).\n"+
						"💡 Solución:\n"+
						"  1. Si usas Ollama o LM Studio, verifica que esté iniciado en tu PC.\n"+
						"  2. Si prefieres un modelo en la nube (OpenRouter, OpenAI, DeepSeek), escribe en la TUI:\n"+
						"     /key openrouter <tu-api-key>\n"+
						"     /key openai <tu-api-key>", provName),
				})
			} else {
				emit(AgentEvent{Type: "error", Error: "LLM error: " + errStr})
			}
			return
		}

		// --- Consumir el stream del LLM en este turno ---
		var turnText string
		var turnThinking string
		var turnToolCalls []providers.ToolCall
		var textChunks []string

		var sentenceStreamer *voice.SentenceStreamer
		if params.VoiceMode && speakerQueue != nil {
			sentenceStreamer = voice.NewSentenceStreamer(func(sentence string) {
				emit(AgentEvent{Type: "voice:sentence", Content: sentence})
				speakerQueue.Enqueue(sentence)
			})
		}

		var inStreamThought bool
		for chunk := range chunkCh {
			if ctx.Err() != nil {
				emit(AgentEvent{Type: "error", Error: "cancelado"})
				return
			}
			switch chunk.Type {
			case "thinking":
				turnThinking += chunk.Content
				emit(AgentEvent{Type: "agent:thinking", Content: chunk.Content})
			case "text":
				turnText += chunk.Content
				textChunks = append(textChunks, chunk.Content)

				if strings.Contains(chunk.Content, "<thought>") || strings.Contains(chunk.Content, "<think>") {
					inStreamThought = true
				}
				if inStreamThought {
					emit(AgentEvent{Type: "agent:thinking", Content: chunk.Content})
					if strings.Contains(chunk.Content, "</thought>") || strings.Contains(chunk.Content, "</think>") {
						inStreamThought = false
					}
				} else {
					if sentenceStreamer != nil && len(turnToolCalls) == 0 {
						sentenceStreamer.Feed(chunk.Content)
					}
				}
			case "tool_call":
				if chunk.ToolCall != nil {
					turnToolCalls = append(turnToolCalls, *chunk.ToolCall)
				}
				if speakerQueue != nil {
					speakerQueue.Cancel()
				}
			case "error":
				emit(AgentEvent{Type: "error", Error: chunk.Content})
				return
			}
		}

		// Si el modelo incluye etiquetas <think>...</think> o <thought>...</thought> en el texto principal
		if strings.Contains(turnText, "<think>") && strings.Contains(turnText, "</think>") {
			start := strings.Index(turnText, "<think>")
			end := strings.Index(turnText, "</think>")
			if end > start {
				extractedThought := turnText[start+7 : end]
				turnThinking += strings.TrimSpace(extractedThought)
				turnText = strings.TrimSpace(turnText[:start] + turnText[end+8:])
			}
		}
		if strings.Contains(turnText, "<thought>") && strings.Contains(turnText, "</thought>") {
			start := strings.Index(turnText, "<thought>")
			end := strings.Index(turnText, "</thought>")
			if end > start {
				extractedThought := turnText[start+9 : end]
				turnThinking += strings.TrimSpace(extractedThought)
				turnText = strings.TrimSpace(turnText[:start] + turnText[end+10:])
			}
		}

		// Limpiar artefactos comunes de modelos locales cuantizados (ej: literales \n al inicio)
		for strings.HasPrefix(turnText, `\n`) || strings.HasPrefix(turnText, `\r\n`) {
			turnText = strings.TrimPrefix(turnText, `\n`)
			turnText = strings.TrimPrefix(turnText, `\r\n`)
			turnText = strings.TrimSpace(turnText)
		}

		// --- Parsear tool calls del texto si el provider falló en parsearlos nativamente ---
		// extractToolCallsFromText valida estrictamente que la herramienta pertenezca a las registradas.
		if len(turnToolCalls) == 0 {
			turnToolCalls = extractToolCallsFromText(turnText)
		}
		// Red de seguridad agéntica: Si el LLM no emitió tool calls o emitió una llamada genérica ("las ventanas")
		if turn == 0 {
			hasGenericClose := false
			if len(turnToolCalls) == 1 && turnToolCalls[0].Name == "os_close_window" {
				var p struct{ Title string `json:"title"` }
				_ = json.Unmarshal(turnToolCalls[0].Input, &p)
				pTitle := strings.ToLower(strings.TrimSpace(p.Title))
				if pTitle == "ventanas" || pTitle == "las ventanas" || pTitle == "ambos" || pTitle == "ambas" || pTitle == "los programas" {
					hasGenericClose = true
				}
			}
			if len(turnToolCalls) == 0 || hasGenericClose {
				rescued := rescueDirectUserIntent(params.UserMessage, turnText)
				if len(rescued) > 0 {
					turnToolCalls = rescued
				}
			}
		}

		// Red de seguridad anti-fuga: Si el modelo emitió un bloque JSON de llamada a herramienta
		// pero no fue reconocido por el parser estándar, intentar rescate semántico inmediato.
		if len(turnToolCalls) == 0 && isLeakedToolJSON(turnText) {
			if rescuedCalls := tryRescueLeakedJSON(turnText); len(rescuedCalls) > 0 {
				turnToolCalls = rescuedCalls
			}
		}

		// --- Añadir turno del assistant al historial ---
		assistantMsg := providers.Message{
			Role: "assistant",
		}
		if len(turnToolCalls) > 0 {
			assistantMsg.ToolCalls = turnToolCalls
			// NO asignamos Content cuando hay herramientas invocadas: evita que textos de
			// rechazo o comentarios preliminares contaminen el contexto del LLM en los turnos posteriores.
			assistantMsg.Content = ""
		} else {
			// Si todavía es un JSON huérfano sin rescatar, NUNCA emitirlo como texto crudo al usuario
			if isLeakedToolJSON(turnText) {
				turnText = "Entendido. Estoy procesando la información requerida en el sistema..."
				textChunks = []string{turnText}
			}

			// Filtrar alucinaciones de tokens repetitivos (\n\n\n...)
			cleanCheck := strings.ReplaceAll(turnText, `\n`, "")
			cleanCheck = strings.ReplaceAll(cleanCheck, `\r`, "")
			cleanCheck = strings.ReplaceAll(cleanCheck, `\`, "")
			cleanCheck = strings.TrimSpace(cleanCheck)
			if cleanCheck == "" {
				if len(allToolCalls) > 0 {
					turnText = "Listo, he completado la acción solicitada. ¿Deseas hacer algo más?"
				} else if turnThinking != "" {
					turnText = turnThinking
				} else {
					turnText = "Entendido. ¿En qué más te puedo colaborar o qué acción necesitas realizar?"
				}
				textChunks = []string{turnText}
			}

			assistantMsg.Content = turnText
			// Si NO hubo llamadas a herramientas, este turno contiene la respuesta final al usuario.
			// Emitir los chunks de texto y acumular en finalContent.
			for _, chunkStr := range textChunks {
				emit(AgentEvent{Type: "message:delta", Content: chunkStr})
			}
			finalContent += turnText
			if sentenceStreamer != nil {
				sentenceStreamer.Flush()
			}
		}
		history = append(history, assistantMsg)

		// --- Si no hay tool calls → verificar si hay pasos pendientes del plan autónomo ---
		if len(turnToolCalls) == 0 {
			if !pendingRequirementPrompted {
				if pendingPrompt := checkPendingTaskRequirements(params.UserMessage, allToolCalls); pendingPrompt != "" && turn < maxAgentTurns-1 {
					pendingRequirementPrompted = true
					history = append(history, providers.Message{
						Role:    "user",
						Content: pendingPrompt,
					})
					continue
				}
			}

			// Si hubo llamadas a herramientas y el LLM no emitió texto final explicativo
			if strings.TrimSpace(finalContent) == "" && len(allToolCalls) > 0 {
				finalContent = "He completado la acción solicitada."
				emit(AgentEvent{Type: "message:delta", Content: finalContent})
				if speakerQueue != nil {
					speakerQueue.Enqueue(finalContent)
				}
			}

			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent, Thinking: turnThinking})
			
			// --- Auto-Skill Evaluation ---
			if !params.VoiceMode && len(allToolCalls) > 0 {
				EvaluateAndSaveSkill(params.Provider, history, params.UserMessage)
				RecordInteractionForTraining(params.UserMessage, finalContent, allToolCalls)
			}

			// --- Auto-Memory Extraction (Continuous Learning) ---
			userForMem := params.UserID
			if userForMem == "" && params.Chat != nil {
				userForMem = params.Chat.UserID
			}
			if userForMem == "" {
				userForMem = db.DefaultUserID()
			}
			if extractor := memory.DefaultExtractor(); extractor != nil && !params.VoiceMode && len(strings.TrimSpace(params.UserMessage)) > 8 {
				extractor.ExtractAndPersistAsync(context.Background(), userForMem, params.UserMessage, finalContent)
			}
			
			// Si no hubo stream de voz (por ejemplo respuesta instantánea sin deltas), encolar respuesta final
			if params.VoiceMode && speakerQueue != nil && finalContent != "" && !speakerQueue.IsPlaying() {
				speakerQueue.Enqueue(finalContent)
			}
			
			return
		}

		allToolCalls = append(allToolCalls, turnToolCalls...)

		// --- Ejecutar cada tool call y añadir resultados ---
		toolSuccessMap := make(map[string]bool)
		for _, tc := range turnToolCalls {
			if ctx.Err() != nil {
				emit(AgentEvent{Type: "error", Error: "cancelado"})
				return
			}

			inputStr := string(tc.Input)
			emit(AgentEvent{
				Type:      "tool:call",
				ToolID:    tc.ID,
				ToolName:  tc.Name,
				ToolInput: inputStr,
			})

			// --- Verificar autorización / permisos ---
			step := toolCallToPlanStep(tc)
			auth, err := ValidateAndAuthorize(step, permLevel, sandbox)
			if err != nil {
				toolResult := fmt.Sprintf("ERROR DE AUTORIZACIÓN: %v", err)
				emit(AgentEvent{
					Type:        "tool:result",
					ToolID:      tc.ID,
					ToolName:    tc.Name,
					ToolInput:   inputStr,
					ToolOutput:  toolResult,
					ToolSuccess: false,
				})
				history = appendToolResult(history, tc.ID, toolResult)
				continue
			}

			// --- Auditoría CHARC (Seguridad y Loop Breaker Heurístico) ---
			taskStep := &models.TaskStep{
				ID:         tc.ID,
				StepOrder:  turn + 1,
				ActionType: step.ActionType,
				Payload:    string(tc.Input),
			}
			charcDecision := charcAuditor.AuditPreExecution(ctx, models.AgentTask{ID: taskID, Title: params.UserMessage}, taskStep)
			if charcDecision.EscalateToNine {
				nineAdvice := FormulateNineIntervention(ctx, params.Provider, params.UserMessage, tc.Name, string(tc.Input), charcDecision.Reason)
				emit(AgentEvent{Type: "agent:thinking", Content: nineAdvice})
				log.Printf("[loop:charc->nine] Intervención de Nine activada: %s", nineAdvice)
				history = appendToolResult(history, tc.ID, nineAdvice)
				continue
			}
			if charcDecision.RequiresPIN || charcDecision.RiskLevel == RiskLevelCritical {
				auth.RequiresConfirmation = true
			}

			// --- Si requiere aprobación del usuario → suspender loop ---
			if auth.RequiresConfirmation && (params.Project == nil || params.Project.AgentConsent != "always") {
				emit(AgentEvent{
					Type:      "tool:approval_request",
					ToolID:    tc.ID,
					ToolName:  tc.Name,
					ToolInput: inputStr,
				})
				emit(AgentEvent{Type: "state:sync", State: "awaiting"})

				approved := waitForApproval(ctx, session, tc.ID)
				if !approved {
					toolResult := "Herramienta denegada por el usuario."
					emit(AgentEvent{
						Type:        "tool:result",
						ToolID:      tc.ID,
						ToolName:    tc.Name,
						ToolInput:   inputStr,
						ToolOutput:  toolResult,
						ToolSuccess: false,
					})
					history = appendToolResult(history, tc.ID, toolResult)
					continue
				}
			}

			// --- Ejecutar herramienta ---
			emit(AgentEvent{Type: "state:sync", State: "executing"})
			start := time.Now()
			output, success := executeToolCall(ctx, tc, auth, sandbox)
			toolSuccessMap[tc.ID] = success
			durationMs := time.Since(start).Milliseconds()

			emit(AgentEvent{
				Type:        "tool:result",
				ToolID:      tc.ID,
				ToolName:    tc.Name,
				ToolInput:   inputStr,
				ToolOutput:  output,
				ToolSuccess: success,
				DurationMs:  durationMs,
			})

			// --- Añadir resultado al historial para el siguiente turno ---
			history = appendToolResult(history, tc.ID, output)
		}

		// Fast completion determinista para modelos locales
		if handleLocalModelFastCompletion(params, taskID, turn, turnThinking, turnToolCalls, toolSuccessMap, history, allToolCalls, emit, speakerQueue) {
			return
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

	// Historial previo del chat (ventana deslizante de los últimos 15 mensajes)
	var prevMessages []models.Message
	if params.Chat != nil && params.Chat.ID != "" {
		msgs, err := db.GetMessages(params.Chat.ID)
		if err != nil {
			log.Printf("agent loop: error cargando historial: %v", err)
		} else {
			prevMessages = msgs
		}
	}
	maxPrev := 15
	if params.Provider != nil && (params.Provider.Name() == "llamacpp" || params.Provider.Name() == "ollama" || params.Provider.Name() == "lmstudio") {
		maxPrev = 4 // Ventana concisa para evitar alucinaciones por contexto previo en modelos locales pequeños
	}
	if len(prevMessages) > maxPrev {
		prevMessages = prevMessages[len(prevMessages)-maxPrev:]
	}

	alreadyAppendedCurrent := false
	if len(prevMessages) > 0 {
		lastMsg := prevMessages[len(prevMessages)-1]
		if lastMsg.Role == "user" && strings.TrimSpace(lastMsg.Content) == strings.TrimSpace(params.UserMessage) {
			alreadyAppendedCurrent = true
		}
	}

	for _, m := range prevMessages {
		content := m.Content
		if len(content) > 2500 {
			content = content[:2500] + "\n[...historial previo truncado para optimizar tokens...]"
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		history = append(history, providers.Message{Role: m.Role, Content: content})
	}

	// Añadir el mensaje del usuario actual únicamente si no estaba ya al final
	if !alreadyAppendedCurrent && strings.TrimSpace(params.UserMessage) != "" {
		history = append(history, providers.Message{Role: "user", Content: params.UserMessage})
	}

	// Saneamiento de protocolo: garantizar que el primer mensaje tras system sea 'user'
	// y que no haya roles consecutivos iguales que rompan la API de Cohere o Anthropic.
	history = sanitizeHistoryRoles(history)
	return history
}

func sanitizeHistoryRoles(msgs []providers.Message) []providers.Message {
	if len(msgs) <= 1 {
		return msgs
	}
	out := make([]providers.Message, 0, len(msgs))
	out = append(out, msgs[0]) // system prompt

	firstContentFound := false
	for _, m := range msgs[1:] {
		// La API de Cohere y Anthropic exigen que el primer mensaje después de system sea 'user'
		if !firstContentFound {
			if m.Role != "user" {
				continue // Ignora mensajes huérfanos de asistente iniciales
			}
			firstContentFound = true
			out = append(out, m)
			continue
		}

		// Evitar mensajes de roles consecutivos iguales (ej: user seguido de user)
		lastRole := out[len(out)-1].Role
		if m.Role == lastRole && m.Role != "tool" {
			out[len(out)-1].Content += "\n\n" + m.Content
		} else {
			out = append(out, m)
		}
	}

	// Si tras el filtrado no quedó ningún mensaje de usuario, añadir el último disponible
	if len(out) == 1 && len(msgs) > 1 {
		out = append(out, providers.Message{Role: "user", Content: msgs[len(msgs)-1].Content})
	}
	return out
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

// buildSandbox crea el sandbox del proyecto o el espacio de trabajo local si no hay proyecto vinculado.
func buildSandbox(project *models.Project) *Sandbox {
	rootPath := "."
	if project != nil && project.RootPath != "" {
		rootPath = project.RootPath
	}
	sb, err := NewSandbox(rootPath)
	if err != nil {
		log.Printf("agent loop: no se pudo crear sandbox en %s: %v", rootPath, err)
		return nil
	}
	return sb
}
