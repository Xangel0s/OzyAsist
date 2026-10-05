package memory

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ozyassist/backend/internal/db"
)

// KnowledgeNode representa un nodo ontológico en el Grafo de Conocimiento del Sistema.
type KnowledgeNode struct {
	ID           string    `json:"id"`
	Label        string    `json:"label"`
	Category     string    `json:"category"`
	Summary      string    `json:"summary"`
	Keywords     []string  `json:"keywords"`
	RelatedTools []string  `json:"related_tools"`
	VectorJSON   string    `json:"vector_json,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// KnowledgeEdge representa una arista direccional ponderada entre dos conceptos o herramientas.
type KnowledgeEdge struct {
	SourceID string  `json:"source_id"`
	TargetID string  `json:"target_id"`
	Relation string  `json:"relation"` // "complements", "requires", "alternative_to", "manages", "executes_on"
	Weight   float64 `json:"weight"`
}

// ScoredKnowledgeNode asocia un nodo con su puntuación de relevancia tras activación expansiva.
type ScoredKnowledgeNode struct {
	Node     *KnowledgeNode
	Score    float64
	OutEdges []*KnowledgeEdge
	InEdges  []*KnowledgeEdge
}

// GraphRAGEngine administra el Grafo de Conocimiento ontológico en memoria RAM pura
// sincronizado con SQLite (tablas knowledge_nodes y knowledge_edges de migración 016).
type GraphRAGEngine struct {
	mu            sync.RWMutex
	nodes         map[string]*KnowledgeNode
	outEdges      map[string][]*KnowledgeEdge
	inEdges       map[string][]*KnowledgeEdge
	invertedIndex map[string][]string // token -> lista de node IDs
}

var (
	graphRAGOnce     sync.Once
	graphRAGInstance *GraphRAGEngine
)

// GetGraphRAGEngine retorna la instancia singleton del motor GraphRAG en RAM.
func GetGraphRAGEngine() *GraphRAGEngine {
	graphRAGOnce.Do(func() {
		graphRAGInstance = &GraphRAGEngine{
			nodes:         make(map[string]*KnowledgeNode),
			outEdges:      make(map[string][]*KnowledgeEdge),
			inEdges:       make(map[string][]*KnowledgeEdge),
			invertedIndex: make(map[string][]string),
		}
		graphRAGInstance.seedDefaultKnowledge()
		graphRAGInstance.loadFromSQLite()
	})
	return graphRAGInstance
}

// AddNode registra un nuevo nodo en el grafo y actualiza el índice invertido.
func (g *GraphRAGEngine) AddNode(node *KnowledgeNode) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodes[node.ID] = node

	// Indexar tokens del label, categoría, keywords y resumen
	textToIndex := fmt.Sprintf("%s %s %s %s", node.Label, node.Category, strings.Join(node.Keywords, " "), strings.Join(node.RelatedTools, " "))
	tokens := tokenizeText(textToIndex)
	for _, tok := range tokens {
		g.invertedIndex[tok] = appendUnique(g.invertedIndex[tok], node.ID)
	}
}

// AddEdge agrega una arista direccional ponderada.
func (g *GraphRAGEngine) AddEdge(edge *KnowledgeEdge) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.outEdges[edge.SourceID] = append(g.outEdges[edge.SourceID], edge)
	g.inEdges[edge.TargetID] = append(g.inEdges[edge.TargetID], edge)
}

func tokenizeText(text string) []string {
	clean := NormalizeColloquialText(text)
	raw := strings.Fields(clean)
	var tokens []string
	for _, w := range raw {
		if len(w) >= 3 {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

// QueryGraphRAG ejecuta búsqueda híbrida en RAM (Tokens + Difusión Expansiva / Spreading Activation)
// y retorna una síntesis estructurada para el contexto del LLM.
func (g *GraphRAGEngine) QueryGraphRAG(query string, topK int) string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if topK <= 0 {
		topK = 3
	}

	cleanQuery := NormalizeColloquialText(query)
	queryTokens := tokenizeText(cleanQuery)
	if len(queryTokens) == 0 {
		return "No se especificaron términos de consulta válidos para el Grafo de Conocimiento."
	}

	// 1. Scoring Inicial por Coincidencia Léxica / Semántica
	nodeScores := make(map[string]float64)
	for _, tok := range queryTokens {
		if nodeIDs, ok := g.invertedIndex[tok]; ok {
			for _, id := range nodeIDs {
				node := g.nodes[id]
				if node == nil {
					continue
				}

				// Ponderación por campo
				weight := 1.0
				if strings.Contains(strings.ToLower(node.Label), tok) {
					weight += 2.0
				}
				if strings.Contains(strings.ToLower(node.Category), tok) {
					weight += 1.5
				}
				for _, kw := range node.Keywords {
					if strings.Contains(strings.ToLower(kw), tok) {
						weight += 1.2
					}
				}
				for _, rt := range node.RelatedTools {
					if strings.Contains(strings.ToLower(rt), tok) {
						weight += 2.5
					}
				}

				nodeScores[id] += weight
			}
		}
	}

	if len(nodeScores) == 0 {
		return fmt.Sprintf("ℹ️ No se hallaron nodos de conocimiento específicos para %q en el Grafo de Conocimiento.", query)
	}

	// 2. Activación Expansiva (Spreading Activation) de 1 Hop
	// Propaga la energía de relevancia a través de aristas con factor de decaimiento alpha = 0.5
	decayAlpha := 0.5
	activatedScores := make(map[string]float64)
	for id, score := range nodeScores {
		activatedScores[id] += score
		// Propagar por aristas salientes
		for _, edge := range g.outEdges[id] {
			activatedScores[edge.TargetID] += score * edge.Weight * decayAlpha
		}
		// Propagar por aristas entrantes
		for _, edge := range g.inEdges[id] {
			activatedScores[edge.SourceID] += score * edge.Weight * (decayAlpha * 0.7)
		}
	}

	// 3. Ordenar y seleccionar los mejores Top K
	type candidate struct {
		id    string
		score float64
	}
	var ranking []candidate
	for id, score := range activatedScores {
		if _, exists := g.nodes[id]; exists {
			ranking = append(ranking, candidate{id: id, score: score})
		}
	}
	sort.Slice(ranking, func(i, j int) bool {
		return ranking[i].score > ranking[j].score
	})

	if len(ranking) > topK {
		ranking = ranking[:topK]
	}

	// 4. Sintetizar respuesta Markdown compacta para el LLM
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🧠 === CONOCIMIENTO ONTOLÓGICO DEL SISTEMA (Top %d) ===\n", len(ranking)))
	sb.WriteString(fmt.Sprintf("Consulta: %s\n\n", query))

	for i, c := range ranking {
		node := g.nodes[c.id]
		sb.WriteString(fmt.Sprintf("### %d. %s [%s] (Relevancia: %.2f)\n", i+1, node.Label, strings.ToUpper(node.Category), c.score))
		sb.WriteString(fmt.Sprintf("• **Resumen & Reglas Técnicas**: %s\n", node.Summary))
		if len(node.RelatedTools) > 0 {
			sb.WriteString(fmt.Sprintf("• **Herramientas Asociadas**: `%s`\n", strings.Join(node.RelatedTools, "`, `")))
		}

		// Relaciones ontológicas salientes
		var rels []string
		for _, edge := range g.outEdges[c.id] {
			if target, ok := g.nodes[edge.TargetID]; ok {
				rels = append(rels, fmt.Sprintf("%s -> %s (peso: %.1f)", edge.Relation, target.Label, edge.Weight))
			}
		}
		if len(rels) > 0 {
			sb.WriteString(fmt.Sprintf("• **Relaciones**: %s\n", strings.Join(rels, " | ")))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("💡 *Directiva para el LLM*: Emplea las herramientas y directivas descritas sin alucinar parámetros no contemplados en el grafo.\n")
	return sb.String()
}

// seedDefaultKnowledge precarga el grafo con el conocimiento central del sistema OzyAssist.
func (g *GraphRAGEngine) seedDefaultKnowledge() {
	defaultNodes := []*KnowledgeNode{
		{
			ID:       "node_audio_wasapi",
			Label:    "Subtecnología de Audio WASAPI y Multimedia",
			Category: "multimedia",
			Summary:  "Control nativo de volumen maestro y mute mediante interfaz WASAPI IAudioEndpointVolume en Go puro. Soporta niveles 0-100, mute/unmute y control multimedia virtual de reproducción (play/pause, next, prev) sin depender de PowerShell.",
			Keywords: []string{"audio", "volumen", "sonido", "bulla", "silencio", "mute", "altavoz", "musica", "spotify", "wasapi"},
			RelatedTools: []string{"os_audio_device", "os_media_control"},
		},
		{
			ID:       "node_camera_rust",
			Label:    "Motor Nativo de Cámara en Rust (ozy-core)",
			Category: "vision",
			Summary:  "Captura visual ultrarrápida en microsegundos usando ozy-core.exe compilado en Rust con nokhwa y MediaFoundation (MSMF). Guarda fotogramas PNG/JPEG en ~/.ozy/captures/ sin congelar el hilo principal ni requerir OpenCV para captura.",
			Keywords: []string{"camara", "camera", "foto", "captura", "webcam", "video", "nokhwa", "rust", "vision", "dispositivo de video"},
			RelatedTools: []string{"os_camera_capture", "os_camera_list", "os_analyze_screen"},
		},
		{
			ID:       "node_python_workspace",
			Label:    "Mesa de Trabajo Python 3.11",
			Category: "python",
			Summary:  "Entorno de ejecución aislado en ~/.ozy/workspace/ ejecutando Python 3.11.9 nativo. Incluye pandas, openpyxl, pillow, opencv, pymupdf y requests. Posee bucle Self-Healing: si falta un módulo (ModuleNotFoundError), ejecuta pip install silenciosamente y reintenta.",
			Keywords: []string{"python", "script", "workspace", "datos", "excel", "pandas", "calculo", "grafica", "analisis", "codigo"},
			RelatedTools: []string{"os_python_exec", "os_workspace_list"},
		},
		{
			ID:       "node_mcts_planner",
			Label:    "Motor de Búsqueda Monte Carlo Tree Search (MCTS)",
			Category: "planning",
			Summary:  "Planificador cognitivo basado en UCB1 ejecutado en Python/Go para simular árboles de acción, resolver trayectorias de recuperación ante errores de herramientas y elegir la ruta óptima minimizando el Blast Radius y tiempo de ejecución.",
			Keywords: []string{"mcts", "monte carlo", "arbol", "decision", "recuperacion", "estrategia", "trayectoria", "error recovery", "ucb1"},
			RelatedTools: []string{"nine_mcts_solve", "nine_strategic_plan"},
		},
		{
			ID:       "node_process_control",
			Label:    "Centinela de Procesos y Ventanas Win32",
			Category: "process",
			Summary:  "Administración de alta integridad para ventanas y procesos de Windows. Cierre elegante inicial vía WM_CLOSE (0x0010) al HWND. Terminación forzada multinivel con fallback a CIM/WMI para procesos elevados con permisos de administrador.",
			Keywords: []string{"proceso", "cerrar", "matar", "kill", "taskmgr", "administrador de tareas", "ventana", "colgado", "responder", "memoria", "cpu"},
			RelatedTools: []string{"os_close_window", "os_kill_process", "os_process_sentinel", "os_active_windows"},
		},
		{
			ID:       "node_windows_services",
			Label:    "Controlador de Servicios de Windows y ShellExecute",
			Category: "system",
			Summary:  "Consulta y control de servicios de Windows (list, status, start, stop, restart). Invocación nativa procShellExecuteW para arrancar aplicaciones elevadas (ej. Taskmgr, Configuración) sin errores de elevación 740.",
			Keywords: []string{"servicio", "service", "arrancar", "detener", "daemon", "elevado", "administrador", "shell", "ejecutar"},
			RelatedTools: []string{"os_service_manager", "os_launch_app", "run_command"},
		},
		{
			ID:       "node_hardware_telemetry",
			Label:    "Telemetría de Hardware y Auditoría de Privacidad",
			Category: "hardware",
			Summary:  "Inspección profunda de CPU, RAM, GPU (nvidia-smi), temperaturas térmicas ACPI, salud SMART de discos y auditoría de privacidad en tiempo real: reporta si la cámara web o micrófono están en uso activo y qué PID los ocupa.",
			Keywords: []string{"hardware", "ram", "cpu", "bateria", "disco", "smart", "temperatura", "camara en uso", "microfono activo", "espia", "privacidad"},
			RelatedTools: []string{"os_hardware_inspector", "os_power_profile", "os_disk_cleaner"},
		},
		{
			ID:       "node_tiling_windows",
			Label:    "Acomodador y División de Ventanas en Pantalla",
			Category: "desktop",
			Summary:  "División dinámica de ventanas (left, right, maximize, minimize, restore, show_desktop) calculando dinámicamente el área de trabajo del monitor (SPI_GETWORKAREA) con APIs MoveWindow de Win32.",
			Keywords: []string{"acomodar", "pantalla", "izquierda", "derecha", "maximizar", "minimizar", "escritorio", "tiling", "organizar ventanas"},
			RelatedTools: []string{"os_tile_windows", "os_focus_window", "os_active_windows"},
		},
		{
			ID:       "node_document_engines",
			Label:    "Motores Documentales Nativos Pure-Go (.docx y .pdf)",
			Category: "documents",
			Summary:  "Generación de documentos Word (.docx) compatibles con Office 365 / Google Docs mediante empaquetado OpenXML ZIP puro en Go, y documentos PDF profesionales con gofpdf. Cumple 100%% la directiva Zero-Docker.",
			Keywords: []string{"docx", "word", "pdf", "reporte", "documento", "convertir", "generar pdf", "openxml"},
			RelatedTools: []string{"os_create_docx", "os_create_pdf", "os_convert_to_pdf"},
		},
		{
			ID:       "node_network_diagnostics",
			Label:    "Diagnóstico de Red, Wi-Fi e Inspección de Puertos",
			Category: "network",
			Summary:  "Auditoría de puertos TCP en escucha con PID y proceso asociado, pruebas de conectividad ICMP con cálculo de latencia en ms, telemetría Wi-Fi en vivo (SSID, señal %, canal) y vaciado de caché DNS.",
			Keywords: []string{"red", "puerto", "puertos", "wifi", "dns", "ping", "latencia", "conexion", "internet", "tcp"},
			RelatedTools: []string{"os_port_inspector", "os_network_diagnostics", "os_wifi_manager"},
		},
		{
			ID:       "node_nine_strategist",
			Label:    "Estratega Cognitivo NINE",
			Category: "cognition",
			Summary:  "Subagente de razonamiento de alto nivel. Descompone objetivos complejos, formula planes estratégicos multi-etapa, genera scripts Python analíticos en la mesa de trabajo y coordina al auditor CHARC y al ejecutor OZY.",
			Keywords: []string{"nine", "estrategia", "plan", "razonamiento", "complejo", "pensamiento", "subagente"},
			RelatedTools: []string{"nine_strategic_plan", "nine_mcts_solve"},
		},
		{
			ID:       "node_system_graph_engrams",
			Label:    "Red de Engramas Reflejos y Fast-Track",
			Category: "cognition",
			Summary:  "Memoria refleja en RAM que asocia lenguaje coloquial con acciones deterministas sin latencia de LLM (~0 ms). Protegido por Guardián de Blast Radius: acciones destructivas (kill, delete) jamás son Fast-Track.",
			Keywords: []string{"engrama", "fasttrack", "reflejo", "automatizacion", "blast radius", "rollback", "revertir"},
			RelatedTools: []string{"learn_engram"},
		},
	}

	for _, n := range defaultNodes {
		g.AddNode(n)
	}

	defaultEdges := []*KnowledgeEdge{
		{SourceID: "node_camera_rust", TargetID: "node_hardware_telemetry", Relation: "inspects_device", Weight: 0.95},
		{SourceID: "node_mcts_planner", TargetID: "node_nine_strategist", Relation: "search_engine_for", Weight: 1.0},
		{SourceID: "node_mcts_planner", TargetID: "node_python_workspace", Relation: "executes_on", Weight: 0.95},
		{SourceID: "node_process_control", TargetID: "node_windows_services", Relation: "system_management", Weight: 0.85},
		{SourceID: "node_audio_wasapi", TargetID: "node_hardware_telemetry", Relation: "endpoint_telemetry", Weight: 0.80},
		{SourceID: "node_python_workspace", TargetID: "node_document_engines", Relation: "data_to_doc", Weight: 0.85},
		{SourceID: "node_tiling_windows", TargetID: "node_process_control", Relation: "window_target", Weight: 0.90},
		{SourceID: "node_nine_strategist", TargetID: "node_system_graph_engrams", Relation: "coordinates", Weight: 0.90},
		{SourceID: "node_network_diagnostics", TargetID: "node_process_control", Relation: "correlates_pid", Weight: 0.90},
	}

	for _, e := range defaultEdges {
		g.AddEdge(e)
	}
}

// loadFromSQLite lee nodos y aristas persistidos en SQLite para enriquecer la RAM.
func (g *GraphRAGEngine) loadFromSQLite() {
	if db.DB == nil {
		return
	}

	// 1. Cargar nodos
	rows, err := db.DB.Query("SELECT id, label, category, summary, vector_json FROM knowledge_nodes")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, label, category, summary string
			var vecJSON sql.NullString
			if err := rows.Scan(&id, &label, &category, &summary, &vecJSON); err == nil {
				if existing, ok := g.nodes[id]; ok {
					existing.Summary = summary
					existing.Label = label
					existing.Category = category
				} else {
					g.AddNode(&KnowledgeNode{
						ID:         id,
						Label:      label,
						Category:   category,
						Summary:    summary,
						VectorJSON: vecJSON.String,
						CreatedAt:  time.Now(),
					})
				}
			}
		}
	}

	// 2. Cargar aristas
	edgeRows, err := db.DB.Query("SELECT source_id, target_id, relation, weight FROM knowledge_edges")
	if err == nil {
		defer edgeRows.Close()
		for edgeRows.Next() {
			var src, tgt, rel string
			var weight float64
			if err := edgeRows.Scan(&src, &tgt, &rel, &weight); err == nil {
				g.AddEdge(&KnowledgeEdge{
					SourceID: src,
					TargetID: tgt,
					Relation: rel,
					Weight:   weight,
				})
			}
		}
	}

	// Sincronizar hacia SQLite los nodos sembrados por defecto si no existen
	go g.syncDefaultsToSQLite()
}

func (g *GraphRAGEngine) syncDefaultsToSQLite() {
	if db.DB == nil {
		return
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, node := range g.nodes {
		query := `INSERT OR IGNORE INTO knowledge_nodes (id, label, category, summary, vector_json)
		          VALUES (?, ?, ?, ?, ?)`
		_, _ = db.DB.Exec(query, node.ID, node.Label, node.Category, node.Summary, node.VectorJSON)
	}

	for _, edges := range g.outEdges {
		for _, e := range edges {
			query := `INSERT OR IGNORE INTO knowledge_edges (source_id, target_id, relation, weight)
			          VALUES (?, ?, ?, ?)`
			_, _ = db.DB.Exec(query, e.SourceID, e.TargetID, e.Relation, e.Weight)
		}
	}
	log.Println("[GraphRAG Engine] Sincronización ontológica en SQLite completa.")
}
