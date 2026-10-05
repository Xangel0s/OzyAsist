package agent

import (
	"strings"
	"sync"

	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
)

// ToolDomain representa un clúster temático de herramientas de alta cohesión.
type ToolDomain string

const (
	DomainOffice   ToolDomain = "office"
	DomainFiles    ToolDomain = "files"
	DomainApps     ToolDomain = "apps"
	DomainHardware ToolDomain = "hardware"
	DomainNetwork  ToolDomain = "network"
	DomainComms    ToolDomain = "comms"
	DomainCode     ToolDomain = "code"
	DomainMemory   ToolDomain = "memory"
)

// RAMToolRouter es un enrutador en memoria RAM de latencia ultra-baja (< 0.05 ms).
// Pre-serializa y mantiene en RAM los esquemas de herramientas y aplica poda semántica
// mediante diccionarios invertidos de tokens sin costo de disco ni LLM.
type RAMToolRouter struct {
	mu           sync.RWMutex
	toolIndex    map[string]providers.ToolDef
	domainTools  map[ToolDomain][]string
	keywordIndex map[string][]ToolDomain
	anchorTools  []string // Herramientas de navegación y exploración universales
}

var (
	defaultRouter     *RAMToolRouter
	defaultRouterOnce sync.Once
)

// DefaultRAMToolRouter retorna la instancia singleton del enrutador de herramientas en RAM.
func DefaultRAMToolRouter() *RAMToolRouter {
	defaultRouterOnce.Do(func() {
		defaultRouter = newRAMToolRouter()
	})
	return defaultRouter
}

