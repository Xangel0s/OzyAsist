package memory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
	"github.com/ozyassist/backend/internal/providers"
)

type mockDreamerProvider struct {
	response string
}

func (m *mockDreamerProvider) Name() string { return "mock-dreamer" }

func (m *mockDreamerProvider) SupportsTools() bool { return false }

func (m *mockDreamerProvider) StreamCompletion(ctx context.Context, messages []providers.Message, opts providers.CompletionOptions) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk, 1)
	ch <- providers.StreamChunk{Type: "text", Content: m.response}
	close(ch)
	return ch, nil
}

func (m *mockDreamerProvider) Models() []string { return []string{"mock"} }

func setupDreamerTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("error abriendo sqlite en memoria: %v", err)
	}

	schema := `
	CREATE TABLE users (
		id TEXT PRIMARY KEY,
		name TEXT,
		profile_md TEXT DEFAULT ''
	);

	CREATE TABLE user_memories (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		category TEXT NOT NULL,
		content TEXT NOT NULL,
		confidence REAL DEFAULT 1.0,
		access_count INTEGER DEFAULT 1,
		last_recalled_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	INSERT INTO users (id, name, profile_md) VALUES ('user-1', 'Lenovo User', '# Perfil Antiguo');
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("error creando esquema de test: %v", err)
	}

	return db
}

func TestDreamerSubagent_DreamSuccess(t *testing.T) {
	db := setupDreamerTestDB(t)
	defer db.Close()

	store := NewStore(db)
	cache := NewMemoryKVStore()
	defer cache.Close()

	ctx := context.Background()

	// Insertar hechos de prueba
	_ = store.UpsertFact(ctx, "user-1", CategoryStack, "Usa Go 1.22", 0.9)
	_ = store.UpsertFact(ctx, "user-1", CategoryStack, "Migrado a Go 1.26 con Bubble Tea", 0.98)

	facts, _ := store.GetAllFacts(ctx, "user-1")
	if len(facts) != 2 {
		t.Fatalf("se esperaban 2 hechos, se obtuvieron: %d", len(facts))
	}

	oldFactID := facts[0].ID

	// Mock LLM consolidando perfil y marcando oldFactID como obsoleto
	mockResponse := fmt.Sprintf(`{
		"updated_profile_md": "# Perfil del Usuario: Lenovo User\n## Stack\n- Go 1.26\n- Bubble Tea TUI",
		"obsolete_fact_ids": ["%s"],
		"summary": "Consolidados 2 hechos. Purgada referencia obsoleta a Go 1.22."
	}`, oldFactID)

	provider := &mockDreamerProvider{response: mockResponse}
	dreamer := NewDreamerSubagent(provider, store, db, cache)

	res, err := dreamer.Dream(ctx, "user-1")
	if err != nil {
		t.Fatalf("error en Dream: %v", err)
	}

	if res.PurgedFactsCount != 1 {
		t.Fatalf("se esperaba 1 hecho purgado, obtuvo: %d", res.PurgedFactsCount)
	}

	// Verificar actualización en DB de users
	var updatedProfile string
	_ = db.QueryRow("SELECT profile_md FROM users WHERE id = 'user-1'").Scan(&updatedProfile)
	if !strings.Contains(updatedProfile, "Go 1.26") {
		t.Fatalf("el perfil en DB no se actualizó correctamente: %s", updatedProfile)
	}

	// Verificar actualización en Cache
	cachedProfile, found, _ := cache.Get(ctx, "user_profile:user-1")
	if !found || !strings.Contains(cachedProfile, "Bubble Tea TUI") {
		t.Fatalf("el cache no contiene el perfil actualizado: %v, %s", found, cachedProfile)
	}

	// Verificar que el hecho obsoleto fue eliminado de user_memories
	remainingFacts, _ := store.GetAllFacts(ctx, "user-1")
	if len(remainingFacts) != 1 {
		t.Fatalf("debía quedar solo 1 hecho, quedaron: %d", len(remainingFacts))
	}
	if remainingFacts[0].ID == oldFactID {
		t.Fatalf("el hecho antiguo debía haber sido eliminado")
	}
}
