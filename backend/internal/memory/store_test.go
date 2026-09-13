package memory

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
	"github.com/ozyassist/backend/internal/providers"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Error abriendo SQLite en memoria: %v", err)
	}

	migration := `
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

	CREATE VIRTUAL TABLE user_memories_fts USING fts5(
		id UNINDEXED,
		content,
		category,
		tokenize = 'porter unicode61'
	);

	CREATE TRIGGER trg_user_memories_ai AFTER INSERT ON user_memories BEGIN
		INSERT INTO user_memories_fts(id, content, category) VALUES (new.id, new.content, new.category);
	END;`

	if _, err := db.Exec(migration); err != nil {
		t.Fatalf("Error aplicando migración de prueba: %v", err)
	}
	return db
}

func TestStore_UpsertAndFTS(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)
	ctx := context.Background()

	err := store.UpsertFact(ctx, "usr_1", CategoryStack, "Desarrolla en Go con arquitectura hexagonal", 0.95)
	if err != nil {
		t.Fatalf("Error al guardar hecho: %v", err)
	}

	// Búsqueda por palabra clave FTS5
	results, err := store.SearchRelevant(ctx, "usr_1", "hexagonal architecture in Go", 5)
	if err != nil {
		t.Fatalf("Fallo en búsqueda FTS5: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("Esperaba encontrar 1 recuerdo indexado por FTS5")
	}

	if results[0].Category != CategoryStack {
		t.Errorf("Categoría incorrecta: %s", results[0].Category)
	}
}

type mockMemoryProvider struct {
	response string
}

func (m *mockMemoryProvider) StreamCompletion(ctx context.Context, messages []providers.Message, opts providers.CompletionOptions) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk, 2)
	ch <- providers.StreamChunk{Type: "text", Content: m.response}
	ch <- providers.StreamChunk{Type: "done"}
	close(ch)
	return ch, nil
}

func (m *mockMemoryProvider) Name() string        { return "mock-memory" }
func (m *mockMemoryProvider) SupportsTools() bool { return false }
func (m *mockMemoryProvider) Models() []string    { return []string{"mock-model"} }

func TestFactExtractor_ExtractAndPersist(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)
	mockResp := `[{"category": "preferencia", "content": "Prefiere café sin azúcar", "confidence": 0.9}]`
	extractor := NewFactExtractor(&mockMemoryProvider{response: mockResp}, store)

	err := extractor.ExtractAndPersist(context.Background(), "usr_1", "Recuerda que siempre tomo café sin azúcar", "Entendido")
	if err != nil {
		t.Fatalf("Fallo en extractor: %v", err)
	}

	facts, _ := store.GetRecentFacts(context.Background(), "usr_1", 5)
	if len(facts) != 1 || facts[0].Content != "Prefiere café sin azúcar" {
		t.Fatalf("Hecho no persistido correctamente: %+v", facts)
	}
}
