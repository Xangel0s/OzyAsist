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

	"github.com/gorilla/websocket"
	"github.com/ozyassist/backend/internal/agent"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/ocr"
	"github.com/ozyassist/backend/internal/providers"
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
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	MessageID string `json:"message_id,omitempty"`
	ToolID    string `json:"tool_id,omitempty"`
	ToolName  string `json:"tool_name,omitempty"`
	ToolInput string `json:"tool_input,omitempty"`
	Warning   string `json:"warning,omitempty"`
}

func handleChatMessage(conn *websocket.Conn, msg clientMessage) {
	chat, err := db.GetChat(msg.ChatID)
	if err != nil {
		writeJSON(conn, serverMessage{Type: "error", Content: "chat no encontrado: " + err.Error()})
		return
	}

	// Agent consent check — only in code mode with a project
	if chat.ProjectID != "" && chat.Mode == "code" && agent.DetectActionIntent(msg.Content) {
		project, err := db.GetProject(chat.ProjectID)
		if err == nil {
			switch project.AgentConsent {
			case "always":
				runAgentTask(conn, msg, chat, project, "always")
				return
			case "ask":
				writeJSON(conn, serverMessage{Type: "consent_required", Content: msg.Content})
				storePendingConsent(conn, msg, chat, project.UserID, project.PermissionLevel)
				return
			}
		}
	}

	handleChatMessageNormal(conn, msg, chat)
}

func handleChatMessageNormal(conn *websocket.Conn, msg clientMessage, chat *models.Chat) {

	ctx, cancel := context.WithCancel(context.Background())
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
		writeJSON(conn, serverMessage{Type: "warn", Warning: w})
	}

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
		writeJSON(conn, serverMessage{Type: "error", Content: "error guardando mensaje"})
		return
	}
	memory.StoreChatMessage(chat.UserID, chat.ProjectID, msg.ChatID, "user", msg.Content)

	prevMessages, err := db.GetMessages(msg.ChatID)
	if err != nil {
		log.Printf("Error cargando historial: %v", err)
	}

	providerName := chat.Provider
	if providerName == "" {
		providerName = "openai"
	}
	provider, err := providers.Get(providerName)
	if err != nil {
		writeJSON(conn, serverMessage{Type: "error", Content: "provider no disponible: " + err.Error()})
		return
	}

	var history []providers.Message

	systemPrompt := `Eres Ozy, el asistente de OzyAssist — una aplicación de escritorio para productividad
de desarrollo. Corres como un LLM con acceso a estas capacidades reales:

- Memoria de proyecto (Capa 1): puedes leer y mantener contexto de archivos .md del
  proyecto activo (instrucciones, decisiones de arquitectura, convenciones).
- Memoria episódica (Capa 2): recuerdas automáticamente fragmentos de conversaciones
  pasadas usando búsqueda semántica vectorial (Qdrant).
- Skills: el usuario puede definir y ejecutar skills personalizadas (prompt templates,
  scripts, API calls) desde la sección Skills.
- Conectores MCP: el usuario puede conectar herramientas externas vía MCP (Model
  Context Protocol) desde la sección Connectors.
- Agente: el usuario puede lanzar tareas autónomas multi-paso (leer/escribir archivos,
  ejecutar comandos en sandbox) que requieren confirmación explícita. Las tareas de
  agente se crean y monitorean desde los endpoints de Agent, no desde el chat normal.
- Búsqueda global (Ctrl+K): búsqueda full-text (FTS5) en todas las conversaciones
  anteriores y entradas de memoria.
- Selección de modelo y proveedor: el usuario puede elegir entre distintos modelos
  y proveedores LLM desde el selector en el chat.

No tienes acceso directo a internet. No puedes controlar el sistema operativo del
usuario directamente — toda acción en el filesystem pasa por el sandbox del agente
y requiere confirmación explícita. No inventes capacidades que no tienes.

Responde siempre en español a menos que el usuario te pida otro idioma. Sé conciso
y directo. Si no sabes algo, dilo sin rodeos.`

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

	history = append(history, providers.Message{Role: "system", Content: systemPrompt})

	for i, m := range prevMessages {
		content := m.Content
		// Replace last user message with processed content (OCR'd)
		if m.Role == "user" && i == len(prevMessages)-1 {
			content = proc.content
		}
		history = append(history, providers.Message{Role: m.Role, Content: content})
	}

	chunkCh, err := provider.StreamCompletion(ctx, history, providers.CompletionOptions{Stream: true, Model: chat.Model})
	if err != nil {
		writeJSON(conn, serverMessage{Type: "error", Content: err.Error()})
		return
	}

	var fullContent string
	var toolCalls []map[string]any
	msgID := uuid.NewString()

	for chunk := range chunkCh {
		switch chunk.Type {
		case "text":
			fullContent += chunk.Content
			writeJSON(conn, serverMessage{Type: "text", Content: chunk.Content})
		case "tool_call":
			toolCalls = append(toolCalls, map[string]any{
				"id":        chunk.ToolID,
				"name":      chunk.ToolName,
				"arguments": chunk.ToolInput,
			})
			writeJSON(conn, serverMessage{
				Type:      "tool_call",
				ToolID:    chunk.ToolID,
				ToolName:  chunk.ToolName,
				ToolInput: chunk.ToolInput,
			})
		case "done":
			assistantMsg := &models.Message{
				ID:             msgID,
				ChatID:         msg.ChatID,
				Role:           "assistant",
				Content:        fullContent,
				CreatedAt:      time.Now(),
			}
			if len(toolCalls) > 0 {
				tcJSON, _ := json.Marshal(toolCalls)
				assistantMsg.ToolCallsJSON = string(tcJSON)
			}
			if err := db.CreateMessage(assistantMsg); err != nil {
				log.Printf("Error guardando mensaje assistant: %v", err)
			}
			memory.StoreChatMessage(chat.UserID, chat.ProjectID, msg.ChatID, "assistant", fullContent)
			writeJSON(conn, serverMessage{Type: "done", MessageID: msgID})
		case "error":
			writeJSON(conn, serverMessage{Type: "error", Content: chunk.Content})
		}
	}
}

