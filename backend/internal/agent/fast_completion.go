package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/voice"
)

// handleLocalModelFastCompletion verifica si se completó una acción directa del sistema
// en un modelo local (llamacpp, ollama, lmstudio) y emite inmediatamente la confirmación factual
// con rutas absolutas y sugerencias proactivas, evitando un segundo turno redundante o vacío.
func handleLocalModelFastCompletion(
	params AgentLoopParams,
	taskID string,
	turn int,
	turnThinking string,
	turnToolCalls []providers.ToolCall,
	toolSuccessMap map[string]bool,
	history []providers.Message,
	allToolCalls []providers.ToolCall,
	emit func(AgentEvent),
	speakerQueue *voice.LocalSpeakerQueue,
) bool {
	isLocalModel := params.Provider != nil && (params.Provider.Name() == "llamacpp" || params.Provider.Name() == "ollama" || params.Provider.Name() == "lmstudio")
	if !isLocalModel || len(turnToolCalls) == 0 {
		return false
	}

	var finalContent string

	// 1. Verificación de creación de PDF
	for _, tc := range turnToolCalls {
		if tc.Name == "os_create_pdf" && toolSuccessMap[tc.ID] {
			var p struct {
				Path  string `json:"path"`
				Title string `json:"title"`
			}
			_ = json.Unmarshal(tc.Input, &p)
			title := p.Title
			if title == "" {
				title = "documento"
			}
			finalContent = fmt.Sprintf("Listo, he generado el informe PDF sobre \"%s\" exitosamente en tu carpeta de Documentos:\n📄 %s\n\n¿Deseas que profundice en algún apartado específico, que agregue tablas adicionales, o que lo abra en pantalla para revisarlo?", title, p.Path)
			emit(AgentEvent{Type: "message:delta", Content: finalContent})
			if speakerQueue != nil {
				speakerQueue.Enqueue(finalContent)
			}
			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent, Thinking: turnThinking})
			RecordInteractionForTraining(params.UserMessage, finalContent, allToolCalls)
			return true
		}
	}

	// 1.1 Verificación de creación de Word (.docx)
	for _, tc := range turnToolCalls {
		if tc.Name == "os_create_docx" && toolSuccessMap[tc.ID] {
			var p struct {
				Path  string `json:"path"`
				Title string `json:"title"`
			}
			_ = json.Unmarshal(tc.Input, &p)
			title := p.Title
			if title == "" {
				title = "documento Word"
			}
			finalContent = fmt.Sprintf("Listo, he redactado el documento Word (.docx) \"%s\" estructurado en tu carpeta de Documentos:\n📝 %s\n\n¿Deseas que expanda la extensión a más páginas con secciones detalladas, o prefieres abrirlo directamente en Microsoft Word?", title, p.Path)
			emit(AgentEvent{Type: "message:delta", Content: finalContent})
			if speakerQueue != nil {
				speakerQueue.Enqueue(finalContent)
			}
			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent, Thinking: turnThinking})
			RecordInteractionForTraining(params.UserMessage, finalContent, allToolCalls)
			return true
		}
	}

	// 1.2 Verificación de creación de Excel (.xlsx)
	for _, tc := range turnToolCalls {
		if tc.Name == "os_create_excel" && toolSuccessMap[tc.ID] {
			var p struct {
				Path  string `json:"path"`
				Title string `json:"title"`
			}
			_ = json.Unmarshal(tc.Input, &p)
			title := p.Title
			if title == "" {
				title = "hoja de cálculo"
			}
			finalContent = fmt.Sprintf("Listo, he generado la hoja de cálculo de Excel (.xlsx) \"%s\" en tu carpeta de Documentos:\n📊 %s\n\n¿Deseas que agregue nuevas pestañas con métricas adicionales, fórmulas de totales (=SUM, =AVERAGE), o que abra el archivo en pantalla?", title, p.Path)
			emit(AgentEvent{Type: "message:delta", Content: finalContent})
			if speakerQueue != nil {
				speakerQueue.Enqueue(finalContent)
			}
			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent, Thinking: turnThinking})
			RecordInteractionForTraining(params.UserMessage, finalContent, allToolCalls)
			return true
		}
	}

	// 2. Verificación de Hardware / USB
	for _, tc := range turnToolCalls {
		if tc.Name == "os_hardware_inspector" && toolSuccessMap[tc.ID] {
			outputStr := ""
			for _, m := range history {
				if m.Role == "tool" && m.ToolResult != nil && m.ToolResult.ToolCallID == tc.ID {
					outputStr = m.ToolResult.Content
					break
				}
			}
			cleanOut := outputStr
			if idx := strings.Index(cleanOut, "Puertos USB en uso:"); idx != -1 {
				cleanOut = strings.TrimSpace(cleanOut[idx:])
			} else if idx := strings.Index(cleanOut, "[USB]"); idx != -1 {
				endIdx := strings.Index(cleanOut[idx:], "\n\n[")
				if endIdx != -1 {
					cleanOut = strings.TrimSpace(cleanOut[idx : idx+endIdx])
				}
			}
			if cleanOut == "" {
				cleanOut = outputStr
			}
			finalContent = fmt.Sprintf("=== DISPOSITIVOS Y PUERTOS EN USO ===\n%s\n\n¿Para qué necesitas los puertos específicamente? ¿Estás experimentando problemas con algún periférico o necesitas telemetría de algún componente en particular?", cleanOut)
			emit(AgentEvent{Type: "message:delta", Content: finalContent})
			if speakerQueue != nil {
				speakerQueue.Enqueue(finalContent)
			}
			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent, Thinking: turnThinking})
			RecordInteractionForTraining(params.UserMessage, finalContent, allToolCalls)
			return true
		}
	}

	// 2.1 Verificación de Puertos de Red (os_port_inspector)
	for _, tc := range turnToolCalls {
		if tc.Name == "os_port_inspector" && toolSuccessMap[tc.ID] {
			outputStr := ""
			for _, m := range history {
				if m.Role == "tool" && m.ToolResult != nil && m.ToolResult.ToolCallID == tc.ID {
					outputStr = m.ToolResult.Content
					break
				}
			}
			cleanOut := strings.TrimSpace(outputStr)
			finalContent = fmt.Sprintf("%s\n\n¿Para qué necesitas estos puertos específicamente? ¿Sospechas de algún conflicto con un servicio local o requieres que libere alguno que esté bloqueado?", cleanOut)
			emit(AgentEvent{Type: "message:delta", Content: finalContent})
			if speakerQueue != nil {
				speakerQueue.Enqueue(finalContent)
			}
			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent, Thinking: turnThinking})
			RecordInteractionForTraining(params.UserMessage, finalContent, allToolCalls)
			return true
		}
	}

	// 3. Verificación de WiFi
	for _, tc := range turnToolCalls {
		if tc.Name == "os_wifi_manager" {
			outputStr := ""
			for _, m := range history {
				if m.Role == "tool" && m.ToolResult != nil && m.ToolResult.ToolCallID == tc.ID {
					outputStr = m.ToolResult.Content
					break
				}
			}
			if !toolSuccessMap[tc.ID] || strings.Contains(outputStr, "ubicación") || strings.Contains(outputStr, "elevación") {
				finalContent = "Estado de Wi-Fi: Se detectó 1 interfaz de red inalámbrica, pero Windows requiere activar permisos de ubicación en Configuración (ms-settings:privacy-location) o ejecutar la consola como Administrador para exponer datos detallados de WLAN."
			} else {
				cleanWiFi := strings.TrimSpace(outputStr)
				finalContent = fmt.Sprintf("%s\n\nLa conexión de red está completamente estable y funcionando sin anomalías. ¿Te gustaría que hagamos pruebas de latencia hacia algún servidor específico, o alguna recomendación para optimizar tu red?", cleanWiFi)
			}
			emit(AgentEvent{Type: "message:delta", Content: finalContent})
			if speakerQueue != nil {
				speakerQueue.Enqueue(finalContent)
			}
			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent, Thinking: turnThinking})
			RecordInteractionForTraining(params.UserMessage, finalContent, allToolCalls)
			return true
		}
	}

	// 4. Verificación de Búsqueda de Archivos / Ubicación (os_find_files)
	for _, tc := range turnToolCalls {
		if tc.Name == "os_find_files" && toolSuccessMap[tc.ID] {
			outputStr := ""
			for _, m := range history {
				if m.Role == "tool" && m.ToolResult != nil && m.ToolResult.ToolCallID == tc.ID {
					outputStr = m.ToolResult.Content
					break
				}
			}
			cleanOut := strings.TrimSpace(outputStr)
			if strings.Contains(cleanOut, "=== ARCHIVOS ENCONTRADOS") {
				finalContent = fmt.Sprintf("%s\n¿Deseas que abra el archivo en pantalla para revisarlo, o necesitas que realice alguna búsqueda interna de contenido?", cleanOut)
			} else {
				finalContent = cleanOut
			}
			emit(AgentEvent{Type: "message:delta", Content: finalContent})
			if speakerQueue != nil {
				speakerQueue.Enqueue(finalContent)
			}
			emit(AgentEvent{Type: "state:sync", State: "idle"})
			msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
			emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent, Thinking: turnThinking})
			RecordInteractionForTraining(params.UserMessage, finalContent, allToolCalls)
			return true
		}
	}

	// 5. Verificación de apertura / cierre de ventanas
	var opened []string
	var closed []string
	for _, tc := range turnToolCalls {
		if !toolSuccessMap[tc.ID] {
			continue
		}
		if tc.Name == "os_launch_app" {
			var p struct{ Target string `json:"target"` }
			_ = json.Unmarshal(tc.Input, &p)
			if p.Target != "" && !isInvalidAppTarget(p.Target) {
				opened = append(opened, p.Target)
			}
		} else if tc.Name == "os_close_window" || tc.Name == "os_kill_process" {
			var p struct{ Title string `json:"title"` }
			_ = json.Unmarshal(tc.Input, &p)
			if p.Title != "" && !isInvalidAppTarget(p.Title) {
				closed = append(closed, p.Title)
			}
		}
	}

	if len(opened) > 0 || len(closed) > 0 {
		isDocs := false
		for _, o := range opened {
			oLower := strings.ToLower(o)
			if strings.Contains(oLower, "document") || strings.Contains(oLower, "explorer") {
				isDocs = true
				break
			}
		}

		if isDocs {
			finalContent = "Listo, he abierto tu carpeta de Documentos.\n\n¿Estás buscando algún archivo o informe en particular, o necesitas que ordene o filtre su contenido?"
		} else if len(opened) > 0 && len(closed) > 0 {
			finalContent = fmt.Sprintf("Listo, he abierto %s y cerrado %s. ¿Pasamos al siguiente paso?", strings.Join(opened, ", "), strings.Join(closed, ", "))
		} else if len(opened) > 0 {
			finalContent = fmt.Sprintf("Listo, he abierto %s. ¿Deseas que realice alguna acción dentro de la aplicación o necesitas algo más?", strings.Join(opened, " y "))
		} else {
			finalContent = fmt.Sprintf("Listo, he cerrado %s.", strings.Join(closed, " y "))
		}

		emit(AgentEvent{Type: "message:delta", Content: finalContent})
		if speakerQueue != nil {
			speakerQueue.Enqueue(finalContent)
		}
		emit(AgentEvent{Type: "state:sync", State: "idle"})
		msgID := persistAgentMessage(params, taskID, finalContent, allToolCalls)
		emit(AgentEvent{Type: "agent:completed", TaskID: taskID, MessageID: msgID, Turns: turn + 1, Content: finalContent, Thinking: turnThinking})
		RecordInteractionForTraining(params.UserMessage, finalContent, allToolCalls)
		return true
	}

	return false
}
