package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ozyassist/backend/internal/providers"
)

// DreamResult resume el resultado del ciclo de consolidación cognitiva
type DreamResult struct {
	ConsolidatedFactsCount int      `json:"consolidated_facts_count"`
	PurgedFactsCount       int      `json:"purged_facts_count"`
	UpdatedProfileMd       string   `json:"updated_profile_md"`
	PurgedFactIDs          []string `json:"purged_fact_ids"`
	Summary                string   `json:"summary"`
}

type dreamLLMResponse struct {
	UpdatedProfileMd string   `json:"updated_profile_md"`
	ObsoleteFactIDs  []string `json:"obsolete_fact_ids"`
	Summary          string   `json:"summary"`
}

// DreamerSubagent es el subagente cognitivo asíncrono para consolidación de memoria
type DreamerSubagent struct {
	provider providers.Provider
	store    *Store
	db       *sql.DB
	cache    KVStore
	mu       sync.Mutex
	isBusy   bool
}

var (
	defaultDreamer   *DreamerSubagent
	defaultDreamerMu sync.RWMutex
)

// InitDreamerSubagent inicializa la instancia global del subagente DREAMER
func InitDreamerSubagent(provider providers.Provider, store *Store, db *sql.DB, cache KVStore) *DreamerSubagent {
	defaultDreamerMu.Lock()
	defer defaultDreamerMu.Unlock()
	defaultDreamer = NewDreamerSubagent(provider, store, db, cache)
	return defaultDreamer
}

// DefaultDreamer retorna la instancia global del subagente DREAMER
func DefaultDreamer() *DreamerSubagent {
	defaultDreamerMu.RLock()
	defer defaultDreamerMu.RUnlock()
	return defaultDreamer
}

// NewDreamerSubagent crea una nueva instancia de DreamerSubagent
func NewDreamerSubagent(provider providers.Provider, store *Store, db *sql.DB, cache KVStore) *DreamerSubagent {
	if cache == nil {
		cache = DefaultCache()
	}
	return &DreamerSubagent{
		provider: provider,
		store:    store,
		db:       db,
		cache:    cache,
	}
}

// DreamAsync lanza el proceso de consolidación en segundo plano sin bloquear la UI
func (d *DreamerSubagent) DreamAsync(ctx context.Context, userID string, onProgress func(msg string)) {
	go func() {
		if onProgress != nil {
			onProgress("[DREAMING] Iniciando ciclo de consolidacion de memoria...")
		}
		res, err := d.Dream(context.Background(), userID)
		if err != nil {
			if onProgress != nil {
				onProgress(fmt.Sprintf("[DREAMING] [ERROR] Consolidacion abortada: %v", err))
			}
			log.Printf("[DREAMING] [ERROR] Consolidación abortada: %v", err)
			return
		}
		if onProgress != nil {
			onProgress(fmt.Sprintf("[DREAMING] [OK] %s", res.Summary))
		}
	}()
}