func newRAMToolRouter() *RAMToolRouter {
	r := &RAMToolRouter{
		toolIndex:    make(map[string]providers.ToolDef),
		domainTools:  make(map[ToolDomain][]string),
		keywordIndex: make(map[string][]ToolDomain),
		anchorTools:  []string{"os_explore", "web_search", "os_launch_app"},
	}

	r.refreshIndex()

	// 2. Definir clústeres de dominio ordenados por prioridad funcional directa
	r.domainTools[DomainOffice] = []string{
		"os_create_docx", "os_create_pdf", "os_create_excel", "os_read_document",
		"os_convert_to_pdf",
	}
	r.domainTools[DomainFiles] = []string{
		"os_find_files", "os_explore", "os_file_info", "os_delete_item", "os_move_item",
		"os_organize_folder", "os_search_content", "os_compress_zip", "os_extract_zip",
		"read_file", "write_file", "patch_file",
	}
	r.domainTools[DomainApps] = []string{
		"os_peek_state", "os_launch_app", "os_close_window", "os_active_windows", "os_tile_windows",
		"os_kill_process", "os_detect_dialogs", "os_get_desktop", "os_list_apps",
		"os_inspect_active_ui",
	}
	r.domainTools[DomainHardware] = []string{
		"os_peek_state", "os_hardware_inspector", "os_system_info", "os_hardware_control", "os_power_profile",
		"os_power_state", "os_audio_device", "os_wifi_manager", "os_keyboard_layout",
		"os_display_config", "os_camera_capture", "os_camera_list", "os_service_manager",
	}
	r.domainTools[DomainNetwork] = []string{
		"web_search", "deep_search", "web_fetch", "web_dns_lookup", "os_network_diagnostics",
		"os_port_inspector", "os_download_file",
	}
	r.domainTools[DomainComms] = []string{
		"os_draft_email", "os_draft_whatsapp", "os_draft_telegram", "os_notify",
		"os_toast_notify",
	}
	r.domainTools[DomainCode] = []string{
		"os_skeletonize", "os_run_command", "run_command", "os_python_exec", "os_workspace_list",
	}
	r.domainTools[DomainMemory] = []string{
		"remember_fact", "search_memory", "update_user_profile", "learn_engram",
	}

	// 3. Diccionario invertido de palabras clave en RAM (Mapeo directo microsegundo)
	keywords := map[string][]ToolDomain{
		// Office
		"word": {DomainOffice}, "docx": {DomainOffice}, "documento": {DomainOffice, DomainFiles},
		"informe": {DomainOffice}, "reporte": {DomainOffice}, "pdf": {DomainOffice},
		"excel": {DomainOffice}, "xlsx": {DomainOffice}, "tabla": {DomainOffice},
		"hoja": {DomainOffice}, "calcular": {DomainOffice}, "resumen": {DomainOffice},

		// Files
		"archivo": {DomainFiles}, "archivos": {DomainFiles}, "carpeta": {DomainFiles},
		"carpetas": {DomainFiles}, "directorio": {DomainFiles}, "busca": {DomainFiles, DomainNetwork},
		"buscar": {DomainFiles, DomainNetwork}, "donde": {DomainFiles}, "ruta": {DomainFiles},
		"elimina": {DomainFiles}, "borra": {DomainFiles}, "mueve": {DomainFiles},
		"organiza": {DomainFiles}, "zip": {DomainFiles}, "comprime": {DomainFiles},
		"descomprime": {DomainFiles}, "contenido": {DomainFiles}, "grep": {DomainFiles},
		"descargas": {DomainFiles}, "documentos": {DomainFiles}, "escritorio": {DomainFiles},

		// Apps
		"abre": {DomainApps}, "abrir": {DomainApps}, "lanza": {DomainApps}, "lanzar": {DomainApps},
		"cierra": {DomainApps}, "cerrar": {DomainApps}, "ventana": {DomainApps},
		"ventanas": {DomainApps}, "app": {DomainApps}, "programa": {DomainApps},
		"calculadora": {DomainApps}, "notepad": {DomainApps}, "bloc": {DomainApps},
		"chrome": {DomainApps}, "minimiza": {DomainApps}, "maximiza": {DomainApps},
		"acomoda": {DomainApps}, "mueve ventana": {DomainApps}, "mata": {DomainApps, DomainHardware},

		// Hardware
		"volumen": {DomainHardware}, "audio": {DomainHardware}, "sonido": {DomainHardware},
		"mute": {DomainHardware}, "silencia": {DomainHardware}, "pantalla": {DomainHardware, DomainApps},
		"brillo": {DomainHardware}, "wifi": {DomainHardware}, "bateria": {DomainHardware},
		"cpu": {DomainHardware}, "ram": {DomainHardware}, "hardware": {DomainHardware},
		"usb": {DomainHardware}, "disco": {DomainHardware}, "temperatura": {DomainHardware},
		"camara": {DomainHardware}, "webcam": {DomainHardware}, "foto": {DomainHardware},
		"servicio": {DomainHardware}, "servicios": {DomainHardware}, "spooler": {DomainHardware},
		"estado": {DomainHardware, DomainApps}, "pc": {DomainHardware, DomainApps},
		"sistema": {DomainHardware, DomainApps}, "peek": {DomainApps, DomainHardware},

		// Network & Web
		"web": {DomainNetwork}, "internet": {DomainNetwork}, "google": {DomainNetwork},
		"investiga": {DomainNetwork}, "noticia": {DomainNetwork}, "noticias": {DomainNetwork},
		"sitio": {DomainNetwork}, "url": {DomainNetwork}, "dns": {DomainNetwork},
		"ping": {DomainNetwork}, "red": {DomainNetwork, DomainHardware}, "puerto": {DomainNetwork},
		"descarga": {DomainNetwork, DomainFiles}, "bajar": {DomainNetwork, DomainFiles},

		// Comms
		"correo": {DomainComms}, "email": {DomainComms}, "gmail": {DomainComms},
		"whatsapp": {DomainComms}, "mensaje": {DomainComms}, "telegram": {DomainComms},
		"notificacion": {DomainComms}, "notifica": {DomainComms}, "avisa": {DomainComms},

		// Code & Shell
		"comando": {DomainCode}, "consola": {DomainCode}, "terminal": {DomainCode},
		"powershell": {DomainCode}, "git": {DomainCode}, "script": {DomainCode},
		"python": {DomainCode}, "ejecuta": {DomainCode}, "codigo": {DomainCode},
		"compila": {DomainCode}, "test": {DomainCode}, "npm": {DomainCode},
		"esqueleto": {DomainCode}, "skeleton": {DomainCode}, "estructura": {DomainCode, DomainFiles},
		"firmas": {DomainCode}, "interfaz": {DomainApps}, "botones": {DomainApps}, "ui": {DomainApps},

		// Memory
		"recuerda": {DomainMemory}, "memoriza": {DomainMemory}, "olvida": {DomainMemory},
		"perfil": {DomainMemory}, "preferencia": {DomainMemory}, "acuerdate": {DomainMemory},
	}

	r.keywordIndex = keywords
	return r
}

