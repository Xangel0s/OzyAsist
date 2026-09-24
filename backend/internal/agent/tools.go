package agent

import (
	"encoding/json"
	"strings"

	"github.com/ozyassist/backend/internal/mcp"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
)

// AgentTools reúne todas las herramientas activas disponibles para el agente autónomo.
var AgentTools = func() []providers.ToolDef {
	var all []providers.ToolDef
	all = append(all, FileTools...)
	all = append(all, WebTools...)
	all = append(all, DesktopTools...)
	all = append(all, OfficeTools...)
	all = append(all, HardwareTools...)
	return all
}()

var VoiceAgentTools = []providers.ToolDef{
	{
		Name:        "os_get_desktop",
		Description: "Lista los archivos e iconos del escritorio de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_list_apps",
		Description: "Busca aplicaciones instaladas en Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{"filter":{"type":"string","description":"Nombre a buscar"}}}`),
	},
	{
		Name:        "os_active_windows",
		Description: "Lista las ventanas abiertas actualmente.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},

	{
		Name:        "os_system_info",
		Description: "Telemetr\u00eda de hardware en tiempo real: CPU, RAM, disco.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_explore",
		Description: "Explora el contenido de una carpeta.",
		InputSchema: mustJSON(`{"type":"object","properties":{"path":{"type":"string","description":"Ruta a explorar"}},"required":["path"]}`),
	},
	{
		Name:        "os_organize_folder",
		Description: "Organiza autom\u00e1ticamente los archivos de una carpeta.",
		InputSchema: mustJSON(`{"type":"object","properties":{"path":{"type":"string","description":"Ruta de la carpeta a organizar"}},"required":["path"]}`),
	},
	{
		Name:        "web_search",
		Description: "Busca informaci\u00f3n en internet.",
		InputSchema: mustJSON(`{"type":"object","properties":{"query":{"type":"string","description":"T\u00e9rmino de b\u00fasqueda"}},"required":["query"]}`),
	},
	{
		Name:        "os_hardware_control",
		Description: "Ajusta el brillo o volumen del PC (ej: 'volume' 50, 'brightness' 100).",
		InputSchema: mustJSON(`{"type":"object","properties":{"setting":{"type":"string"},"value":{"type":"integer"}},"required":["setting","value"]}`),
	},
	{
		Name:        "os_power_state",
		Description: "Apaga, reinicia o suspende la PC ('shutdown', 'restart', 'sleep').",
		InputSchema: mustJSON(`{"type":"object","properties":{"state":{"type":"string"}},"required":["state"]}`),
	},
	{
		Name:        "os_launch_app",
		Description: "Lanza o abre una aplicación en la PC (soporta 'antigravity', 'code', 'calc', 'notepad', etc.) y opcionalmente una ruta de carpeta o proyecto.",
		InputSchema: mustJSON(`{"type":"object","properties":{"appName":{"type":"string"},"path":{"type":"string"}},"required":["appName"]}`),
	},
	{
		Name:        "os_run_command",
		Description: "Ejecuta un comando en consola de Windows (PowerShell/Git/CLI). Úsalo para 'git log', 'git status', 'dir', etc.",
		InputSchema: mustJSON(`{"type":"object","properties":{"command":{"type":"string","description":"Comando a ejecutar"}},"required":["command"]}`),
	},
	{
		Name:        "deep_search",
		Description: "Investigación profunda en internet con citas grounded y reporte completo.",
		InputSchema: mustJSON(`{"type":"object","properties":{"query":{"type":"string","description":"Tema a investigar"}},"required":["query"]}`),
	},
	{
		Name:        "os_draft_email",
		Description: "Redacta un correo electrónico formal y abre la ventana del cliente de correo en pantalla para revisar y enviar.",
		InputSchema: mustJSON(`{"type":"object","properties":{"to":{"type":"string","description":"Destinatario"},"subject":{"type":"string","description":"Asunto"},"body":{"type":"string","description":"Cuerpo del correo"}},"required":["to","subject","body"]}`),
	},
	{
		Name:        "os_draft_whatsapp",
		Description: "Redacta un mensaje de WhatsApp y abre la conversación con el mensaje listo para enviar.",
		InputSchema: mustJSON(`{"type":"object","properties":{"phone":{"type":"string","description":"Teléfono"},"text":{"type":"string","description":"Mensaje"}},"required":["text"]}`),
	},
	{
		Name:        "os_draft_telegram",
		Description: "Redacta un mensaje de Telegram y abre el chat listo para enviar.",
		InputSchema: mustJSON(`{"type":"object","properties":{"recipient":{"type":"string","description":"Usuario o número"},"text":{"type":"string","description":"Mensaje"}},"required":["text"]}`),
	},
	{
		Name:        "os_get_clipboard",
		Description: "Lee el contenido actual del portapapeles de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_set_clipboard",
		Description: "Copia texto al portapapeles de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
	},
	{
		Name:        "os_notify",
		Description: "Muestra una notificación emergente en la pantalla de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}`),
	},
	{
		Name:        "os_read_document",
		Description: "Lee y extrae texto de un documento local (PDF, Excel, Word, CSV, TXT).",
		InputSchema: mustJSON(`{"type":"object","properties":{"path":{"type":"string","description":"Ruta al archivo"}},"required":["path"]}`),
	},
	{
		Name:        "os_schedule_alarm",
		Description: "Programa una alarma o recordatorio ('10m', '30s', '1h', '15:30').",
		InputSchema: mustJSON(`{"type":"object","properties":{"time_in":{"type":"string"},"message":{"type":"string"}},"required":["time_in","message"]}`),
	},
	{
		Name:        "os_list_alarms",
		Description: "Lista los recordatorios activos pendientes.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_close_window",
		Description: "Cierra una ventana de Windows por título (ej: 'Administrador de tareas', 'Calculadora') o por su HWND/PID.",
		InputSchema: mustJSON(`{"type":"object","properties":{"title":{"type":"string"},"hwnd":{"type":"integer"},"pid":{"type":"integer"}}}`),
	},
	{
		Name:        "os_tile_windows",
		Description: "Acomoda o divide ventanas en pantalla (left, right, maximize, minimize, restore, show_desktop).",
		InputSchema: mustJSON(`{"type":"object","properties":{"title":{"type":"string"},"layout":{"type":"string"}}}`),
	},
	{
		Name:        "os_wifi_manager",
		Description: "Muestra el estado de la conexión Wi-Fi actual y señal.",
		InputSchema: mustJSON(`{"type":"object","properties":{"action":{"type":"string"}}}`),
	},
	{
		Name:        "os_audio_device",
		Description: "Controla audio, volumen y silencio (mute): get_volume, set_volume, mute, unmute o list.",
		InputSchema: mustJSON(`{"type":"object","properties":{"action":{"type":"string"},"level":{"type":"number"},"mute":{"type":"boolean"}}}`),
	},
	{
		Name:        "os_hardware_inspector",
		Description: "Inspecciona hardware: salud SMART ('health'), puertos USB ('devices'), cámara/mic en uso ('in_use'), y telemetría de CPU, RAM, GPU, temperatura y batería ('telemetry').",
		InputSchema: mustJSON(`{"type":"object","properties":{"action":{"type":"string"}}}`),
	},
	{
		Name:        "os_power_profile",
		Description: "Consulta o cambia el plan de energía (status, set_plan) y ajusta brillo de pantalla (set_brightness).",
		InputSchema: mustJSON(`{"type":"object","properties":{"action":{"type":"string"},"plan":{"type":"string"},"brightness":{"type":"integer"}}}`),
	},
	{
		Name:        "os_toast_notify",
		Description: "Envía una notificación Toast nativa en la pantalla de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{"title":{"type":"string"},"message":{"type":"string"}},"required":["message"]}`),
	},
	{
		Name:        "os_network_diagnostics",
		Description: "Prueba latencia de internet (test/ping), IPs locales (ip_info) y vacía caché DNS (flush_dns).",
		InputSchema: mustJSON(`{"type":"object","properties":{"action":{"type":"string"},"host":{"type":"string"}}}`),
	},
	{
		Name:        "os_kill_process",
		Description: "Cierra o termina un programa o proceso por nombre (ej: 'notepad', 'calc') o por PID.",
		InputSchema: mustJSON(`{"type":"object","properties":{"name":{"type":"string"},"pid":{"type":"integer"}}}`),
	},
	{
		Name:        "remember_fact",
		Description: "Registra y guarda en memoria permanente un hecho, regla, configuración o preferencia del usuario para no tener que volver a preguntarlo en futuras sesiones.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"category": {
					"type": "string",
					"enum": ["preferencia", "stack", "hardware", "regla", "contexto"],
					"description": "Categoría del hecho memorizado"
				},
				"content": {
					"type": "string",
					"description": "El hecho o preferencia conciso y atómico a recordar"
				}
			},
			"required": ["content"]
		}`),
	},
	{
		Name:        "search_memory",
		Description: "Busca en la memoria persistente hechos, configuraciones, preferencias previas o reglas del usuario usando búsqueda semántica y FTS5.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Términos de búsqueda o pregunta sobre recuerdos pasados"
				},
				"limit": {
					"type": "integer",
					"description": "Cantidad máxima de recuerdos a recuperar (default: 5)"
				}
			},
			"required": ["query"]
		}`),
	},
	{
		Name:        "update_user_profile",
		Description: "Actualiza o expande la ficha de perfil persistente del usuario (quién es, a qué se dedica, rol, proyectos que maneja y estilo de trabajo).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"profile_md": {
					"type": "string",
					"description": "Contenido completo en Markdown de la ficha del perfil del usuario"
				}
			},
			"required": ["profile_md"]
		}`),
	},
	{
		Name:        "learn_engram",
		Description: "Enseña o registra un nuevo atajo/engrama en el cerebro secundario en RAM del sistema para asociar una frase o modismo coloquial con una herramienta y parámetros específicos.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"trigger_phrase": {
					"type": "string",
					"description": "Frase o modismo coloquial que disparará la acción (ej: 'modo cine', 'prepara el reporte')"
				},
				"tool_name": {
					"type": "string",
					"description": "Herramienta de Windows a ejecutar (ej: 'os_power_profile', 'os_tile_windows', 'os_audio_device')"
				},
				"args": {
					"type": "object",
					"description": "Parámetros que recibirá la herramienta"
				},
				"use_case": {
					"type": "string",
					"description": "Descripción corta del caso de uso funcional"
				}
			},
			"required": ["trigger_phrase", "tool_name"]
		}`),
	},
	{
		Name:        "os_detect_dialogs",
		Description: "Inspecciona cuadros de diálogo modales y errores activos en pantalla (#32770).",
		InputSchema: mustJSON(`{"type":"object","properties":{"app_filter":{"type":"string","description":"Filtro por app o título"}}}`),
	},
	{
		Name:        "os_keyboard_layout",
		Description: "Inspecciona o cambia la distribución de teclado de Windows ('status', 'set' con 'latam', 'spain', 'us').",
		InputSchema: mustJSON(`{"type":"object","properties":{"action":{"type":"string","enum":["status","list","set"]},"layout":{"type":"string"}}}`),
	},
	{
		Name:        "os_notification_focus",
		Description: "Consulta o configura notificaciones emergentes y modo no molestar ('status', 'set' con enabled: true/false).",
		InputSchema: mustJSON(`{"type":"object","properties":{"action":{"type":"string","enum":["status","set"]},"enabled":{"type":"boolean"}}}`),
	},
	{
		Name:        "os_get_clipboard",
		Description: "Obtiene el texto actual copiado en el portapapeles de Windows.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_media_control",
		Description: "Control multimedia (play_pause, next, previous, stop) en Spotify/YouTube.",
		InputSchema: mustJSON(`{"type":"object","properties":{"action":{"type":"string","enum":["play_pause","next","previous","stop"]}}}`),
	},
	{
		Name:        "os_display_config",
		Description: "Configura proyección de pantalla ('status', 'extend', 'clone', 'internal', 'external').",
		InputSchema: mustJSON(`{"type":"object","properties":{"action":{"type":"string","enum":["status","extend","clone","internal","external"]}}}`),
	},
}

