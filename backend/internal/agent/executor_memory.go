package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
)

// execRememberFact almacena un nuevo hecho o preferencia del usuario en la memoria continua
func execRememberFact(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var args struct {
		Category string `json:"category"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(tc.Input, &args); err != nil {
		return fmt.Sprintf("[MEMORIA] Error en argumentos: %v", err), false
	}
	content := strings.TrimSpace(args.Content)
	if content == "" {
		return "[MEMORIA] El contenido a recordar no puede estar vacío.", false
	}

	cat := memory.MemoryCategory(strings.ToLower(strings.TrimSpace(args.Category)))
	if cat != memory.CategoryPreference && cat != memory.CategoryStack && cat != memory.CategoryHardware && cat != memory.CategoryRule && cat != memory.CategoryContext {
		cat = memory.CategoryContext
	}

	store := memory.DefaultStore()
	if store == nil && db.DB != nil {
		store = memory.NewStore(db.DB)
	}
	if store == nil {
		return "[MEMORIA] Almacén de memoria no inicializado en este momento.", false
	}

	userID := db.DefaultUserID()
	if err := store.UpsertFact(ctx, userID, cat, content, 1.0); err != nil {
		return fmt.Sprintf("[MEMORIA] Error al guardar hecho: %v", err), false
	}

	return fmt.Sprintf("[MEMORIA] Hecho registrado con éxito: [%s] \"%s\"", strings.ToUpper(string(cat)), content), true
}

// execSearchMemory consulta recuerdos previos en la base de memoria episódica/FTS5.
func execSearchMemory(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var args struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(tc.Input, &args); err != nil {
		return fmt.Sprintf("[MEMORIA] Error en argumentos: %v", err), false
	}
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return "[MEMORIA] La consulta de búsqueda no puede estar vacía.", false
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 5
	}

	store := memory.DefaultStore()
	if store == nil && db.DB != nil {
		store = memory.NewStore(db.DB)
	}
	if store == nil {
		return "[MEMORIA] Almacén de memoria no disponible.", false
	}

	userID := db.DefaultUserID()
	facts, err := store.SearchRelevant(ctx, userID, query, limit)
	if err != nil {
		return fmt.Sprintf("[MEMORIA] Error consultando memoria: %v", err), false
	}
	if len(facts) == 0 {
		return fmt.Sprintf("[MEMORIA] No se encontraron recuerdos relevantes para: \"%s\"", query), true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[MEMORIA] Recuerdos recuperados (%d):\n", len(facts)))
	for i, f := range facts {
		sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, strings.ToUpper(string(f.Category)), f.Content))
	}
	return sb.String(), true
}

// execUpdateUserProfile actualiza la ficha Markdown del perfil del usuario en la base de datos.
func execUpdateUserProfile(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var args struct {
		ProfileMd string `json:"profile_md"`
	}
	if err := json.Unmarshal(tc.Input, &args); err != nil {
		return fmt.Sprintf("[PERFIL] Error en argumentos: %v", err), false
	}
	profile := strings.TrimSpace(args.ProfileMd)
	if profile == "" {
		return "[PERFIL] El contenido del perfil no puede estar vacío.", false
	}

	userID := db.DefaultUserID()
	if err := db.UpdateUserProfile(userID, profile); err != nil {
		return fmt.Sprintf("[PERFIL] Error al actualizar perfil: %v", err), false
	}

	return "[PERFIL] Perfil de usuario actualizado y persistido con éxito.", true
}

// execLearnEngram registra un nuevo atajo/engrama en el SystemGraph en RAM y en SQLite
func execLearnEngram(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var args struct {
		TriggerPhrase string         `json:"trigger_phrase"`
		ToolName      string         `json:"tool_name"`
		Args          map[string]any `json:"args"`
		UseCase       string         `json:"use_case"`
	}
	if err := json.Unmarshal(tc.Input, &args); err != nil {
		return fmt.Sprintf("[ENGRAMAS] Error en argumentos: %v", err), false
	}
	trigger := strings.TrimSpace(args.TriggerPhrase)
	tool := strings.TrimSpace(args.ToolName)
	if trigger == "" || tool == "" {
		return "[ENGRAMAS] La frase disparadora y el nombre de herramienta son requeridos.", false
	}

	graph := memory.GetSystemGraph()
	if graph == nil {
		return "[ENGRAMAS] Grafo de sistema no disponible en este momento.", false
	}

	useCase := args.UseCase
	if useCase == "" {
		useCase = "custom_user_shortcut"
	}

	engram, err := graph.LearnEngram(trigger, tool, args.Args, useCase)
	if err != nil {
		return fmt.Sprintf("[ENGRAMAS] Error registrando engrama: %v", err), false
	}

	return fmt.Sprintf("[ENGRAMAS] Neurona aprendida con éxito: ID '%s', disparador: '%s' -> %s", engram.ID, trigger, tool), true
}