func (r *RAMToolRouter) refreshIndex() {
	allTools := GetActiveTools(false)
	for _, t := range allTools {
		r.toolIndex[t.Name] = t
	}
	for _, t := range VoiceAgentTools {
		if _, exists := r.toolIndex[t.Name]; !exists {
			r.toolIndex[t.Name] = t
		}
	}
}

func isPurelyGreeting(tokens []string) bool {
	if len(tokens) == 0 || len(tokens) > 5 {
		return false
	}
	greetings := map[string]bool{
		"hola": true, "buenos": true, "dias": true, "tardes": true, "noches": true,
		"como": true, "estas": true, "que": true, "tal": true, "gracias": true,
		"saludos": true, "adios": true, "chao": true, "hey": true, "hello": true, "hi": true,
	}
	for _, t := range tokens {
		if !greetings[t] {
			return false
		}
	}
	return true
}

// RouteTools realiza el despacho y poda semántica de herramientas en memoria RAM.
// Devuelve un subconjunto ultracompacto (5-7 herramientas) para reducir el overhead de prefill
// de miles de tokens a < 600 tokens con modelos locales o modos rápidos.
func (r *RAMToolRouter) RouteTools(query string, voiceMode bool, isLocal bool) []providers.ToolDef {
	if voiceMode {
		return VoiceAgentTools
	}

	clean := memory.NormalizeColloquialText(query)
	tokens := strings.Fields(clean)

	// Saludos puramente conversacionales sin intención de acción (ahorro total de tokens en modelo local)
	if isLocal && isPurelyGreeting(tokens) {
		return nil
	}

	if len(r.toolIndex) == 0 {
		r.refreshIndex()
	}

	// 1. Conteo de afinidad de dominios en RAM (Ultra-rápido: ~15 µs)
	domainScores := make(map[ToolDomain]int)
	for _, tok := range tokens {
		if doms, found := r.keywordIndex[tok]; found {
			for _, d := range doms {
				domainScores[d]++
			}
		}
	}

	var orderedToolNames []string
	seen := make(map[string]bool)
	addTool := func(name string) {
		if !seen[name] {
			seen[name] = true
			orderedToolNames = append(orderedToolNames, name)
		}
	}

	// 2. Si no hay afinidad de palabras clave en RAM, consultar el grafo de conocimiento
	if len(domainScores) == 0 {
		graph := memory.GetSystemGraph()
		if graph != nil {
			clauses := memory.SplitCompoundClauses(query)
			for _, clause := range clauses {
				if match, ok := graph.ResolveIntent(clause); ok && match != nil && match.Engram != nil {
					addTool(match.Engram.ToolName)
					for _, co := range match.Engram.CoOccurringTools {
						addTool(co)
					}
				}
			}
		}
	}

	// 3. Seleccionar dominios dominantes por puntaje
	topDomain := ToolDomain("")
	maxScore := 0
	for d, score := range domainScores {
		if score > maxScore {
			maxScore = score
			topDomain = d
		}
	}

	if topDomain != "" {
		for _, t := range r.domainTools[topDomain] {
			addTool(t)
		}
	}

	// Si hay un segundo dominio con buen puntaje, añadir sus herramientas
	for d, score := range domainScores {
		if d != topDomain && score >= 2 {
			for _, t := range r.domainTools[d] {
				addTool(t)
			}
		}
	}

	// 4. Si ningún dominio destacó (consulta ambigua o general), usar suite generalista balanceada
	if len(orderedToolNames) == 0 {
		generalSuite := []string{
			"web_search", "os_explore", "os_launch_app", "os_close_window",
			"os_run_command", "os_read_document", "os_system_info", "remember_fact",
		}
		for _, t := range generalSuite {
			addTool(t)
		}
	} else {
		// Incluir anclas básicas al final si hay espacio
		for _, anchor := range r.anchorTools {
			addTool(anchor)
		}
	}

	// 5. Construir lista final desde la memoria RAM (cero JSON Marshal)
	maxTools := 7
	if !isLocal {
		maxTools = 16
	}

	var results []providers.ToolDef
	for _, name := range orderedToolNames {
		if def, ok := r.toolIndex[name]; ok {
			results = append(results, def)
			if isLocal && len(results) >= maxTools {
				break
			}
		}
	}

	return results
}
