package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/mcp"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
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

	for turn := 0; turn < maxAgentTurns; turn++ {
		if ctx.Err() != nil {
			emit(AgentEvent{Type: "error", Error: "loop cancelado por el usuario"})
			return
		}

		emit(AgentEvent{Type: "state:sync", State: "thinking"})

		// --- Llamada al LLM con tool definitions ---
		// GetActiveTools incluye dinámicamente las herramientas MCP registradas en modo texto/consola
		// y las aísla en VoiceMode para mantener latencia ultra-baja.
		tools := GetActiveTools(params.VoiceMode)
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
			case "tool_call":
				if chunk.ToolCall != nil {
					turnToolCalls = append(turnToolCalls, *chunk.ToolCall)
				}
			case "error":
				emit(AgentEvent{Type: "error", Error: chunk.Content})
				return
			}
		}

		// Si el modelo incluye etiquetas <think>...</think> en el texto principal
		if strings.Contains(turnText, "<think>") && strings.Contains(turnText, "</think>") {
			start := strings.Index(turnText, "<think>")
			end := strings.Index(turnText, "</think>")
			if end > start {
				extractedThought := turnText[start+7 : end]
				turnThinking += strings.TrimSpace(extractedThought)
				turnText = strings.TrimSpace(turnText[:start] + turnText[end+8:])
			}
		}

		// --- Parsear tool calls del texto si el provider falló en parsearlos nativamente ---
		// extractToolCallsFromText valida estrictamente que la herramienta pertenezca a las registradas.
		if len(turnToolCalls) == 0 {
			turnToolCalls = extractToolCallsFromText(turnText)
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
			assistantMsg.Content = turnText
			// Si NO hubo llamadas a herramientas, este turno contiene la respuesta final al usuario.
			// Emitir los chunks de texto y acumular en finalContent.
			for _, chunkStr := range textChunks {
				emit(AgentEvent{Type: "message:delta", Content: chunkStr})
			}
			finalContent += turnText
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
				finalContent = "✓ He completado la acción solicitada."
				emit(AgentEvent{Type: "message:delta", Content: finalContent})
			}

			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent, Thinking: turnThinking})
			
			// --- Auto-Skill Evaluation ---
			if !params.VoiceMode && len(allToolCalls) > 0 {
				EvaluateAndSaveSkill(params.Provider, history, params.UserMessage)
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
			
			// --- Reproducir Voz Nativamente (Piper) ---
			if params.VoiceMode && finalContent != "" {
				go func(text string) {
					// Use relative path assuming backend is CWD
					pipe := voice.NewAudioPipeline(voice.Config{
						PiperBinary: "tools/piper/piper/piper.exe",
						PiperModel:  "tools/piper/es_ES-davefx-medium.onnx",
					})
					wav, err := pipe.SynthesizeSpeech(context.Background(), text)
					if err == nil && len(wav) > 0 {
						voice.PlayWAV(wav)
					} else {
						log.Printf("Error sintetizando voz: %v", err)
					}
				}(finalContent)
			}
			
			return
		}

		allToolCalls = append(allToolCalls, turnToolCalls...)

		// --- Ejecutar cada tool call y añadir resultados ---
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
	const maxPrev = 15
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

func BuildSystemPromptForTest(params AgentLoopParams) string {
	return buildAgentSystemPrompt(params)
}

