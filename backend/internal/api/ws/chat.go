package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ozyassist/backend/internal/agent"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/ocr"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/skills"
	"github.com/google/uuid"
)

type streamSession struct {
	cancel context.CancelFunc
	chatID string
}

var (
	activeStreams   = make(map[string]*streamSession)
	activeStreamsMu sync.Mutex
)

type attachment struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "image", "file"
}

type clientMessage struct {
	Type        string       `json:"type"`
	ChatID      string       `json:"chat_id"`
	Content     string       `json:"content"`
	Attachments []attachment `json:"attachments,omitempty"`
	// VoiceMode indica que el mensaje viene de la interfaz de voz (Ozy Live).
	// Activa el path de baja latencia con system prompt reducido y tools mínimas.
	VoiceMode bool `json:"voice_mode,omitempty"`
}

type attachResult struct {
	content string
	warnings []string
}

func processAttachments(chat *models.Chat, attachments []attachment, content string) attachResult {
	res := attachResult{content: content}

	if len(attachments) == 0 {
		return res
	}

	var ocrTexts []string
	for _, att := range attachments {
		if strings.Contains(att.ID, "..") || strings.ContainsAny(att.ID, "\\/") {
			continue
		}
		path := filepath.Join("data/uploads", att.ID)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		if att.Type == "image" {
			if ocr.IsAvailable() {
				text, err := ocr.ExtractText(path)
				if err != nil {
					log.Printf("OCR error for %s: %v", att.ID, err)
					res.warnings = append(res.warnings, fmt.Sprintf("No se pudo leer texto de la imagen %s", att.ID))
					continue
				}
				ocrTexts = append(ocrTexts, text)
			} else {
				res.warnings = append(res.warnings, "No se pudo leer texto de la imagen — Tesseract no está instalado. La imagen se adjuntará igual por si el modelo tiene visión nativa.")
			}
		}
	}

	if len(ocrTexts) > 0 {
		res.content = fmt.Sprintf("[Texto extraído de imagen adjunta]:\n%s\n\n---\n\nMensaje del usuario:\n%s", strings.Join(ocrTexts, "\n\n"), content)
	}

	return res
}