// GetActiveTools returns the tools to inject into the LLM, including dynamic MCP tools.
// En voiceMode devuelve únicamente VoiceAgentTools para garantizar baja latencia y prefill instantáneo.
func GetActiveTools(voiceMode bool) []providers.ToolDef {
	if voiceMode {
		return VoiceAgentTools
	}

	tools := make([]providers.ToolDef, len(AgentTools))
	copy(tools, AgentTools)

	// Dynamic MCP tools (solo en modo texto / consola para proteger latencia y tokens en voz)
	mcpTools := mcp.DefaultRegistry.GetAllTools()
	for name, mcpTool := range mcpTools {
		// Convert MCP input schema to JSON RawMessage
		schemaBytes, _ := json.Marshal(mcpTool.InputSchema)

		tools = append(tools, providers.ToolDef{
			Name:        name,
			Description: mcpTool.Description + " [MCP]",
			InputSchema: json.RawMessage(schemaBytes),
		})
	}

	return tools
}

// GetActiveToolsForQuery aplica poda dinámica de herramientas (Dynamic Tool Pruning)
// utilizando el SystemGraph para que modelos pequeños locales y modo voz reciban únicamente
// las 1 a 3 herramientas candidatas más sus complementos de co-ocurrencia.
func GetActiveToolsForQuery(query string, voiceMode bool, isLocal bool) []providers.ToolDef {
	baseTools := GetActiveTools(voiceMode)
	if !isLocal && !voiceMode {
		return baseTools
	}

	clean := memory.NormalizeColloquialText(query)
	tokens := strings.Fields(clean)
	if isLocal && memory.IsMetaConversationalQuery(query, clean, tokens) {
		// En consultas conversacionales, de depuración o reflexivas, no saturar al modelo local con 50 herramientas
		return nil
	}

	graph := memory.GetSystemGraph()
	if graph == nil {
		if isLocal && !voiceMode {
			return VoiceAgentTools
		}
		return baseTools
	}

	clauses := memory.SplitCompoundClauses(query)
	allowedNames := make(map[string]bool)

	for _, clause := range clauses {
		if match, ok := graph.ResolveIntent(clause); ok && match != nil && match.Engram != nil {
			targetTool := match.Engram.ToolName
			allowedNames[targetTool] = true
			for _, co := range graph.GetCoOccurringTools([]string{targetTool}) {
				allowedNames[co] = true
			}
			for _, co := range match.Engram.CoOccurringTools {
				allowedNames[co] = true
			}
		}
	}

	if len(allowedNames) == 0 {
		if isLocal && !voiceMode {
			return VoiceAgentTools
		}
		return baseTools
	}

	var pruned []providers.ToolDef
	for _, t := range baseTools {
		if allowedNames[t.Name] {
			pruned = append(pruned, t)
		}
	}

	// Si por alguna razón la poda quedó vacía, devolver la lista base
	if len(pruned) == 0 {
		if isLocal && !voiceMode {
			return VoiceAgentTools
		}
		return baseTools
	}

	return pruned
}

func mustJSON(s string) json.RawMessage {
	// Comprimir el JSON eliminando espacios innecesarios
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic("tools.go: invalid JSON schema: " + err.Error())
	}
	raw, _ := json.Marshal(v)
	return json.RawMessage(raw)
}