// Dream ejecuta sincrónicamente la consolidación cognitiva de recuerdos
func (d *DreamerSubagent) Dream(ctx context.Context, userID string) (*DreamResult, error) {
	d.mu.Lock()
	if d.isBusy {
		d.mu.Unlock()
		return nil, fmt.Errorf("el subagente DREAMER ya esta en ejecucion")
	}
	d.isBusy = true
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		d.isBusy = false
		d.mu.Unlock()
	}()

	if d.store == nil || d.db == nil {
		return nil, fmt.Errorf("almacen o base de datos no inicializados")
	}

	// 1. Recuperar perfil actual
	var currentProfileMd string
	err := d.db.QueryRowContext(ctx, `SELECT COALESCE(profile_md, '') FROM users WHERE id = ?`, userID).Scan(&currentProfileMd)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("error leyendo perfil de usuario: %w", err)
	}

	// 2. Recuperar todos los hechos
	facts, err := d.store.GetAllFacts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error consultando hechos de usuario: %w", err)
	}

	if len(facts) == 0 && strings.TrimSpace(currentProfileMd) == "" {
		return &DreamResult{
			Summary: "No hay hechos ni perfil previo para consolidar.",
		}, nil
	}

	// 3. Formular prompt cognitivo para síntesis
	var factsBuilder strings.Builder
	for i, f := range facts {
		fmt.Fprintf(&factsBuilder, "- [ID: %s] [%s] (confianza: %.2f, accesos: %d): %s\n",
			f.ID, f.Category, f.Confidence, f.AccessCount, f.Content)
		if i >= 100 { // Limitar a los 100 hechos más relevantes para respetar contexto
			break
		}
	}

	prompt := fmt.Sprintf(`Eres DREAMER, el subagente cognitivo de consolidación de memoria continua de OzyAssist.
Tu objetivo es analizar los hechos aprendidos del usuario y el perfil actual, detectar contradicciones, eliminar hechos obsoletos y redactar una ficha de perfil Markdown perfectamente estructurada y concisa.

=== PERFIL DE USUARIO ACTUAL ===
%s

=== HECHOS REGISTRADOS EN MEMORIA CONTINUA ===
%s

INSTRUCCIONES CLAVE:
1. Resuelve contradicciones: si un hecho contradice a otro (ej. cambio de versión de Go, cambio de IDE principal), prevalece el más reciente o con mayor nivel de confianza/accesos.
2. Genera una ficha Markdown elegante y estructurada con las secciones:
   # Perfil del Usuario
   ## Rol y Enfoque Profesional
   ## Stack Técnico y Herramientas
   ## Hardware y Entorno
   ## Reglas y Directivas de Código
   ## Proyectos Clave y Contexto
3. Si un hecho registrado queda obsoleto, contradictorio o ya está totalmente absorbido y sintetizado de forma permanente en el perfil Markdown, incluye su ID en "obsolete_fact_ids" para purgarlo de la base de datos atómica.
4. Responde ÚNICAMENTE en JSON válido con el siguiente formato:
{
  "updated_profile_md": "# Perfil del Usuario...",
  "obsolete_fact_ids": ["id_a_eliminar_1", "id_a_eliminar_2"],
  "summary": "Consolidación exitosa: integrados X hechos en el perfil, depurados Y hechos."
}`, currentProfileMd, factsBuilder.String())

	llmText, err := d.queryLLM(ctx, prompt, 0.1, 1500)
	if err != nil {
		return nil, fmt.Errorf("fallo consulta a LLM en dreaming: %w", err)
	}

	var resp dreamLLMResponse
	cleanText := strings.TrimSpace(llmText)
	if start := strings.Index(cleanText, "{"); start != -1 {
		if end := strings.LastIndex(cleanText, "}"); end != -1 && end > start {
			if err := json.Unmarshal([]byte(cleanText[start:end+1]), &resp); err != nil {
				return nil, fmt.Errorf("error parseando JSON de dreaming: %w", err)
			}
		}
	}

	if resp.UpdatedProfileMd == "" {
		return nil, fmt.Errorf("el modelo no retorno un perfil consolidado valido")
	}

	// 4. Actualizar perfil en SQLite
	_, err = d.db.ExecContext(ctx, `UPDATE users SET profile_md = ? WHERE id = ?`, resp.UpdatedProfileMd, userID)
	if err != nil {
		return nil, fmt.Errorf("error persistiendo perfil actualizado en db: %w", err)
	}

	// 5. Actualizar en caché RAM
	if d.cache != nil {
		_ = d.cache.Set(ctx, "user_profile:"+userID, resp.UpdatedProfileMd, 24*time.Hour)
	}

	// 6. Purgar hechos obsoletos
	purgedCount := 0
	for _, id := range resp.ObsoleteFactIDs {
		if err := d.store.DeleteFact(ctx, id); err == nil {
			purgedCount++
		}
	}

	res := &DreamResult{
		ConsolidatedFactsCount: len(facts),
		PurgedFactsCount:       purgedCount,
		UpdatedProfileMd:       resp.UpdatedProfileMd,
		PurgedFactIDs:          resp.ObsoleteFactIDs,
		Summary:                resp.Summary,
	}

	if res.Summary == "" {
		res.Summary = fmt.Sprintf("Consolidados %d hechos en el perfil. Purgados %d hechos obsoletos.", len(facts), purgedCount)
	}

	log.Printf("[DREAMING] [OK] %s", res.Summary)
	return res, nil
}

func (d *DreamerSubagent) queryLLM(ctx context.Context, prompt string, temp float64, maxTokens int) (string, error) {
	if d.provider == nil {
		return "", fmt.Errorf("llm provider no disponible")
	}

	messages := []providers.Message{
		{Role: "user", Content: prompt},
	}

	chunkCh, err := d.provider.StreamCompletion(ctx, messages, providers.CompletionOptions{
		Temperature: temp,
		MaxTokens:   maxTokens,
		Stream:      false,
	})
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for chunk := range chunkCh {
		if chunk.Type == "text" || chunk.Type == "" {
			sb.WriteString(chunk.Content)
		}
	}

	return sb.String(), nil
}