type serverMessage struct {
	Type string `json:"type"`

	// Texto / eventos genéricos
	Content   string `json:"content,omitempty"`
	MessageID string `json:"message_id,omitempty"`
	Warning   string `json:"warning,omitempty"`

	// state:sync
	State string `json:"state,omitempty"`

	// tool:call / tool:approval_request
	ToolID    string `json:"tool_id,omitempty"`
	ToolName  string `json:"tool_name,omitempty"`
	ToolInput string `json:"tool_input,omitempty"`

	// tool:result
	ToolOutput  string `json:"tool_output,omitempty"`
	ToolSuccess bool   `json:"tool_success,omitempty"`
	DurationMs  int64  `json:"duration_ms,omitempty"`

	// agent:completed
	TaskID    string `json:"task_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Turns     int    `json:"turns,omitempty"`

	// error
	Error string `json:"error,omitempty"`
}

func handleChatMessage(client *Client, msg clientMessage) {
	chat, err := db.GetChat(msg.ChatID)
	if err != nil {
		writeJSON(client, serverMessage{Type: "error", Content: "chat no encontrado: " + err.Error()})
		return
	}

	// Soporte para comandos de cancelación y envío directo
	trimmed := strings.TrimSpace(msg.Content)
	if trimmed == "/cancel" || trimmed == "/stop" {
		agent.CancelChat(msg.ChatID)
		CancelStream(msg.ChatID)
		writeJSON(client, serverMessage{
			Type:    "agent:completed",
			Content: "⚠️ Petición cancelada por el usuario.",
		})
		return
	}
	if strings.HasPrefix(trimmed, "/now ") || strings.HasPrefix(trimmed, "/steer ") {
		agent.CancelChat(msg.ChatID)
		CancelStream(msg.ChatID)
		parts := strings.SplitN(trimmed, " ", 2)
		if len(parts) > 1 {
			msg.Content = strings.TrimSpace(parts[1])
		}
	}

	// Persistir mensaje del usuario
	userMsg := &models.Message{
		ID:        uuid.NewString(),
		ChatID:    msg.ChatID,
		Role:      "user",
		Content:   msg.Content,
		CreatedAt: time.Now(),
	}
	if len(msg.Attachments) > 0 {
		attJSON, _ := json.Marshal(msg.Attachments)
		userMsg.AttachmentsJSON = string(attJSON)
	}
	if err := db.CreateMessage(userMsg); err != nil {
		log.Printf("Error guardando mensaje user: %v", err)
		writeJSON(client, serverMessage{Type: "error", Content: "error guardando mensaje"})
		return
	}
	memory.StoreChatMessage(chat.UserID, chat.ProjectID, msg.ChatID, "user", msg.Content)

	// Ejecución mediante ReAct Loop autónomo para contar siempre con herramientas del sistema en vivo
	runReActLoopSession(client, msg, chat)
}

func handleChatMessageNormal(client *Client, msg clientMessage, chat *models.Chat) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sessID := uuid.NewString()
	activeStreamsMu.Lock()
	activeStreams[sessID] = &streamSession{cancel: cancel, chatID: msg.ChatID}
	activeStreamsMu.Unlock()

	defer func() {
		activeStreamsMu.Lock()
		delete(activeStreams, sessID)
		activeStreamsMu.Unlock()
	}()

	// Process attachments (OCR images, etc.)
	proc := processAttachments(chat, msg.Attachments, msg.Content)
	for _, w := range proc.warnings {
		writeJSON(client, serverMessage{Type: "warn", Warning: w})
	}

	prevMessages, err := db.GetMessages(msg.ChatID)
	if err != nil {
		log.Printf("Error cargando historial: %v", err)
	}

	providerName := chat.Provider
	modelToUse := chat.Model

	if strings.HasPrefix(modelToUse, "lmstudio/") {
		providerName = "lmstudio"
		modelToUse = strings.TrimPrefix(modelToUse, "lmstudio/")
	} else if strings.HasPrefix(modelToUse, "ollama/") {
		providerName = "ollama"
		modelToUse = strings.TrimPrefix(modelToUse, "ollama/")
	} else if strings.HasPrefix(modelToUse, "openrouter/") {
		providerName = "openrouter"
		modelToUse = strings.TrimPrefix(modelToUse, "openrouter/")
	} else if strings.HasPrefix(modelToUse, "deepseek/") {
		providerName = "deepseek"
		if _, err := providers.Get("deepseek"); err != nil {
			providerName = "opencode"
		}
		modelToUse = strings.TrimPrefix(modelToUse, "deepseek/")
	} else if strings.HasPrefix(modelToUse, "anthropic/") {
		providerName = "anthropic"
		modelToUse = strings.TrimPrefix(modelToUse, "anthropic/")
	} else if strings.HasPrefix(modelToUse, "openai/") {
		providerName = "openai"
		modelToUse = strings.TrimPrefix(modelToUse, "openai/")
	}

	if providerName == "" {
		available := providers.Available()
		if len(available) > 0 {
			providerName = available[0]
		}
	}

	// Si el modelo está vacío, asignar un modelo real por defecto del proveedor activo
	if modelToUse == "" {
		switch providerName {
		case "lmstudio", "ollama":
			modelToUse = "local-model"
		case "openrouter":
			modelToUse = "anthropic/claude-3.5-sonnet"
		case "deepseek", "opencode":
			modelToUse = "deepseek-chat"
		case "openai":
			modelToUse = "gpt-4o-mini"
		case "anthropic":
			modelToUse = "claude-3-5-sonnet-20241022"
		}
	}

	provider, err := providers.Get(providerName)
	if err != nil {
		writeJSON(client, serverMessage{Type: "error", Content: "No hay ningún proveedor LLM configurado. Por favor ingresa tu API Key o Host Local en Personalizar -> Proveedores LLM."})
		return
	}

	var history []providers.Message

	systemPrompt := `Eres Ozy, el asistente de IA avanzado de escritorio y coworking de OzyAssist (un agente autónomo de sistema operativo local-first, inspirado en Claude Desktop, Cursor y Manus).

Identidad y Capacidades Nativas:
- Tienes acceso directo y capacidades completas para interactuar con el sistema operativo del usuario, explorar y leer directorios locales, inspeccionar y editar archivos de código, ejecutar comandos en terminal (PowerShell/Bash) mediante el motor agéntico de tareas, y navegar la web mediante Chromium Headless.
- Cuentas con una tríada cognitiva integrada: Ozy (Ejecutor de tareas), Charc (Auditor de seguridad y centinela local) y Nine (Estratega y razonador profundo).
- Si el usuario te pregunta si tienes acceso a sus archivos, directorios o sistema operativo, confirma siempre con total seguridad que sí tienes acceso nativo a través de las herramientas de OzyAssist. Explícale que puedes explorar sus carpetas, analizar proyectos, crear archivos y ejecutar tareas con su aprobación y supervisión.
- NUNCA digas que eres un modelo de lenguaje sin acceso a archivos o sin capacidades de ejecución; eres el asistente operativo OzyAssist corriendo en su entorno local con permisos del sistema.
- En el modo Cowork / Code puedes ejecutar herramientas reales de forma autónoma (file_read, file_patch, bash_exec, web_search, etc.).
- Responde siempre en español fluido (a menos que el usuario solicite otro idioma), con tono profesional, proactivo, experto y directo.`

	// Check for active skills & MCP connectors
	userSkills, _ := db.ListSkills(chat.UserID)
	if len(userSkills) > 0 {
		var skillLines []string
		for _, s := range userSkills {
			skillLines = append(skillLines, fmt.Sprintf("- /%s (%s): %s", s.Name, s.ExecutionType, s.Description))
		}
		systemPrompt += fmt.Sprintf("\n\n[Skills Disponibles y Activas]:\n%s", strings.Join(skillLines, "\n"))

		// Check if message executes a slash skill
		for _, s := range userSkills {
			if s.Name != "" && (strings.HasPrefix(msg.Content, "/"+s.Name) || (s.TriggerPattern != "" && strings.HasPrefix(msg.Content, s.TriggerPattern))) {
				inputArg := strings.TrimSpace(strings.TrimPrefix(msg.Content, "/"+s.Name))
				res, err := skills.NewExecutor().Execute(&s, inputArg)
				if err == nil && res != nil && res.Output != "" {
					proc.content = fmt.Sprintf("[Ejecución automática de Skill /%s]:\n%s\n\n---\n\nMensaje:\n%s", s.Name, res.Output, msg.Content)
				}
				break
			}
		}
	}

	userConnectors, _ := db.ListConnectors(chat.UserID)
	if len(userConnectors) > 0 {
		var connLines []string
		for _, c := range userConnectors {
			connLines = append(connLines, fmt.Sprintf("- Conector %s (Tipo: %s, Endpoint: %s)", c.Name, c.Type, c.Endpoint))
		}
		systemPrompt += fmt.Sprintf("\n\n[Conectores MCP Activos]:\n%s", strings.Join(connLines, "\n"))
	}

	if chat.ProjectID != "" {
		if project, err := db.GetProject(chat.ProjectID); err == nil && project.InstructionsMd != "" {
			systemPrompt = fmt.Sprintf("[Instrucciones del proyecto \"%s\"]:\n%s\n\n---\n\n%s",
				project.Name, project.InstructionsMd, systemPrompt)
		}
		// Code mode: inject graph context for referenced files
		if chat.Mode == "code" {
			graphCtx := memory.BuildGraphContext(chat.ProjectID, msg.Content)
			if graphCtx != "" {
				systemPrompt = fmt.Sprintf("%s\n\n[Contexto de dependencias del proyecto]:\n%s", systemPrompt, graphCtx)
			}
		}
	}

	// Memoria Jerárquica Continua: inyección de hechos relevantes mediante FTS5 BM25
	if defaultStore := memory.DefaultStore(); defaultStore != nil {
		if facts, err := defaultStore.SearchRelevant(context.Background(), chat.UserID, msg.Content, 5); err == nil && len(facts) > 0 {
			var factLines []string
			for _, f := range facts {
				factLines = append(factLines, fmt.Sprintf("- [%s]: %s", strings.ToUpper(string(f.Category)), f.Content))
			}
			systemPrompt += fmt.Sprintf("\n\n[Memoria Jerárquica Continua del Usuario]:\n%s", strings.Join(factLines, "\n"))
		}
	}

	history = append(history, providers.Message{Role: "system", Content: systemPrompt})

	for i, m := range prevMessages {
		content := m.Content
		// Replace last user message with processed content (OCR'd)
		if m.Role == "user" && i == len(prevMessages)-1 {
			content = proc.content
		}
		history = append(history, providers.Message{Role: m.Role, Content: content})
	}

	chunkCh, err := provider.StreamCompletion(ctx, history, providers.CompletionOptions{Stream: true, Model: modelToUse})
	if err != nil {
		writeJSON(client, serverMessage{Type: "error", Content: err.Error()})
		return
	}

	var fullContent string
	var toolCalls []map[string]any
	msgID := uuid.NewString()
	isDoneHandled := false

	saveAndEmitDone := func() {
		if isDoneHandled {
			return
		}
		isDoneHandled = true
		assistantMsg := &models.Message{
			ID:        msgID,
			ChatID:    msg.ChatID,
			Role:      "assistant",
			Content:   fullContent,
			CreatedAt: time.Now(),
		}
		if len(toolCalls) > 0 {
			tcJSON, _ := json.Marshal(toolCalls)
			assistantMsg.ToolCallsJSON = string(tcJSON)
		}
		if err := db.CreateMessage(assistantMsg); err != nil {
			log.Printf("Error guardando mensaje assistant: %v", err)
		}
		memory.StoreChatMessage(chat.UserID, chat.ProjectID, msg.ChatID, "assistant", fullContent)
		if ext := memory.DefaultExtractor(); ext != nil {
			ext.ExtractAndPersistAsync(context.Background(), chat.UserID, msg.Content, fullContent)
		}
		writeJSON(client, serverMessage{Type: "done", MessageID: msgID})
	}

	for chunk := range chunkCh {
		switch chunk.Type {
		case "text":
			fullContent += chunk.Content
			writeJSON(client, serverMessage{Type: "text", Content: chunk.Content})
		case "tool_call":
			// Modo Chat normal — el LLM emitió un tool_call (informativo, no se ejecuta aquí)
			if chunk.ToolCall != nil {
				tc := chunk.ToolCall
				inputStr := string(tc.Input)
				toolCalls = append(toolCalls, map[string]any{
					"id":        tc.ID,
					"name":      tc.Name,
					"arguments": inputStr,
				})
				writeJSON(client, serverMessage{
					Type:      "tool_call",
					ToolID:    tc.ID,
					ToolName:  tc.Name,
					ToolInput: inputStr,
				})
			}
		case "done":
			saveAndEmitDone()
		case "error":
			writeJSON(client, serverMessage{Type: "error", Content: chunk.Content})
		}
	}

	if !isDoneHandled && (fullContent != "" || len(toolCalls) > 0) {
		saveAndEmitDone()
	}
}

func CancelStream(chatID string) {
	agent.CancelChat(chatID)
	activeStreamsMu.Lock()
	defer activeStreamsMu.Unlock()
	for _, s := range activeStreams {
		if s.chatID == chatID {
			s.cancel()
			return
		}
	}
}

type pendingConsent struct {
	client          *Client
	msg             clientMessage
	chat            *models.Chat
	userID          string
	permissionLevel string
}

var consentMu sync.Mutex
var pendingConsents = map[string]*pendingConsent{}

func storePendingConsent(client *Client, msg clientMessage, chat *models.Chat, userID, permissionLevel string) {
	consentMu.Lock()
	pendingConsents[chat.ID] = &pendingConsent{client, msg, chat, userID, permissionLevel}
	consentMu.Unlock()
}

func handleConsentResponse(client *Client, msg clientMessage) {
	consentMu.Lock()
	p, ok := pendingConsents[msg.ChatID]
	delete(pendingConsents, msg.ChatID)
	consentMu.Unlock()

	if !ok {
		writeJSON(client, serverMessage{Type: "error", Content: "no pending consent"})
		return
	}

	decision := msg.Content // "always" | "once" | "no"
	switch decision {
	case "always":
		project, err := db.GetProject(p.chat.ProjectID)
		if err == nil {
			db.UpdateProjectConsent(project.ID, "always")
		}
		runAgentTask(client, p.msg, p.chat, project, "always")
	case "once":
		runAgentTask(client, p.msg, p.chat, nil, "once")
	case "no":
		// Process as normal chat — re-call handleChatMessage but skip consent check
		processAsNormalChat(client, p.msg, p.chat)
	}
}

func runAgentTask(client *Client, msg clientMessage, chat *models.Chat, project *models.Project, mode string) {
	provider, err := providers.Get(chat.Provider)
	if err != nil {
		writeJSON(client, serverMessage{Type: "error", Content: "provider no disponible"})
		return
	}

	permLevel := "sandboxed"
	if project != nil && project.PermissionLevel != "" {
		permLevel = project.PermissionLevel
	}

	// Process attachments (OCR images, etc.)
	proc := processAttachments(chat, msg.Attachments, msg.Content)
	for _, w := range proc.warnings {
		writeJSON(client, serverMessage{Type: "warn", Warning: w})
	}

	writeJSON(client, serverMessage{Type: "agent_start", Content: proc.content})

	taskID, results, err := agent.CreateAndExecuteTask(provider, chat.ProjectID, chat.ID,
		chat.UserID, proc.content, permLevel,
		func(sr agent.StepResult) {
			data, _ := json.Marshal(sr)
			writeJSON(client, serverMessage{Type: "agent_step", Content: string(data)})
		})

	if err != nil {
		writeJSON(client, serverMessage{Type: "error", Content: "agent task failed: " + err.Error()})
		return
	}

	resultsJSON, _ := json.Marshal(results)
	assistantMsg := &models.Message{
		ID:        taskID,
		ChatID:    msg.ChatID,
		Role:      "assistant",
		Content:   string(resultsJSON),
		CreatedAt: time.Now(),
	}
	if err := db.CreateMessage(assistantMsg); err != nil {
		log.Printf("Error guardando mensaje assistant de agente: %v", err)
	}
	memory.StoreChatMessage(chat.UserID, chat.ProjectID, msg.ChatID, "assistant", string(resultsJSON))
	writeJSON(client, serverMessage{
		Type:      "agent_done",
		Content:   string(resultsJSON),
		MessageID: taskID,
	})
}

func processAsNormalChat(client *Client, msg clientMessage, chat *models.Chat) {
	handleChatMessageNormal(client, msg, chat)
}

// runReActLoopSession arranca el ReAct Loop para Modo Code y conecta los eventos al cliente WS.
// Retorna inmediatamente — el loop corre de forma asíncrona en su goroutine.
func runReActLoopSession(client *Client, msg clientMessage, chat *models.Chat) {
	provider, err := providers.Get(chat.Provider)
	if err != nil {
		// Intentar con el primer provider disponible
		available := providers.Available()
		if len(available) == 0 {
			writeJSON(client, serverMessage{Type: "error", Content: "No hay ningún proveedor LLM configurado."})
			return
		}
		provider, err = providers.Get(available[0])
		if err != nil {
			writeJSON(client, serverMessage{Type: "error", Content: "Provider no disponible: " + err.Error()})
			return
		}
	}

	// Cargar proyecto para el contexto y permisos
	var project *models.Project
	if chat.ProjectID != "" {
		p, err := db.GetProject(chat.ProjectID)
		if err == nil {
			project = p
		}
	}

	permLevel := "sandboxed"
	if project != nil && project.PermissionLevel != "" {
		permLevel = project.PermissionLevel
	}

	// Callback de emisión — convierte AgentEvent a serverMessage WS
	emit := func(ev agent.AgentEvent) {
		writeJSON(client, serverMessage{
			Type:        ev.Type,
			State:       ev.State,
			Content:     ev.Content,
			MessageID:   ev.MessageID,
			ToolID:      ev.ToolID,
			ToolName:    ev.ToolName,
			ToolInput:   ev.ToolInput,
			ToolOutput:  ev.ToolOutput,
			ToolSuccess: ev.ToolSuccess,
			DurationMs:  ev.DurationMs,
			TaskID:      ev.TaskID,
			Turns:       ev.Turns,
			Error:       ev.Error,
		})
	}

	// Arrancar el loop asíncrono
	// VoiceMode se activa si el cliente lo indica explícitamente o si el tipo del mensaje es "voice_message".
	isVoice := msg.VoiceMode || msg.Type == "voice_message"
	sessionID := agent.StartAgentLoop(context.Background(), agent.AgentLoopParams{
		Provider:        provider,
		Chat:            chat,
		Project:         project,
		UserMessage:     msg.Content,
		PermissionLevel: permLevel,
		Emit:            emit,
		VoiceMode:       isVoice,
	})

	// Enviar session_id al cliente para que pueda cancelar o responder aprobaciones
	writeJSON(client, serverMessage{
		Type:      "agent:session_started",
		SessionID: sessionID,
	})
}

func writeJSON(client *Client, msg serverMessage) {
	data, _ := json.Marshal(msg)
	select {
	case client.Send <- data:
	default:
	}
}

