package memory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	"github.com/ozyassist/backend/internal/providers"
)

type MemoryCategory string

const (
	CategoryPreference MemoryCategory = "preferencia"
	CategoryStack      MemoryCategory = "stack"
	CategoryHardware   MemoryCategory = "hardware"
	CategoryRule       MemoryCategory = "regla"
	CategoryContext    MemoryCategory = "contexto"
)

type FactMemory struct {
	ID             string         `json:"id"`
	UserID         string         `json:"user_id"`
	Category       MemoryCategory `json:"category"`
	Content        string         `json:"content"`
	Confidence     float64        `json:"confidence"`
	AccessCount    int            `json:"access_count"`
	LastRecalledAt time.Time      `json:"last_recalled_at"`
	CreatedAt      time.Time      `json:"created_at"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

var (
	defaultStore     *Store
	defaultExtractor *FactExtractor
)

func InitContinuousMemory(db *sql.DB, provider providers.Provider) {
	defaultStore = NewStore(db)
	defaultExtractor = NewFactExtractor(provider, defaultStore)
}

func DefaultStore() *Store {
	return defaultStore
}

func DefaultExtractor() *FactExtractor {
	return defaultExtractor
}

// UpsertFact guarda o actualiza un hecho deduplicado por contenido normalizado
func (s *Store) UpsertFact(ctx context.Context, userID string, cat MemoryCategory, content string, confidence float64) error {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}

	hash := sha256.Sum256([]byte(userID + ":" + string(cat) + ":" + strings.ToLower(trimmed)))
	factID := hex.EncodeToString(hash[:16])

	query := `
	INSERT INTO user_memories (id, user_id, category, content, confidence, access_count, updated_at)
	VALUES (?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP)
	ON CONFLICT(id) DO UPDATE SET
		content = excluded.content,
		confidence = excluded.confidence,
		access_count = user_memories.access_count + 1,
		last_recalled_at = CURRENT_TIMESTAMP,
		updated_at = CURRENT_TIMESTAMP;`

	_, err := s.db.ExecContext(ctx, query, factID, userID, cat, trimmed, confidence)
	return err
}

// SearchRelevant busca recuerdos relevantes para un prompt usando FTS5 BM25
func (s *Store) SearchRelevant(ctx context.Context, userID, query string, limit int) ([]FactMemory, error) {
	if limit <= 0 {
		limit = 5
	}

	sanitizedQuery := sanitizeFTSQuery(query)
	if sanitizedQuery == "" {
		return s.GetRecentFacts(ctx, userID, limit)
	}

	sqlQuery := `
	SELECT m.id, m.user_id, m.category, m.content, m.confidence, m.access_count, m.last_recalled_at, m.created_at
	FROM user_memories m
	JOIN user_memories_fts fts ON m.id = fts.id
	WHERE m.user_id = ? AND user_memories_fts MATCH ?
	ORDER BY bm25(user_memories_fts) ASC, m.access_count DESC
	LIMIT ?;`

	rows, err := s.db.QueryContext(ctx, sqlQuery, userID, sanitizedQuery, limit)
	if err != nil {
		return s.GetRecentFacts(ctx, userID, limit)
	}
	defer rows.Close()

	var facts []FactMemory
	for rows.Next() {
		var f FactMemory
		if err := rows.Scan(&f.ID, &f.UserID, &f.Category, &f.Content, &f.Confidence, &f.AccessCount, &f.LastRecalledAt, &f.CreatedAt); err == nil {
			facts = append(facts, f)
		}
	}
	return facts, nil
}

// GetRecentFacts recupera los hechos más recientes o más recordados
func (s *Store) GetRecentFacts(ctx context.Context, userID string, limit int) ([]FactMemory, error) {
	query := `
	SELECT id, user_id, category, content, confidence, access_count, last_recalled_at, created_at
	FROM user_memories
	WHERE user_id = ?
	ORDER BY access_count DESC, updated_at DESC
	LIMIT ?;`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []FactMemory
	for rows.Next() {
		var f FactMemory
		if err := rows.Scan(&f.ID, &f.UserID, &f.Category, &f.Content, &f.Confidence, &f.AccessCount, &f.LastRecalledAt, &f.CreatedAt); err == nil {
			facts = append(facts, f)
		}
	}
	return facts, nil
}

// GetAllFacts recupera todos los hechos persistidos para un usuario
func (s *Store) GetAllFacts(ctx context.Context, userID string) ([]FactMemory, error) {
	query := `
	SELECT id, user_id, category, content, confidence, access_count, last_recalled_at, created_at
	FROM user_memories
	WHERE user_id = ?
	ORDER BY created_at ASC;`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []FactMemory
	for rows.Next() {
		var f FactMemory
		if err := rows.Scan(&f.ID, &f.UserID, &f.Category, &f.Content, &f.Confidence, &f.AccessCount, &f.LastRecalledAt, &f.CreatedAt); err == nil {
			facts = append(facts, f)
		}
	}
	return facts, nil
}

// DeleteFact elimina un hecho de la memoria continua por su identificador único
func (s *Store) DeleteFact(ctx context.Context, factID string) error {
	query := `DELETE FROM user_memories WHERE id = ?;`
	_, err := s.db.ExecContext(ctx, query, factID)
	return err
}

func sanitizeFTSQuery(raw string) string {
	words := strings.Fields(raw)
	var valid []string
	for _, w := range words {
		cleaned := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				return r
			}
			return -1
		}, w)
		if len(cleaned) > 2 {
			valid = append(valid, cleaned+"*")
		}
	}
	return strings.Join(valid, " OR ")
}
