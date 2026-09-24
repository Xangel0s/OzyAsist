package memory

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ozyassist/backend/internal/db"
)

// SystemEngram representa una "neurona" o patrón de conocimiento en RAM que asocia
// lenguaje coloquial y casos de uso con contratos verídicos de herramientas del sistema.
type SystemEngram struct {
	ID                 string         `json:"id"`
	UseCase            string         `json:"use_case"`
	TriggerWords       []string       `json:"trigger_words"`
	ToolName           string         `json:"tool_name"`
	DefaultArgs        map[string]any `json:"default_args"`
	CoOccurringTools   []string       `json:"co_occurring_tools"`
	FastTrack          bool           `json:"fast_track"`
	Destructive        bool           `json:"destructive"`
	Confidence         float64        `json:"confidence"`
	Maturity           string         `json:"maturity"` // "reflex" (>=0.95), "validated" (>=0.80), "candidate" (0.70)
	SuccessCount       int            `json:"success_count"`
	FeedbackTemplates  []string       `json:"feedback_templates"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// RollbackState registra el estado anterior antes de una acción Fast-Track para permitir reversión instantánea.
type RollbackState struct {
	EngramID       string         `json:"engram_id"`
	ToolName       string         `json:"tool_name"`
	ActionTaken    string         `json:"action_taken"`
	RevertToolName string         `json:"revert_tool_name"`
	RevertArgs     map[string]any `json:"revert_args"`
	Timestamp      time.Time      `json:"timestamp"`
	Description    string         `json:"description"`
}

// EngramMatch representa el resultado de resolución semántica de una intención del usuario.
type EngramMatch struct {
	Engram        *SystemEngram
	Score         float64
	ExtractedArgs map[string]any
	IsFastTrack   bool
	Feedback      string
}

// SystemGraph es el repositorio central en RAM que gestiona el grafo de capacidades,
// engramas, índice invertido, co-ocurrencias y búfer de rollback.
type SystemGraph struct {
	mu             sync.RWMutex
	engrams        map[string]*SystemEngram
	invertedIndex  map[string][]string // token normalizado -> lista de engram IDs
	coOccurrence   map[string][]string // toolName -> herramientas complementarias
	rollbackBuffer []RollbackState
	maxRollback    int
}

var (
	graphOnce     sync.Once
	systemGraph   *SystemGraph
	numberRegex   = regexp.MustCompile(`\b(\d+)\b`)
	rollbackWords = []string{"deshazlo", "deshacer", "espera no", "vuelve a ponerlo", "restaura", "reviertelo", "revertir", "ctrl z"}
)

// GetSystemGraph retorna la instancia única en memoria del grafo de sistema.
func GetSystemGraph() *SystemGraph {
	graphOnce.Do(func() {
		systemGraph = &SystemGraph{
			engrams:        make(map[string]*SystemEngram),
			invertedIndex:  make(map[string][]string),
			coOccurrence:   make(map[string][]string),
			rollbackBuffer: make([]RollbackState, 0, 10),
			maxRollback:    10,
		}
		systemGraph.initCoOccurrenceGraph()
		systemGraph.seedDefaultEngrams()
		systemGraph.loadFromSQLite()
	})
	return systemGraph
}

// initCoOccurrenceGraph define relaciones de colaboración entre herramientas complementarias.
func (g *SystemGraph) initCoOccurrenceGraph() {
	g.coOccurrence["os_take_screenshot"] = []string{"os_analyze_screen"}
	g.coOccurrence["os_analyze_screen"] = []string{"os_take_screenshot"}
	g.coOccurrence["os_explore"] = []string{"os_read_document", "os_file_info", "os_find_files"}
	g.coOccurrence["os_find_files"] = []string{"os_explore", "os_read_document", "os_file_info"}
	g.coOccurrence["os_active_windows"] = []string{"os_tile_windows", "os_close_window", "os_focus_window"}
	g.coOccurrence["web_search"] = []string{"web_fetch", "deep_search"}
	g.coOccurrence["os_hardware_inspector"] = []string{"os_disk_cleaner", "os_power_profile"}
	g.coOccurrence["os_create_docx"] = []string{"os_read_document"}
	g.coOccurrence["os_create_excel"] = []string{"os_read_document"}
	g.coOccurrence["os_create_pdf"] = []string{"os_convert_to_pdf"}
}

// NormalizeColloquialText limpia muletillas, remueve acentos y normaliza el texto.
func NormalizeColloquialText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	// Reemplazo de caracteres con acentos
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u",
		"à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u",
		"ä", "a", "ë", "e", "ï", "i", "ö", "o", "ü", "u",
		"ñ", "n",
	)
	s = replacer.Replace(s)

	// Eliminar signos de puntuación
	punct := strings.NewReplacer(
		"¿", "", "?", "", "¡", "", "!", "",
		".", "", ",", "", ";", "", ":", "",
		"(", "", ")", "", "\"", "", "'", "",
	)
	s = punct.Replace(s)

	// Remover muletillas comunes al inicio
	prefixes := []string{
		"oye ozy ", "oye ", "hola ozy ", "hola ", "por favor ", "porfa ",
		"hazme el favor ", "tengo reunion ", "puedes ", "quiero que ",
		"necesito que ", "haz el favor de ", "ozy ", "ayudame a ",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			s = strings.TrimSpace(s[len(p):])
		}
	}

	return strings.TrimSpace(s)
}

var closeVerbs = []string{
	"cierra", "cerrar", "cierrame", "apaga", "apagar", "termina", "terminar",
	"mata", "matar", "kill", "quita", "quitar", "finaliza", "finalizar",
}

var openVerbs = []string{
	"abre", "abrir", "abreme", "inicia", "iniciar", "ejecuta", "ejecutar",
	"lanza", "lanzar", "start", "run",
}

func hasAnyVerb(text string, verbs []string) bool {
	words := strings.Fields(text)
	for _, w := range words {
		for _, v := range verbs {
			if w == v {
				return true
			}
		}
	}
	return false
}

// IsMetaConversationalQuery determina si una consulta es reflexiva, conversacional,
// de depuración o una pregunta que debe ser atendida por el modelo cognitivo (LLM)
// y jamás ser interceptada por un reflejo Fast-Track.
func IsMetaConversationalQuery(rawQuery, cleanQuery string, tokens []string) bool {
	// 1. Signos de interrogación directos
	if strings.Contains(rawQuery, "?") || strings.Contains(rawQuery, "¿") {
		return true
	}

	// 2. Marcadores y temas meta-conversacionales / depuración / feedback / análisis
	metaMarkers := []string{
		"por que", "porque", "por que razon", "explica", "explicame", "explicar",
		"que opinas", "revisa", "revisar", "revisa bien", "analiza", "analizar",
		"corrige", "corregir", "actualiza", "actualizar", "grafo", "grafos",
		"error", "fallo", "falla", "bug", "problema", "automanda", "evitar",
		"fast track", "fasttrack", "como podemos", "puedes verificar", "puedes revisar",
		"verifica ello", "ultimo mensaje", "mensaje anterior", "no detecta",
		"trate de", "conversarlo", "en este caso", "pero ya esta", "pero es muy",
		"quien eres", "como funcionas", "que puedes hacer", "que sabes hacer",
		"ayudame a entender", "dime", "cuentame",
	}
	for _, m := range metaMarkers {
		if strings.Contains(cleanQuery, m) {
			return true
		}
	}

	// 3. Consultas compuestas o largas (> 7 palabras) que no comienzan con un comando imperativo directo
	if len(tokens) > 7 {
		first := tokens[0]
		directImperatives := []string{
			"abre", "abrir", "cierra", "cerrar", "sube", "subir", "baja", "bajar",
			"pon", "silencia", "pausa", "pausar", "muestra", "mostrar", "maximiza",
			"minimiza", "bloquea", "apaga", "silencio",
		}
		isDirect := false
		for _, imp := range directImperatives {
			if first == imp {
				isDirect = true
				break
			}
		}
		if !isDirect {
			return true
		}
	}

	return false
}

// seedDefaultEngrams inicializa el catálogo de engramas reflejos en memoria.
func (g *SystemGraph) seedDefaultEngrams() {
	defaults := []SystemEngram{
		// --- MULTIMEDIA Y AUDIO ---
		{
			ID:           "audio_mute",
			UseCase:      "multimedia_mute",
			TriggerWords: []string{"bulla", "silencio", "mute", "apaga el sonido", "apaga la bulla", "silencia", "silencia la pc", "calla", "callate", "callar", "ponme en silencio", "mutea"},
			ToolName:     "os_audio_device",
			DefaultArgs:  map[string]any{"action": "mute", "mute": true},
			CoOccurringTools: []string{"os_media_control"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, audio silenciado. ¿Deseas que reactive el sonido más adelante?",
				"He silenciado el sonido de la PC. ¿En qué más te colaboro?",
			},
		},
		{
			ID:           "audio_unmute",
			UseCase:      "multimedia_unmute",
			TriggerWords: []string{"activa el audio", "reactiva sonido", "desmutea", "reactiva el volumen", "activa el sonido", "desactiva el silencio", "quitar silencio"},
			ToolName:     "os_audio_device",
			DefaultArgs:  map[string]any{"action": "mute", "mute": false},
			CoOccurringTools: []string{"os_media_control"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, sonido reactivado. ¿El volumen está bien o necesitas ajustarlo?",
				"Audio restablecido con éxito. ¿Puedo ayudarte con algo más?",
			},
		},
		{
			ID:           "audio_vol_down",
			UseCase:      "multimedia_volume_down",
			TriggerWords: []string{"baja el volumen", "bajale al audio", "bajale a la bulla", "baja el audio", "bajar volumen", "bajale un poco", "menos volumen"},
			ToolName:     "os_audio_device",
			DefaultArgs:  map[string]any{"action": "set_volume", "level": 25.0},
			CoOccurringTools: []string{"os_media_control"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.96,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, volumen reducido. ¿El nivel es adecuado o necesitas que lo baje más?",
				"Volumen bajado exitosamente. ¿En qué más te puedo asistir?",
			},
		},
		{
			ID:           "audio_vol_up",
			UseCase:      "multimedia_volume_up",
			TriggerWords: []string{"sube el volumen", "subele al audio", "sube el audio", "subir volumen", "mas volumen", "subele un poco"},
			ToolName:     "os_audio_device",
			DefaultArgs:  map[string]any{"action": "set_volume", "level": 70.0},
			CoOccurringTools: []string{"os_media_control"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.96,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, volumen aumentado. ¿Está bien a este nivel o lo subo un poco más?",
				"Volumen ajustado hacia arriba. ¿Deseas reproducir algo más?",
			},
		},
		{
			ID:           "media_play_pause",
			UseCase:      "media_playback_toggle",
			TriggerWords: []string{"pausa la musica", "pausar musica", "pausa el video", "pausar spotify", "reanuda la musica", "play musica", "para la musica", "reproducir musica"},
			ToolName:     "os_media_control",
			DefaultArgs:  map[string]any{"action": "play_pause"},
			CoOccurringTools: []string{"os_audio_device"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.97,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, reproducción alternada. ¿Deseas cambiar de pista o necesitas algo más?",
				"Música pausada/reanudada. ¿En qué más te puedo colaborar?",
			},
		},
		{
			ID:           "media_next_track",
			UseCase:      "media_track_next",
			TriggerWords: []string{"siguiente cancion", "cambia de cancion", "pasa de cancion", "siguiente pista", "pasa la cancion"},
			ToolName:     "os_media_control",
			DefaultArgs:  map[string]any{"action": "next"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.97,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, pasando a la siguiente canción. ¿En qué más te puedo colaborar?",
				"Pista cambiada. ¿Deseas hacer algo más?",
			},
		},

		// --- VENTANAS Y ESCRITORIO ---
		{
			ID:           "window_show_desktop",
			UseCase:      "window_desktop_view",
			TriggerWords: []string{"muestra el escritorio", "ver escritorio", "minimiza todo", "minimizar todo", "mostrar escritorio", "limpia la pantalla", "oculta las ventanas"},
			ToolName:     "os_tile_windows",
			DefaultArgs:  map[string]any{"layout": "show_desktop"},
			CoOccurringTools: []string{"os_active_windows"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, he minimizado las ventanas para mostrarte el escritorio. ¿Necesitas abrir alguna aplicación o archivo?",
				"Escritorio visible. ¿En qué más te puedo asistir?",
			},
		},
		{
			ID:           "window_tile_left",
			UseCase:      "window_position_left",
			TriggerWords: []string{"ventana a la izquierda", "pon la ventana a la izquierda", "acomoda a la izquierda", "mitad izquierda", "pega a la izquierda", "pasa la ventana a la izquierda"},
			ToolName:     "os_tile_windows",
			DefaultArgs:  map[string]any{"layout": "left"},
			CoOccurringTools: []string{"os_active_windows"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.97,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, ventana acomodada a la izquierda de la pantalla. ¿Deseas abrir otra aplicación a la derecha?",
			},
		},
		{
			ID:           "window_tile_right",
			UseCase:      "window_position_right",
			TriggerWords: []string{"ventana a la derecha", "pon la ventana a la derecha", "acomoda a la derecha", "mitad derecha", "pega a la derecha", "pasa la ventana a la derecha"},
			ToolName:     "os_tile_windows",
			DefaultArgs:  map[string]any{"layout": "right"},
			CoOccurringTools: []string{"os_active_windows"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.97,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, ventana acomodada a la derecha de la pantalla. ¿En qué más te puedo colaborar?",
			},
		},
		{
			ID:           "window_maximize",
			UseCase:      "window_state_maximize",
			TriggerWords: []string{"maximiza la ventana", "pantalla completa", "maximizar ventana", "maximizar", "agrandar ventana"},
			ToolName:     "os_tile_windows",
			DefaultArgs:  map[string]any{"layout": "maximize"},
			CoOccurringTools: []string{"os_active_windows"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.97,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, ventana maximizada en pantalla completa. ¿Necesitas realizar alguna otra acción?",
			},
		},

		// --- HARDWARE, BATERÍA Y ALMACENAMIENTO ---
		{
			ID:           "hardware_telemetry",
			UseCase:      "hardware_telemetry_query",
			TriggerWords: []string{"cuanta bateria me queda", "estado de bateria", "bateria restante", "nivel de bateria", "cuanta ram me queda", "uso de cpu", "telemetria de hardware"},
			ToolName:     "os_hardware_inspector",
			DefaultArgs:  map[string]any{"action": "telemetry"},
			CoOccurringTools: []string{"os_power_profile"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.97,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Consultando telemetría de hardware en tiempo real...",
			},
		},
		{
			ID:           "hardware_disk_space",
			UseCase:      "hardware_disk_query",
			TriggerWords: []string{"cuanto espacio me queda", "espacio en disco", "disco lleno", "almacenamiento libre", "cuanto espacio libre hay", "espacio del disco c"},
			ToolName:     "os_hardware_inspector",
			DefaultArgs:  map[string]any{"action": "telemetry"},
			CoOccurringTools: []string{"os_disk_cleaner"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.96,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Consultando almacenamiento y espacio libre en discos...",
			},
		},
		{
			ID:           "hardware_health",
			UseCase:      "hardware_smart_health",
			TriggerWords: []string{"salud de la pc", "diagnostico de hardware", "salud del disco", "estado del sistema", "smart del disco", "temperaturas del sistema"},
			ToolName:     "os_hardware_inspector",
			DefaultArgs:  map[string]any{"action": "health"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Auditando salud SMART e integridad de componentes...",
			},
		},
		{
			ID:           "hardware_cam_mic_in_use",
			UseCase:      "hardware_privacy_snoop",
			TriggerWords: []string{"quien usa mi microfono", "camara en uso", "microfono activo", "quien usa la camara", "espia de camara", "camara transmitiendo"},
			ToolName:     "os_hardware_inspector",
			DefaultArgs:  map[string]any{"action": "in_use"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Auditando periféricos activos (cámara y micrófono)...",
			},
		},

		// --- APLICACIONES RÁPIDAS Y CONTROL DE VENTANAS ---
		{
			ID:           "launch_calc",
			UseCase:      "app_launch_calculator",
			TriggerWords: []string{"abre la calculadora", "abrir calculadora", "inicia la calculadora", "abre calc", "abrir calc", "lanzar calculadora", "ejecuta la calculadora"},
			ToolName:     "os_launch_app",
			DefaultArgs:  map[string]any{"appName": "calc"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, he abierto la Calculadora. ¿Deseas realizar alguna operación o necesitas algo más?",
				"Calculadora abierta y lista en pantalla. ¿Qué cálculo deseas realizar?",
			},
		},
		{
			ID:           "close_calc",
			UseCase:      "app_close_calculator",
			TriggerWords: []string{"cierra la calculadora", "cerrar calculadora", "cierrame la calculadora", "cierra calculadora", "apaga la calculadora", "quita la calculadora", "cierra calc", "cerrar calc"},
			ToolName:     "os_close_window",
			DefaultArgs:  map[string]any{"title": "calculadora"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, he cerrado la Calculadora. ¿En qué más te puedo colaborar?",
				"Calculadora cerrada con éxito. ¿Deseas realizar alguna otra acción?",
			},
		},
		{
			ID:           "launch_notepad",
			UseCase:      "app_launch_notepad",
			TriggerWords: []string{"abre el bloc de notas", "abrir bloc de notas", "inicia el bloc de notas", "abre bloc de notas", "abre notepad", "abrir notepad"},
			ToolName:     "os_launch_app",
			DefaultArgs:  map[string]any{"appName": "notepad"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, he abierto el Bloc de notas. ¿Deseas que redacte alguna nota o necesitas algo más?",
				"Bloc de notas listo en pantalla. ¿Qué te gustaría redactar?",
			},
		},
		{
			ID:           "close_notepad",
			UseCase:      "app_close_notepad",
			TriggerWords: []string{"cierra el bloc de notas", "cerrar bloc de notas", "cierra bloc de notas", "cierra notepad", "cerrar notepad", "cierrame el bloc de notas"},
			ToolName:     "os_close_window",
			DefaultArgs:  map[string]any{"title": "bloc de notas"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, he cerrado el Bloc de notas. ¿Necesitas realizar alguna otra tarea?",
				"Bloc de notas cerrado. ¿En qué más te puedo asistir?",
			},
		},
		{
			ID:           "launch_taskmgr",
			UseCase:      "app_launch_taskmgr",
			TriggerWords: []string{"abre el administrador de tareas", "abrir administrador de tareas", "inicia el administrador de tareas", "abre taskmgr", "abrir taskmgr"},
			ToolName:     "os_launch_app",
			DefaultArgs:  map[string]any{"appName": "taskmgr"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, he abierto el Administrador de tareas. ¿Deseas monitorear algún proceso o consumo de recursos?",
			},
		},
		{
			ID:           "close_taskmgr",
			UseCase:      "app_close_taskmgr",
			TriggerWords: []string{"cierra el administrador de tareas", "cerrar administrador de tareas", "cierra administrador de tareas", "cierra taskmgr", "cerrar taskmgr"},
			ToolName:     "os_close_window",
			DefaultArgs:  map[string]any{"title": "administrador de tareas"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.98,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, he cerrado el Administrador de tareas. ¿En qué más te puedo colaborar?",
			},
		},
		{
			ID:           "close_window_active",
			UseCase:      "window_close_active",
			TriggerWords: []string{"cierra la ventana", "cerrar ventana", "cierra esta ventana", "cerrar esta ventana", "cierra ventana actual", "cerrar", "cierra la app", "cerrar app"},
			ToolName:     "os_close_window",
			DefaultArgs:  map[string]any{"title": "activa"},
			FastTrack:    true,
			Destructive:  false,
			Confidence:   0.97,
			Maturity:     "reflex",
			SuccessCount: 10,
			FeedbackTemplates: []string{
				"Listo, he cerrado la ventana en pantalla. ¿Deseas realizar alguna otra acción?",
				"Ventana cerrada exitosamente. ¿En qué más te puedo asistir?",
			},
		},

		// --- ACCIONES DESTRUCTIVAS O DE RIESGO (FastTrack = false obligatorio por Blast Radius) ---
		{
			ID:           "process_kill",
			UseCase:      "process_terminate_guarded",
			TriggerWords: []string{"mata el proceso", "cierra forzado", "terminar proceso", "kill proceso", "finalizar tarea"},
			ToolName:     "os_kill_process",
			DefaultArgs:  map[string]any{"force": true},
			CoOccurringTools: []string{"os_process_sentinel", "os_active_windows"},
			FastTrack:    false, // PROTEGIDO POR BLAST RADIUS: Requiere LLM / CHARC
			Destructive:  true,
			Confidence:   0.95,
			Maturity:     "validated",
			SuccessCount: 1,
			FeedbackTemplates: []string{"Analizando proceso a terminar..."},
		},
		{
			ID:           "file_delete",
			UseCase:      "file_delete_guarded",
			TriggerWords: []string{"borra el archivo", "eliminar archivo", "borrar carpeta", "elimina la carpeta"},
			ToolName:     "os_delete_item",
			DefaultArgs:  map[string]any{"permanent": false},
			CoOccurringTools: []string{"os_explore", "os_file_info"},
			FastTrack:    false, // PROTEGIDO POR BLAST RADIUS: Requiere LLM / CHARC
			Destructive:  true,
			Confidence:   0.95,
			Maturity:     "validated",
			SuccessCount: 1,
			FeedbackTemplates: []string{"Revisando archivo para borrado seguro..."},
		},
	}

	for i := range defaults {
		g.registerEngramInternal(&defaults[i])
	}
}

// registerEngramInternal indexa el engrama en la RAM y actualiza el índice invertido.
func (g *SystemGraph) registerEngramInternal(engram *SystemEngram) {
	// Guardián de Blast Radius: Si es destructivo, jamás puede ser FastTrack
	if engram.Destructive {
		engram.FastTrack = false
	}

	g.engrams[engram.ID] = engram

	for _, phrase := range engram.TriggerWords {
		cleanPhrase := NormalizeColloquialText(phrase)
		tokens := strings.Fields(cleanPhrase)
		for _, token := range tokens {
			if len(token) >= 3 {
				g.invertedIndex[token] = appendUnique(g.invertedIndex[token], engram.ID)
			}
		}
	}
}

func appendUnique(slice []string, val string) []string {
	for _, item := range slice {
		if item == val {
			return slice
		}
	}
	return append(slice, val)
}

// IsRollbackQuery comprueba si el usuario está solicitando deshacer la acción previa.
func (g *SystemGraph) IsRollbackQuery(query string) bool {
	clean := NormalizeColloquialText(query)
	for _, word := range rollbackWords {
		if strings.Contains(clean, word) {
			return true
		}
	}
	return false
}

// RecordRollback guarda un punto de restauración en memoria volátil antes de un FastTrack.
func (g *SystemGraph) RecordRollback(state RollbackState) {
	g.mu.Lock()
	defer g.mu.Unlock()

	state.Timestamp = time.Now()
	g.rollbackBuffer = append(g.rollbackBuffer, state)
	if len(g.rollbackBuffer) > g.maxRollback {
		g.rollbackBuffer = g.rollbackBuffer[1:]
	}
}

// PopRollback obtiene y retira el último estado registrado para su reversión inmediata.
func (g *SystemGraph) PopRollback() (*RollbackState, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.rollbackBuffer) == 0 {
		return nil, false
	}
	idx := len(g.rollbackBuffer) - 1
	last := g.rollbackBuffer[idx]
	g.rollbackBuffer = g.rollbackBuffer[:idx]
	return &last, true
}

// ResolveIntent analiza la consulta del usuario contra los engramas en RAM en microsegundos.
func (g *SystemGraph) ResolveIntent(query string) (*EngramMatch, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	cleanQuery := NormalizeColloquialText(query)
	if cleanQuery == "" {
		return nil, false
	}

	queryTokens := strings.Fields(cleanQuery)

	// Guardián Meta-Conversacional / Feedback: si la consulta es una pregunta, reflexión,
	// reporte de fallo, explicación o diálogo, jamás debe disparar un reflejo FastTrack determinista.
	// Se deriva inmediatamente al modelo cognitivo (LLM) retornando false.
	if IsMetaConversationalQuery(query, cleanQuery, queryTokens) {
		return nil, false
	}

	hasCloseVerb := hasAnyVerb(cleanQuery, closeVerbs)
	hasOpenVerb := hasAnyVerb(cleanQuery, openVerbs)

	candidateScores := make(map[string]float64)

	// 1. Verificación exacta de frases clave con filtro de polaridad de verbos
	for id, engram := range g.engrams {
		// Antonym Polarity Guard: si el usuario pide cerrar, jamás activar os_launch_app
		if hasCloseVerb && engram.ToolName == "os_launch_app" {
			continue
		}
		// Antonym Polarity Guard: si el usuario pide abrir, jamás activar os_close_window
		if hasOpenVerb && engram.ToolName == "os_close_window" {
			continue
		}

		for _, trigger := range engram.TriggerWords {
			cleanTrigger := NormalizeColloquialText(trigger)
			if cleanQuery == cleanTrigger {
				candidateScores[id] += 1.5 // Coincidencia exacta de frase
			} else if strings.Contains(cleanQuery, cleanTrigger) {
				candidateScores[id] += 1.0 // Contención de frase clave
			}
		}
	}

	// 2. Coincidencia por tokens en índice invertido con filtro de polaridad
	for _, token := range queryTokens {
		if len(token) < 3 {
			continue
		}
		if ids, ok := g.invertedIndex[token]; ok {
			for _, id := range ids {
				engram := g.engrams[id]
				if engram == nil {
					continue
				}
				if hasCloseVerb && engram.ToolName == "os_launch_app" {
					continue
				}
				if hasOpenVerb && engram.ToolName == "os_close_window" {
					continue
				}
				candidateScores[id] += 0.25
			}
		}
	}

	if len(candidateScores) == 0 {
		return nil, false
	}

	// Encontrar el mejor candidato
	var bestID string
	var bestScore float64
	for id, score := range candidateScores {
		if score > bestScore {
			bestScore = score
			bestID = id
		}
	}

	bestEngram := g.engrams[bestID]
	if bestEngram == nil || bestScore < 0.60 {
		return nil, false
	}

	// Extracción dinámica de números/parámetros si aplica (ej. volumen o brillo)
	args := make(map[string]any)
	for k, v := range bestEngram.DefaultArgs {
		args[k] = v
	}

	// Si el usuario mencionó un número específico (ej: "volumen al 50", "brillo a 80")
	if nums := numberRegex.FindAllString(cleanQuery, 1); len(nums) > 0 {
		if val, err := strconv.Atoi(nums[0]); err == nil && val >= 0 && val <= 100 {
			if strings.Contains(bestEngram.ToolName, "audio") {
				args["action"] = "set_volume"
				args["level"] = float64(val)
			} else if strings.Contains(bestEngram.ToolName, "power") {
				args["action"] = "set_brightness"
				args["brightness"] = val
			}
		}
	}

	// Selección de feedback variado
	feedback := "Ejecutando acción..."
	if len(bestEngram.FeedbackTemplates) > 0 {
		idx := rand.Intn(len(bestEngram.FeedbackTemplates))
		feedback = bestEngram.FeedbackTemplates[idx]
	}

	isFastTrack := bestEngram.FastTrack && !bestEngram.Destructive && (bestScore >= 0.95 || bestEngram.Maturity == "reflex")

	return &EngramMatch{
		Engram:        bestEngram,
		Score:         bestScore,
		ExtractedArgs: args,
		IsFastTrack:   isFastTrack,
		Feedback:      feedback,
	}, true
}

// GetCoOccurringTools retorna las herramientas compañeras para armar el clúster podado.
func (g *SystemGraph) GetCoOccurringTools(toolNames []string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[string]bool)
	for _, t := range toolNames {
		result[t] = true
		if companions, ok := g.coOccurrence[t]; ok {
			for _, c := range companions {
				result[c] = true
			}
		}
	}

	out := make([]string, 0, len(result))
	for t := range result {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// LearnEngram registra dinámicamente un nuevo engrama enseñado por el usuario o descubierto.
func (g *SystemGraph) LearnEngram(triggerWord, toolName string, args map[string]any, useCase string) (*SystemEngram, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	id := fmt.Sprintf("custom_%s_%d", NormalizeColloquialText(triggerWord), time.Now().Unix())
	id = strings.ReplaceAll(id, " ", "_")

	engram := &SystemEngram{
		ID:                id,
		UseCase:           useCase,
		TriggerWords:      []string{triggerWord},
		ToolName:          toolName,
		DefaultArgs:       args,
		FastTrack:         false, // Inicia como false hasta ser validado
		Destructive:       strings.Contains(toolName, "kill") || strings.Contains(toolName, "delete") || strings.Contains(toolName, "command"),
		Confidence:        0.75,
		Maturity:          "candidate",
		SuccessCount:      1,
		FeedbackTemplates: []string{fmt.Sprintf("Ejecutando %s según lo aprendido...", toolName)},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	g.registerEngramInternal(engram)
	go g.saveToSQLite(engram)

	return engram, nil
}

// PromoteEngram incrementa el contador de éxito de un engrama y lo promueve a reflex si califica.
func (g *SystemGraph) PromoteEngram(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	engram, ok := g.engrams[id]
	if !ok {
		return
	}

	engram.SuccessCount++
	if engram.SuccessCount >= 2 && !engram.Destructive {
		engram.Maturity = "reflex"
		engram.FastTrack = true
		engram.Confidence = 0.98
	} else if engram.SuccessCount >= 1 {
		engram.Maturity = "validated"
	}
	engram.UpdatedAt = time.Now()

	go g.saveToSQLite(engram)
}

// loadFromSQLite carga los engramas aprendidos persistidos en SQLite a la RAM.
func (g *SystemGraph) loadFromSQLite() {
	if db.DB == nil {
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, use_case, trigger_words_json, tool_name, default_args_json,
		       co_occurring_tools_json, fast_track, destructive, confidence, maturity,
		       success_count, feedback_templates_json
		FROM system_engrams
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var e SystemEngram
		var triggersJSON, argsJSON, coToolsJSON, feedbacksJSON string
		var fastTrackInt, destructiveInt int

		err := rows.Scan(
			&e.ID, &e.UseCase, &triggersJSON, &e.ToolName, &argsJSON,
			&coToolsJSON, &fastTrackInt, &destructiveInt, &e.Confidence, &e.Maturity,
			&e.SuccessCount, &feedbacksJSON,
		)
		if err != nil {
			continue
		}

		// Si ya existe un engrama predeterminado en el grafo, no sobrescribir sus
		// disparadores ni plantillas de feedback actualizadas con datos históricos obsoletos de SQLite.
		if existing, ok := g.engrams[e.ID]; ok {
			existing.SuccessCount = e.SuccessCount
			if e.Maturity != "" {
				existing.Maturity = e.Maturity
			}
			go g.saveToSQLite(existing)
			continue
		}

		_ = json.Unmarshal([]byte(triggersJSON), &e.TriggerWords)
		_ = json.Unmarshal([]byte(argsJSON), &e.DefaultArgs)
		_ = json.Unmarshal([]byte(coToolsJSON), &e.CoOccurringTools)
		_ = json.Unmarshal([]byte(feedbacksJSON), &e.FeedbackTemplates)

		e.FastTrack = fastTrackInt == 1
		e.Destructive = destructiveInt == 1

		g.registerEngramInternal(&e)
	}
}

// saveToSQLite persiste un engrama en la base de datos local SQLite.
func (g *SystemGraph) saveToSQLite(e *SystemEngram) {
	if db.DB == nil {
		return
	}

	triggersJSON, _ := json.Marshal(e.TriggerWords)
	argsJSON, _ := json.Marshal(e.DefaultArgs)
	coToolsJSON, _ := json.Marshal(e.CoOccurringTools)
	feedbacksJSON, _ := json.Marshal(e.FeedbackTemplates)

	fastTrackInt := 0
	if e.FastTrack {
		fastTrackInt = 1
	}
	destructiveInt := 0
	if e.Destructive {
		destructiveInt = 1
	}

	query := `
		INSERT INTO system_engrams (
			id, use_case, trigger_words_json, tool_name, default_args_json,
			co_occurring_tools_json, fast_track, destructive, confidence, maturity,
			success_count, feedback_templates_json, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			maturity = excluded.maturity,
			fast_track = excluded.fast_track,
			success_count = excluded.success_count,
			confidence = excluded.confidence,
			updated_at = CURRENT_TIMESTAMP
	`
	_, _ = db.DB.Exec(query,
		e.ID, e.UseCase, string(triggersJSON), e.ToolName, string(argsJSON),
		string(coToolsJSON), fastTrackInt, destructiveInt, e.Confidence, e.Maturity,
		e.SuccessCount, string(feedbacksJSON),
	)
}
