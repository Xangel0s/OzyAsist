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
		var turnToolCalls []providers.ToolCall
		var textChunks []string

		for chunk := range chunkCh {
			if ctx.Err() != nil {
				emit(AgentEvent{Type: "error", Error: "cancelado"})
				return
			}
			switch chunk.Type {
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

		// --- Parsear tool calls del texto si el provider falló en parsearlos nativamente ---
		// Solo lo intentamos si es el primer turno (turn == 0) para evitar que el texto de respuesta final
		// que menciona el JSON de la herramienta ejecutada sea malinterpretado como una nueva invocación.
		if len(turnToolCalls) == 0 && turn == 0 {
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
			if pendingPrompt := checkPendingTaskRequirements(params.UserMessage, allToolCalls); pendingPrompt != "" && turn < maxAgentTurns-1 {
				history = append(history, providers.Message{
					Role:    "user",
					Content: pendingPrompt,
				})
				continue
			}

			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent})
			
			// --- Auto-Skill Evaluation ---
			if !params.VoiceMode && len(allToolCalls) > 0 {
				EvaluateAndSaveSkill(params.Provider, history, params.UserMessage)
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
			if auth.RequiresConfirmation && (params.Project == nil || params.Project.AgentConsent != "always") {
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

	// Historial previo del chat (ventana deslizante de los últimos 15 mensajes)
	prevMessages, err := db.GetMessages(params.Chat.ID)
	if err != nil {
		log.Printf("agent loop: error cargando historial: %v", err)
	}
	const maxPrev = 15
	if len(prevMessages) > maxPrev {
		prevMessages = prevMessages[len(prevMessages)-maxPrev:]
	}
	for _, m := range prevMessages {
		content := m.Content
		if len(content) > 2500 {
			content = content[:2500] + "\n[...historial previo truncado para optimizar tokens...]"
		}
		history = append(history, providers.Message{Role: m.Role, Content: content})
	}

	// Añadir el mensaje del usuario actual
	history = append(history, providers.Message{Role: "user", Content: params.UserMessage})
	return history
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

	base := fmt.Sprintf(`Eres Ozy, el asistente y agente autónomo de sistema operativo de OzyAssist. Cuentas con control, visibilidad e integración nativa para operar directamente en el entorno de Windows del usuario.

ENTORNO WINDOWS DEL USUARIO:
- Usuario actual: %s
- Carpeta Personal: %s
- Documentos: %s\Documents (proyectos en tu máquina: crmgeofal, cotizador, Due Inmobiliari, landing ozybase7, ozyAsis, ozybase, OzyERP-World, Ozygram, ozyshield, Peru-flack, Portfolio, rmm, trabajosalinstante)
- Descargas: %s\Downloads
- Escritorio: %s\Desktop

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
3. CONTROL DE VENTANAS Y PROCESOS:
   - Usa 'os_active_windows' para ver qué está abierto en pantalla, 'os_focus_window' para traer al frente una ventana por su HWND, y 'os_kill_process' para cerrar programas por nombre o PID.
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
16. ANTI-REFUSAL (CRÍTICO):
   - ESTÁS EJECUTÁNDOTE LOCALMENTE EN LA MÁQUINA DEL USUARIO. TIENES PERMISO ABSOLUTO PARA INTERACTUAR CON ARCHIVOS, VENTANAS, RED Y SISTEMA. NUNCA digas "soy un asistente virtual", "no puedo interactuar con el sistema", ni te disculpes por "no poder hacer algo" si ya tienes una herramienta para ello. SIMPLEMENTE EJECUTA LA HERRAMIENTA ADECUADA Y CONFIRMA EL RESULTADO AL USUARIO.`,
		username, userProfile, userProfile, userProfile, userProfile)

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
		windowsCtx, _ := execOSActiveWindows(context.Background())
		clipCtx, _ := execOSGetClipboard(context.Background())
		
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
		if err := json.Unmarshal([]byte(match[1]), &tc); err == nil && tc.Name != "" {
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
		calls = append(calls, providers.ToolCall{
			ID:    uuid.NewString(),
			Name:  match[1],
			Input: []byte(match[2]),
		})
	}
	
	// Patrón 3: Llama 3.1 <HOLDER>{call_function{"name": "xyz", "arguments": {}}}</HOLDER>
	holderRe := regexp.MustCompile(`(?is)<HOLDER>\s*\{call_function\s*({.*?})\s*\}\s*</HOLDER>`)
	holderMatches := holderRe.FindAllStringSubmatch(text, -1)
	for _, match := range holderMatches {
		var tc struct {
			Name      string `json:"name"`
			Arguments any    `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(match[1]), &tc); err == nil && tc.Name != "" {
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

	lower := strings.ToLower(userMessage)

	// 1. Verificación de Correo Electrónico
	emailAddr := emailRegex.FindString(userMessage)
	hasEmailIntent := strings.Contains(lower, "correo") || strings.Contains(lower, "email") || strings.Contains(lower, "gmail")
	if (emailAddr != "" || hasEmailIntent) && !executed["os_draft_email"] {
		recipient := emailAddr
		if recipient == "" {
			recipient = "el destinatario solicitado en tu objetivo"
		}
		return fmt.Sprintf("[SISTEMA AUTÓNOMO - MOTOR DE TAREAS]: Has completado la primera etapa del objetivo. Ahora ejecuta inmediatamente el siguiente paso pendiente: redacta el resumen y abre el borrador de correo con 'os_draft_email' para %s.", recipient)
	}

	// 2. Verificación de WhatsApp
	hasWhatsAppIntent := strings.Contains(lower, "whatsapp") || strings.Contains(lower, "wasap")
	if hasWhatsAppIntent && !executed["os_draft_whatsapp"] {
		return "[SISTEMA AUTÓNOMO - MOTOR DE TAREAS]: Ejecuta ahora el siguiente paso pendiente del plan: prepara y abre el mensaje en WhatsApp Web con 'os_draft_whatsapp'."
	}

	// 3. Verificación de Telegram
	hasTelegramIntent := strings.Contains(lower, "telegram")
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