func CancelStream(chatID string) {
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
	conn            *websocket.Conn
	msg             clientMessage
	chat            *models.Chat
	userID          string
	permissionLevel string
}

var consentMu sync.Mutex
var pendingConsents = map[string]*pendingConsent{}

func storePendingConsent(conn *websocket.Conn, msg clientMessage, chat *models.Chat, userID, permissionLevel string) {
	consentMu.Lock()
	pendingConsents[chat.ID] = &pendingConsent{conn, msg, chat, userID, permissionLevel}
	consentMu.Unlock()
}

func handleConsentResponse(conn *websocket.Conn, msg clientMessage) {
	consentMu.Lock()
	p, ok := pendingConsents[msg.ChatID]
	delete(pendingConsents, msg.ChatID)
	consentMu.Unlock()

	if !ok {
		writeJSON(conn, serverMessage{Type: "error", Content: "no pending consent"})
		return
	}

	decision := msg.Content // "always" | "once" | "no"
	switch decision {
	case "always":
		project, err := db.GetProject(p.chat.ProjectID)
		if err == nil {
			db.UpdateProjectConsent(project.ID, "always")
		}
		runAgentTask(conn, p.msg, p.chat, project, "always")
	case "once":
		runAgentTask(conn, p.msg, p.chat, nil, "once")
	case "no":
		// Process as normal chat — re-call handleChatMessage but skip consent check
		processAsNormalChat(conn, p.msg, p.chat)
	}
}

func runAgentTask(conn *websocket.Conn, msg clientMessage, chat *models.Chat, project *models.Project, mode string) {
	provider, err := providers.Get(chat.Provider)
	if err != nil {
		writeJSON(conn, serverMessage{Type: "error", Content: "provider no disponible"})
		return
	}

	permLevel := "sandboxed"
	if project != nil && project.PermissionLevel != "" {
		permLevel = project.PermissionLevel
	}

	writeJSON(conn, serverMessage{Type: "agent_start", Content: msg.Content})

	taskID, results, err := agent.CreateAndExecuteTask(provider, chat.ProjectID, chat.ID,
		chat.UserID, msg.Content, permLevel,
		func(sr agent.StepResult) {
			data, _ := json.Marshal(sr)
			writeJSON(conn, serverMessage{Type: "agent_step", Content: string(data)})
		})

	if err != nil {
		writeJSON(conn, serverMessage{Type: "error", Content: "agent task failed: " + err.Error()})
		return
	}

	resultsJSON, _ := json.Marshal(results)
	writeJSON(conn, serverMessage{
		Type:      "agent_done",
		Content:   string(resultsJSON),
		MessageID: taskID,
	})
}

func processAsNormalChat(conn *websocket.Conn, msg clientMessage, chat *models.Chat) {
	handleChatMessageNormal(conn, msg, chat)
}

func writeJSON(conn *websocket.Conn, msg serverMessage) {
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}