func buildAgentSystemPrompt(params AgentLoopParams) string {
	userProfile := os.Getenv("USERPROFILE")
	username := os.Getenv("USERNAME")

	// VoiceMode: prompt ultra-corto para minimizar tokens de prefill y lograr respuestas rápidas.
	if params.VoiceMode {
		return fmt.Sprintf("Eres Ozy, asistente de voz para Windows. Usuario: %s (%s). Responde MUY BREVE (máx 2 oraciones). Ejecuta herramientas OS directamente. Responde siempre en español.", username, userProfile)
	}

	archSummary := ""
	if reg := system.DefaultPathRegistry(); reg != nil {
		archSummary = reg.GetSystemArchitectureSummary()
	}
	if archSummary == "" {
		archSummary = fmt.Sprintf("- Documentos: %s\\Documents (proyectos: crmgeofal, cotizador, Due Inmobiliari, landing ozybase7, ozyAsis, ozybase, OzyERP-World, Ozygram, ozyshield, Peru-flack, Portfolio, rmm, trabajosalinstante)", userProfile)
	}

	base := fmt.Sprintf(`Eres Ozy, el asistente y agente autónomo de sistema operativo de OzyAssist. Cuentas con control, visibilidad e integración nativa para operar directamente en el entorno de Windows del usuario.

ENTORNO WINDOWS DEL USUARIO Y ARQUITECTURA DE PROYECTOS:
- Usuario actual: %s
- Carpeta Personal: %s
- Descargas: %s\Downloads
- Escritorio: %s\Desktop
%s

USO DE HERRAMIENTAS DEL SISTEMA (CRÍTICO):
1. APERTURA DE APLICACIONES Y PROYECTOS:
   - Para abrir cualquier programa (ej: Antigravity IDE, VS Code, Chrome, Bloc de notas, Calculadora, Explorer, Docker, etc.): usa 'os_launch_app'.
   - Si el usuario pide abrir un proyecto o archivo con un editor (ej: "abre con antigravity ide el proyecto crmgeofal", "abre vscode en mi proyecto"): pasa 'appName': "antigravity" (o "code") y 'path': "crmgeofal" (o la ruta al proyecto). Ozy localizará automáticamente el ejecutable instalado y la ruta completa de la carpeta en el sistema.
   - NUNCA te limites a solo explorar los archivos con 'os_explore' si el usuario pidió explícitamente "abrir con [programa]". Debes invocar 'os_launch_app'.
2. GESTIÓN DE ARCHIVOS, EXCEL Y RUTAS:
   - Usa 'os_explore' para inspeccionar contenidos de carpetas, 'os_find_files' para buscar recursivamente.
   - Para crear carpetas o directorios: usa 'os_create_dir'.
   - Para crear o generar hojas de cálculo de Excel (.xlsx): DEBES USAR SIEMPRE 'os_create_excel' (NUNCA 'os_create_dir'). Pasa 'path' (ej: 'Desktop/reporte.xlsx'), 'headers' y 'rows'.
   - Para leer documentos (PDF, Excel .xlsx, Word .docx, CSV, texto plano): usa 'os_read_document'.
   - Usa 'os_move_item' para mover o renombrar, 'os_organize_folder' para clasificar automáticamente.
3. CONTROL DE VENTANAS, PROCESOS, HARDWARE Y SERVICIOS (CRÍTICO - ANTI-ALUCINACIÓN):
   - Usa 'os_active_windows' para ver qué está abierto en pantalla (te devolverá títulos, HWND y PID).
   - Usa 'os_focus_window' para traer al frente una ventana por su HWND.
   - Para ACOMODAR O DIVIDIR VENTANAS (ej: "pon el navegador a la izquierda", "maximiza", "centra", "muestra el escritorio"):
     * USA 'os_tile_windows' pasando 'title' y 'layout' ("left", "right", "maximize", "minimize", "restore", "center", "show_desktop").
   - Para CERRAR ventanas o programas (ej: "cierra el administrador de tareas", "cierra la calculadora", "cierra el bloc de notas", "cierra la ventana de X"):
     * USA PREFERENTEMENTE 'os_close_window' pasando 'title' (ej: "Administrador de tareas", "Calculadora") o el 'hwnd' obtenido con 'os_active_windows'.
     * O usa 'os_kill_process' indicando 'name' (ej: "Taskmgr.exe", "notepad.exe", "calc.exe") o 'pid'.
     * PROHIBICIÓN ESTRICTA ANTI-ALUCINACIÓN: NUNCA digas "He cerrado la ventana X" o "Cerré el programa" si no has ejecutado 'os_close_window' u 'os_kill_process' en ese turno exacto. Si listaste las ventanas con 'os_active_windows', DEBES ejecutar inmediatamente 'os_close_window' para efectuar el cierre real antes de responder al usuario.
   - Para administrar servicios de Windows: usa 'os_service_manager' ('list', 'status', 'start', 'stop', 'restart').
   - Para administrar Docker en el host: usa 'os_docker_manager' ('status', 'list', 'logs', 'start', 'stop', 'restart', 'stats').
   - Para analizar errores en logs o eventos de Windows: usa 'os_analyze_logs'.
   - Para inspeccionar puertos de red abiertos y procesos que escuchan (TCP/UDP): usa 'os_port_inspector'. ÚNICAMENTE para sockets lógicos de red. NUNCA para puertos físicos ni dispositivos USB.
   - Para evaluar la salud del hardware (SMART de discos, umbrales térmicos, espacio crítico en disco C, memoria y eventos WHEA), puertos físicos USB, cámara/mic en uso, y telemetría de CPU/RAM/GPU/Discos: usa 'os_hardware_inspector' ('health', 'devices', 'in_use', 'telemetry').
   - Para consultar o cambiar planes de energía de Windows (Equilibrado, Alto Rendimiento, Ahorro) y ajustar el brillo de pantalla en %%: usa 'os_power_profile'.
   - Para emitir notificaciones interactivas Toast nativas en Windows 10/11: usa 'os_toast_notify'.
   - Para diagnósticos de conectividad de red (ping, latencia en ms, IPs locales y vaciado de caché DNS flush): usa 'os_network_diagnostics'.
   - Para identificar archivos duplicados por hash SHA256 e instaladores huérfanos para liberar espacio: usa 'os_smart_organizer'.
   - Para analizar y limpiar espacio en disco (temporales, papelera): usa 'os_disk_cleaner'.
   - Para controlar el sistema de audio (consultar nivel de volumen %%, subir/bajar volumen, silenciar con mute, o listar/cambiar dispositivos de salida): usa 'os_audio_device'.
   - Para revisar el estado de Wi-Fi, intensidad de señal y redes: usa 'os_wifi_manager'.
   - Para programar tareas autónomas persistentes en segundo plano: usa 'os_schedule_task'.
4. INVESTIGACIÓN WEB Y SITIOS EN VIVO (SUPERPODERES):
   - Cuando el usuario mencione un dominio (ej: "peruflack.com"), URL o pregunte si un sitio web existe/está activo: usa 'web_fetch' directamente para inspeccionar en vivo el sitio (estado HTTP, metadatos, título, OpenGraph y texto Markdown).
   - Usa 'web_dns_lookup' para comprobar IPs, CNAME, MX y registros técnicos del dominio.
   - NUNCA concluyas que un sitio web "no existe" o "está inactivo" basándote solo en una búsqueda de DuckDuckGo vacía. SIEMPRE sondea el dominio directamente con 'web_fetch' y 'web_dns_lookup'.
   - Usa 'web_search' y 'deep_search' para buscar noticias, documentación, paquetes o soluciones técnicas.
5. CORREOS ELECTRÓNICOS Y GMAIL EN NAVEGADOR (CRÍTICO):
   - Cuando el usuario te pida redactar un correo, "ábrelo en mi navegador", "visualizarlo antes de enviarlo" o ver el correo desarrollado: DEBES USAR SIEMPRE 'os_draft_email'.
   - Especifica 'to', 'subject', 'body', 'from_account': "zastuto5@gmail.com" (o el perfil solicitado) y 'client': "gmail".
   - NUNCA uses 'os_launch_app' con URLs genéricas (como '#inbox') para correos. 'os_draft_email' es la única herramienta que abre la pestaña de composición de Gmail con todo el contenido ya escrito y enfoca la ventana para que el usuario solo tenga que revisarlo.
6. MENSAJERÍA INSTANTÁNEA (WHATSAPP Y TELEGRAM):
   - Cuando el usuario te pida redactar o enviar un mensaje por WhatsApp: DEBES USAR SIEMPRE 'os_draft_whatsapp'.
   - Si el usuario proporciona un número (ej: "987654321" o "+51987654321"), pásalo en 'phone'. Pasa el mensaje en 'text'. La herramienta abrirá la pestaña de WhatsApp Web con el chat y texto listos en el navegador del usuario.
   - Cuando el usuario te pida redactar o enviar un mensaje por Telegram: DEBES USAR 'os_draft_telegram'.
   - Pasa el usuario o canal en 'recipient' (ej: "@usuario" o nombre) y el contenido en 'text'.
7. ACCESO A TERMINAL, GIT Y PROYECTOS (CRÍTICO - SIEMPRE CON LA VERDAD):
   - Cuando el usuario pregunte por el historial de git, commits, ramas, cambios o comandos de OTRO proyecto (ej: "los de crmgeofal", "el proyecto cotizador", "en documentos/crmgeofal"):
     * DEBES pasar 'cwd': "crmgeofal" (o la ruta al proyecto) en 'os_run_command' o 'run_command'. El sistema resolverá automáticamente el directorio real en Documentos.
     * NUNCA ejecutes comandos en el directorio actual sin 'cwd' cuando la pregunta se refiera a otro proyecto, porque de lo contrario devolverías los commits de OzyAssist en lugar de la verdad del proyecto consultado.
     * Si el proyecto consultado (como crmgeofal) contiene múltiples repositorios o submódulos Git independientes (ej: api-geofal-crm, crm-geofal, Developmen-CRM), el sistema inspeccionará y te devolverá automáticamente los commits reales de cada uno.
     * NUNCA digas "no tengo acceso a Git". Puedes ejecutar 'git log', 'git status', 'dir', 'npm test', etc., directamente con 'os_run_command'.
8. OBJETIVOS COMPUESTOS Y EJECUCIÓN MULTI-PASO (CRÍTICO - REGLA DE CIERRE):
   - SIEMPRE que el usuario solicite más de una acción en su mensaje (ej: "investiga X, crea un reporte en Markdown y prepara un correo para Y", "busca A y redacta un WhatsApp a B"):
     * DEBES EJECUTAR TODAS LAS HERRAMIENTAS SOLICITADAS ANTES DE CONCLUIR.
     * Si la solicitud menciona "correo", "email", "Gmail" o una dirección de correo (ej: "sistemas@geofal.com.pe"): ES OBLIGATORIO invocar 'os_draft_email'. NO puedes terminar la conversación sólo presentando texto si el usuario pidió redactar o preparar un correo.
     * Si la solicitud menciona "WhatsApp" o un número telefónico: ES OBLIGATORIO invocar 'os_draft_whatsapp'.
     * Si la solicitud menciona "Telegram": ES OBLIGATORIO invocar 'os_draft_telegram'.
   - Plan secuencial estricto:
     Paso 1: Ejecutar la investigación ('deep_search' o 'web_search').
     Paso 2: Inmediatamente en el siguiente turno, tomar los hallazgos y llamar a 'os_draft_email' (u otra herramienta solicitada) con los datos investigados.
     Paso 3: Solo una vez ejecutadas TODAS las herramientas, emitir la confirmación final al usuario.
   - NO te detengas en el primer paso si quedan acciones por realizar. Continúa ejecutando las herramientas necesarias hasta completar todo el objetivo.
9. GENERACIÓN Y CREACIÓN DE HOJAS DE CÁLCULO EXCEL:
   - Cuando el usuario te pida crear o generar un archivo Excel, reporte en hoja de cálculo o tabla exportada (.xlsx): usa 'os_create_excel'.
   - Especifica 'path' (ej: 'Desktop/reporte_ventas.xlsx'), 'title', 'headers' (columnas) y 'rows' (datos).
   - Ozy creará un archivo Excel nativo con formato profesional y encabezados en verde neón (#D1F107).
10. CONSULTAS A BASES DE DATOS SQL LOCALES (SQL EXPLORER):
   - Cuando el usuario pida consultar o auditar una base de datos local SQLite (.db o .sqlite de proyectos como crmgeofal u ozyassist): usa 'os_query_db'.
   - Pasa 'db_path' y la sentencia 'query' (SELECT o PRAGMA). Solo ejecuta consultas de lectura seguras.
11. VISIÓN Y DIAGNÓSTICO VISUAL DE PANTALLA:
   - Si el usuario te pregunta sobre lo que hay en su pantalla ("¿qué error se ve?", "describe mi pantalla", "mira este gráfico o ventana"): usa 'os_analyze_screen'.
   - Ozy tomará la captura y realizará un diagnóstico visual inteligente con IA multimodal.
12. MONITOREO PROACTIVO DE FONDO (WATCHDOG):
   - Para vigilar si un puerto local (ej: '8080', '3000', '5432') o un proceso (ej: 'docker.exe') deja de responder: usa 'os_watchdog' con 'action': "start".
   - Ozy alertará proactivamente al usuario con notificaciones nativas Toast si el servicio cae.
13. FORMATO ESTRICTO DE HERRAMIENTAS:
   - DEBES usar SIEMPRE la invocación nativa de funciones (Tool Calling API). NUNCA escribas bloques de código Markdown con JSON (ej: ` + "```json" + `) para ejecutar herramientas.
   - Si debes ejecutar algo, llama a la herramienta directamente en tu respuesta.
14. AUTOMATIZACIÓN CREATIVA (PYTHON/POWERSHELL):
   - Si el usuario te pide modificar un archivo complejo (como un Excel .xlsx, un PDF) o realizar una tarea para la cual NO tienes una herramienta nativa específica, SÉ CREATIVO: usa 'write_file' para crear un script en Python (ej: script.py con pandas u openpyxl) y luego usa 'os_run_command' para instalar dependencias y ejecutarlo. ¡Tú eres un ingeniero completo!
15. FÁBRICA DE HERRAMIENTAS REUTILIZABLES (~/.ozy/tools):
   - Si creas un script útil de automatización, guárdalo permanentemente usando 'os_save_custom_tool' para que esté disponible para futuras sesiones.
16. MESA DE TRABAJO SEGURA Y AUTO-BACKUP (PROTECCIÓN TOTAL):
   - Si vas a transformar o editar un archivo existente importante (ej: Excel .xlsx, bases de datos, código):
     a) Usa 'os_prepare_staging' para copiarlo a tu mesa de trabajo (~/.ozy/workspace/) con backup automático previo.
     b) Ejecuta tus scripts sobre la copia en la mesa de trabajo sin tocar el original.
      c) Solo cuando verifiques que el resultado es exitoso y no está corrupto, usa 'os_commit_staging' para aplicar los cambios atómicamente.
18. GENERACIÓN Y CONVERSIÓN PROFESIONAL DE DOCUMENTOS PDF (NATIVO):
   - Cuando el usuario te pida crear o generar un informe o documento en PDF (.pdf): DEBES USAR SIEMPRE 'os_create_pdf'.
   - PROHIBICIÓN ESTRICTA: NUNCA renombres un archivo .xlsx, .csv, .txt o .docx cambiándole la extensión a .pdf (eso creará un archivo corrupto que fallará al abrirse en Adobe Acrobat y visores del sistema).
   - 'os_create_pdf' genera un archivo PDF nativo (PDF-1.3) con barra de acento institucional, metadatos, secciones estructuradas con viñetas y tablas elegantes con zebra-striping.
   - Si ya existe un archivo de texto, Markdown o CSV y el usuario quiere un PDF, usa 'os_convert_to_pdf'.
19. GENERACIÓN DE DOCUMENTOS MICROSOFT WORD (.DOCX) NATIVO:
   - Cuando el usuario te pida redactar un informe, contrato o documento en Word (.docx): DEBES USAR SIEMPRE 'os_create_docx'.
   - NUNCA renombres un archivo .txt o .md a .docx. 'os_create_docx' genera un archivo OpenXML estándar con encabezados, párrafos, viñetas y tablas estilizadas compatible con Microsoft Word, Office 365, LibreOffice y Google Docs.
20. GESTIÓN DE ARCHIVOS COMPRIMIDOS (.ZIP):
   - Para empaquetar archivos o carpetas a formato .zip: usa 'os_compress_zip'.
   - Para extraer o descomprimir un archivo .zip: usa 'os_extract_zip'.
21. BÚSQUEDA RÁPIDA DE CONTENIDO EN ARCHIVOS (GREP NATIVO):
   - Para buscar texto, claves de configuración, variables o fragmentos de código dentro de los archivos de una carpeta o proyecto: usa 'os_search_content'. Es instantáneo, multi-hilo y omite carpetas masivas (node_modules, .git, venv).
22. CONTROL E INSPECCIÓN DE PUERTOS DE RED VS PUERTOS FÍSICOS Y HARDWARE:
   - Para averiguar qué proceso (PID y ejecutable) tiene ocupado un puerto de red (ej: 3000, 8080, 5432) o listar los puertos TCP/UDP en escucha: usa 'os_port_inspector'. NUNCA uses 'os_port_inspector' si la pregunta es sobre puertos físicos o USB.
   - Para puertos físicos USB, dispositivos conectados, periféricos en uso (cámara web, micrófono) y telemetría de hardware (CPU, RAM, Discos, GPU, temperatura, batería): usa 'os_hardware_inspector'.
23. DESCARGA DIRECTA DE ARCHIVOS WEB:
   - Para descargar archivos, imágenes, PDFs o paquetes desde una URL HTTP/HTTPS directo al disco: usa 'os_download_file'.
24. AUDITORÍA COGNITIVA CHARC Y DETECCIÓN ESTRICTA DE ERRORES EN APLICACIONES (ANTI-ALUCINACIÓN):
   - Cuando abras un archivo o lances un programa (como Adobe Acrobat, Excel, Word, etc.) y el usuario te pregunte "¿dio algún error?", "¿se abrió bien?" o "¿qué error muestra?":
     * PROHIBICIÓN ABSOLUTA DE ASUMIR O ALUCINAR ÉXITO: NUNCA afirmes que "no hay ningún error" o que "abrió correctamente" sin haber comprobado el estado real del sistema.
     * DEBES invocar INMEDIATAMENTE 'os_detect_dialogs' (pasando opcionalmente el filtro de la aplicación, ej: 'acrobat', 'adobe', 'excel').
     * Si 'os_detect_dialogs' detecta un diálogo modal de error (IsError: true o Severity: ERROR), DEBES reportar al usuario el texto literal exacto del mensaje de error detectado (ej: "Adobe Acrobat Reader no pudo abrir el archivo debido a que no es un tipo de archivo admitido o está dañado").
     * Si requieres inspección visual complementaria de la interfaz, usa 'os_analyze_screen'.
     * La regla fundamental de OzyAssist es la VERDAD y la SENSIBILIDAD A ERRORES: reporta la realidad exacta de lo que ocurre en pantalla y ejecuta la solución correspondiente.
25. PROHIBICIÓN ABSOLUTA DE ATAJOS FRAUDULENTOS:
   - NUNCA intentes cambiar el tipo o formato de un archivo simplemente cambiándole la extensión (ej: de .xlsx a .pdf o de .txt a .docx).
   - Usa siempre la herramienta nativa específica correspondiente ('os_create_pdf', 'os_create_docx', 'os_create_excel').
   - Si se requiere un formato para el cual NO existe una herramienta nativa, crea un script ejecutable en Python o PowerShell en tu mesa de trabajo (~/.ozy/workspace/) que utilice las librerías apropiadas para procesar el formato real.
26. TRÍADA COGNITIVA Y SUBAGENTES INTEGRADOS (NATIVOS):
   - Cuentas con subagentes nativos especializados trabajando en armonía bajo tu misma arquitectura cognitiva:
     * CHARC (Auditor de Seguridad y Supervisor de Bucles): Evalúa riesgos antes de ejecutar acciones en el sistema operativo, previene bucles repetitivos y autoriza cambios críticos.
     * NINE (Estratega de Razonamiento Profundo): Diseña planes alternativos y descompone metas multi-etapa complejas cuando una tarea encuentra bloqueos.
     * DREAMER (Consolidación Cognitiva y Memoria Continua): Subagente asíncrono que sintetiza hechos atómicos aprendidos, resuelve discrepancias y mantiene al día tu perfil de usuario.
   - Si el usuario te pregunta "¿qué subagentes tienes?" o por tu arquitectura interna: EXPLICA CON CLARIDAD TU IDENTIDAD (Ozy: asistente ejecutor central de SO), y la función especializada de tus subagentes nativos CHARC, NINE y DREAMER.`,
		username, userProfile, userProfile, userProfile, archSummary)



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

	// CONTEXT MODE INICIAL
	if !params.VoiceMode {
		ctxWin, cancelWin := context.WithTimeout(context.Background(), 1*time.Second)
		windowsCtx, _ := execOSActiveWindows(ctxWin)
		cancelWin()

		ctxClip, cancelClip := context.WithTimeout(context.Background(), 1*time.Second)
		clipCtx, _ := execOSGetClipboard(ctxClip)
		cancelClip()
		
		var sb strings.Builder
		sb.WriteString("\n\n=== CONTEXTO ACTUAL DE LA PC (TIEMPO REAL) ===\n")
		sb.WriteString("VENTANAS ACTIVAS EN PANTALLA:\n")
		if windowsCtx != "" {
			sb.WriteString(windowsCtx)
		} else {
			sb.WriteString("Ninguna visible.")
		}
		
		sb.WriteString("\n\nPORTAPAPELES ACTUAL:\n")
		if clipCtx != "" && len(clipCtx) < 1000 {
			sb.WriteString(clipCtx)
		} else if len(clipCtx) >= 1000 {
			sb.WriteString(clipCtx[:1000] + "... (recortado)")
		} else {
			sb.WriteString("(Vacío)")
		}
		
		base += sb.String()
		
		// Inyectar AutoSkills aprendidos
		if skillsCtx := LoadAutoSkills(); skillsCtx != "" {
			base += skillsCtx
		}

		// Inyectar Herramientas de la Fábrica (~/.ozy/tools)
		if toolsCtx := LoadCustomTools(); toolsCtx != "" {
			base += toolsCtx
		}
	}

	if !params.VoiceMode {
		mcpTools := mcp.DefaultRegistry.GetAllTools()
		if len(mcpTools) > 0 {
			var sb strings.Builder
			sb.WriteString("\n\nHERRAMIENTAS EXTERNAS Y CONECTORES MCP DISPONIBLES:\n")
			for name, t := range mcpTools {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", name, t.Description))
			}
			base += sb.String()
		}

		// Inyección de Perfil de Usuario Persistente
		userForProfile := params.UserID
		if userForProfile == "" && params.Chat != nil {
			userForProfile = params.Chat.UserID
		}
		if userForProfile == "" && db.DB != nil {
			userForProfile = db.DefaultUserID()
		}

		if db.DB != nil && userForProfile != "" {
			ctxCache := context.Background()
			cachedProfile, found, _ := memory.DefaultCache().Get(ctxCache, "user_profile:"+userForProfile)
			if found && strings.TrimSpace(cachedProfile) != "" {
				base += fmt.Sprintf("\n\n=== PERFIL Y ROL DEL USUARIO ===\n%s\nAdapta tus respuestas, tono, nivel técnico y decisiones a este perfil.", strings.TrimSpace(cachedProfile))
			} else if u, err := db.GetUser(userForProfile); err == nil && u != nil && strings.TrimSpace(u.ProfileMd) != "" {
				_ = memory.DefaultCache().Set(ctxCache, "user_profile:"+userForProfile, u.ProfileMd, 1*time.Hour)
				base += fmt.Sprintf("\n\n=== PERFIL Y ROL DEL USUARIO ===\n%s\nAdapta tus respuestas, tono, nivel técnico y decisiones a este perfil.", strings.TrimSpace(u.ProfileMd))
			}
		}

		// Inyección de Recuerdos Persistentes Relevantes (FTS5 / user_memories)
		store := memory.DefaultStore()
		if store == nil && db.DB != nil {
			store = memory.NewStore(db.DB)
		}
		if store != nil && len(strings.TrimSpace(params.UserMessage)) > 2 {
			ctxMem, cancelMem := context.WithTimeout(context.Background(), 800*time.Millisecond)
			facts, err := store.SearchRelevant(ctxMem, userForProfile, params.UserMessage, 5)
			cancelMem()
			if err == nil && len(facts) > 0 {
				var sbMem strings.Builder
				sbMem.WriteString("\n\n=== RECUERDOS Y PREFERENCIAS APRENDIDAS DEL USUARIO ===\n")
				for i, f := range facts {
					sbMem.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, strings.ToUpper(string(f.Category)), f.Content))
				}
				sbMem.WriteString("Ten en cuenta estos hechos aprendidos para actuar con precisión y no volver a preguntar lo que ya sabes.")
				base += sbMem.String()
			}
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

func isRegisteredTool(name string) bool {
	if strings.HasPrefix(name, "mcp_") {
		return true
	}
	for _, t := range AgentTools {
		if t.Name == name {
			return true
		}
	}
	for _, t := range VoiceAgentTools {
		if t.Name == name {
			return true
		}
	}
	return false
}

func extractToolCallsFromText(text string) []providers.ToolCall {
	var calls []providers.ToolCall
	
	// Patrón 1: <tool_call>{"name": "xyz", "arguments": {}}</tool_call>
	toolCallRe := regexp.MustCompile(`(?is)<tool_call>\s*({.*?})\s*</tool_call>`)
	matches := toolCallRe.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		var tc struct {
			Name      string `json:"name"`
			Arguments any    `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(match[1]), &tc); err == nil && isRegisteredTool(tc.Name) {
			argBytes, _ := json.Marshal(tc.Arguments)
			calls = append(calls, providers.ToolCall{
				ID:    uuid.NewString(),
				Name:  tc.Name,
				Input: argBytes,
			})
		}
	}
	
	// Patrón 2: <tool name="xyz" arguments="{}"></tool> o sin cierre
	xmlRe := regexp.MustCompile(`(?is)<tool\s+name="([^"]+)"\s+arguments='([^']+)'`)
	xmlMatches := xmlRe.FindAllStringSubmatch(text, -1)
	if len(xmlMatches) == 0 {
		xmlRe = regexp.MustCompile(`(?is)<tool\s+name="([^"]+)"\s+arguments="([^"]+)"`)
		xmlMatches = xmlRe.FindAllStringSubmatch(text, -1)
	}
	for _, match := range xmlMatches {
		if isRegisteredTool(match[1]) {
			calls = append(calls, providers.ToolCall{
				ID:    uuid.NewString(),
				Name:  match[1],
				Input: []byte(match[2]),
			})
		}
	}
	
	// Patrón 3: Llama 3.1 <HOLDER>{call_function{"name": "xyz", "arguments": {}}}</HOLDER>
	holderRe := regexp.MustCompile(`(?is)<HOLDER>\s*\{call_function\s*({.*?})\s*\}\s*</HOLDER>`)
	holderMatches := holderRe.FindAllStringSubmatch(text, -1)
	for _, match := range holderMatches {
		var tc struct {
			Name      string `json:"name"`
			Arguments any    `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(match[1]), &tc); err == nil && isRegisteredTool(tc.Name) {
			argBytes, _ := json.Marshal(tc.Arguments)
			calls = append(calls, providers.ToolCall{
				ID:    uuid.NewString(),
				Name:  tc.Name,
				Input: argBytes,
			})
		}
	}
	
	// Patrón 4: JSON multilínea con "name" o "call" y "arguments"
	jsonRe := regexp.MustCompile(`(?s)\{\s*"(?:name|call)"\s*:\s*"([^"]+)"\s*,\s*"arguments"\s*:\s*(null|\{.*?\})\s*\}`)
	jsonMatches := jsonRe.FindAllStringSubmatch(text, -1)
	for _, match := range jsonMatches {
		name := match[1]
		if !isRegisteredTool(name) {
			continue
		}
		args := match[2]
		exists := false
		for _, c := range calls {
			if c.Name == name {
				exists = true
				break
			}
		}
		if !exists {
			calls = append(calls, providers.ToolCall{
				ID:    uuid.NewString(),
				Name:  name,
				Input: []byte(args),
			})
		}
	}
	
	return calls
}

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

// checkPendingTaskRequirements evalúa si el objetivo original del usuario contiene acciones explícitas
// (ej: redactar correo, enviar WhatsApp/Telegram, notificar) que aún no hayan sido invocadas por el LLM.
func checkPendingTaskRequirements(userMessage string, executedCalls []providers.ToolCall) string {
	executed := make(map[string]bool)
	for _, tc := range executedCalls {
		executed[tc.Name] = true
	}

	lower := strings.TrimSpace(strings.ToLower(userMessage))

	// No exigir herramientas para preguntas informativas o conceptuales
	if strings.HasPrefix(lower, "cómo") || strings.HasPrefix(lower, "como") ||
		strings.HasPrefix(lower, "qué") || strings.HasPrefix(lower, "que") ||
		strings.HasPrefix(lower, "cuál") || strings.HasPrefix(lower, "cual") ||
		strings.HasPrefix(lower, "explica") || strings.HasPrefix(lower, "dime") {
		return ""
	}

	hasActionVerb := strings.Contains(lower, "redacta") || strings.Contains(lower, "envía") || strings.Contains(lower, "envia") || strings.Contains(lower, "prepara") || strings.Contains(lower, "escribe")

	// 1. Verificación de Correo Electrónico (solo con destinatario explícito o verbo de acción)
	emailAddr := emailRegex.FindString(userMessage)
	hasEmailIntent := (strings.Contains(lower, "correo") || strings.Contains(lower, "email") || strings.Contains(lower, "gmail")) && hasActionVerb
	if (emailAddr != "" || hasEmailIntent) && !executed["os_draft_email"] {
		recipient := emailAddr
		if recipient == "" {
			recipient = "el destinatario solicitado en tu objetivo"
		}
		return fmt.Sprintf("[SISTEMA AUTÓNOMO - MOTOR DE TAREAS]: Has completado la primera etapa del objetivo. Ahora ejecuta inmediatamente el siguiente paso pendiente: redacta el resumen y abre el borrador de correo con 'os_draft_email' para %s.", recipient)
	}

	// 2. Verificación de WhatsApp (solo con verbo de acción)
	hasWhatsAppIntent := (strings.Contains(lower, "whatsapp") || strings.Contains(lower, "wasap")) && hasActionVerb
	if hasWhatsAppIntent && !executed["os_draft_whatsapp"] {
		return "[SISTEMA AUTÓNOMO - MOTOR DE TAREAS]: Ejecuta ahora el siguiente paso pendiente del plan: prepara y abre el mensaje en WhatsApp Web con 'os_draft_whatsapp'."
	}

	// 3. Verificación de Telegram (solo con verbo de acción)
	hasTelegramIntent := strings.Contains(lower, "telegram") && hasActionVerb
	if hasTelegramIntent && !executed["os_draft_telegram"] && !executed["telegram_send_message"] {
		return "[SISTEMA AUTÓNOMO - MOTOR DE TAREAS]: Ejecuta ahora el siguiente paso pendiente del plan: prepara el mensaje de Telegram usando 'os_draft_telegram'."
	}

	// 4. Verificación de Notificación de Escritorio
	hasNotifyIntent := strings.Contains(lower, "notifica") || strings.Contains(lower, "notifícame") || strings.Contains(lower, "avísame al finalizar")
	if hasNotifyIntent && !executed["os_notify"] && len(executedCalls) > 0 {
		return "[SISTEMA AUTÓNOMO - MOTOR DE TAREAS]: Las acciones principales han finalizado. Envía ahora la notificación de escritorio al usuario con 'os_notify'."
	}

	return ""
}

