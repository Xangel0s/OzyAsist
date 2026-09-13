package security

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

type AuditEntry struct {
	ID         int64     `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Agent      string    `json:"agent"`
	Action     string    `json:"action"`
	Details    string    `json:"details"`
	PrevHash   string    `json:"prev_hash"`
	RecordHash string    `json:"record_hash"`
}

type AuditLogger struct {
	db *sql.DB
	mu sync.Mutex
}

func NewAuditLogger(db *sql.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// CalculateRecordHash genera el hash SHA-256 inmutable de un registro
func CalculateRecordHash(prevHash string, ts time.Time, agent, action, details string) string {
	payload := fmt.Sprintf("%s|%s|%s|%s|%s", prevHash, ts.UTC().Format(time.RFC3339Nano), agent, action, details)
	hash := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(hash[:])
}

// Record registra una acción en la cadena de bloques interna de auditoría
func (l *AuditLogger) Record(ctx context.Context, agent, action, details string) (*AuditEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var prevHash string
	err := l.db.QueryRowContext(ctx, `SELECT record_hash FROM audit_trail_chained ORDER BY id DESC LIMIT 1`).Scan(&prevHash)
	if err == sql.ErrNoRows {
		prevHash = GenesisHash
	} else if err != nil {
		return nil, fmt.Errorf("obtener prev_hash: %w", err)
	}

	now := time.Now().UTC()
	recordHash := CalculateRecordHash(prevHash, now, agent, action, details)

	query := `
	INSERT INTO audit_trail_chained (timestamp, agent, action, details, prev_hash, record_hash)
	VALUES (?, ?, ?, ?, ?, ?);`

	res, err := l.db.ExecContext(ctx, query, now, agent, action, details, prevHash, recordHash)
	if err != nil {
		return nil, fmt.Errorf("insertar auditoría: %w", err)
	}

	id, _ := res.LastInsertId()
	return &AuditEntry{
		ID:         id,
		Timestamp:  now,
		Agent:      agent,
		Action:     action,
		Details:    details,
		PrevHash:   prevHash,
		RecordHash: recordHash,
	}, nil
}

// VerifyIntegrity recorre la cadena completa y certifica que ningún registro ha sido adulterado
func (l *AuditLogger) VerifyIntegrity(ctx context.Context) (bool, int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	rows, err := l.db.QueryContext(ctx, `SELECT id, timestamp, agent, action, details, prev_hash, record_hash FROM audit_trail_chained ORDER BY id ASC`)
	if err != nil {
		return false, 0, err
	}
	defer rows.Close()

	expectedPrevHash := GenesisHash
	count := 0

	for rows.Next() {
		var entry AuditEntry
		var tsStr string
		if err := rows.Scan(&entry.ID, &tsStr, &entry.Agent, &entry.Action, &entry.Details, &entry.PrevHash, &entry.RecordHash); err != nil {
			return false, count, fmt.Errorf("scan fila %d: %w", count+1, err)
		}

		if parsedTs, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
			entry.Timestamp = parsedTs
		} else if parsedTs, err := time.Parse("2006-01-02 15:04:05", tsStr); err == nil {
			entry.Timestamp = parsedTs
		} else if parsedTs, err := time.Parse(time.RFC3339, tsStr); err == nil {
			entry.Timestamp = parsedTs
		}

		// Validar encadenamiento con el hash previo
		if entry.PrevHash != expectedPrevHash {
			return false, count, fmt.Errorf("ruptura de cadena en registro #%d: prev_hash esperado %s, obtenido %s", entry.ID, expectedPrevHash, entry.PrevHash)
		}

		// Recomputar hash del registro
		computedHash := CalculateRecordHash(entry.PrevHash, entry.Timestamp, entry.Agent, entry.Action, entry.Details)
		if computedHash != entry.RecordHash {
			return false, count, fmt.Errorf("hash inválido en registro #%d: esperado %s, calculado %s", entry.ID, entry.RecordHash, computedHash)
		}

		expectedPrevHash = entry.RecordHash
		count++
	}

	return true, count, nil
}

// GetRecentLogs obtiene los registros más recientes de auditoría
func (l *AuditLogger) GetRecentLogs(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
	SELECT id, timestamp, agent, action, details, prev_hash, record_hash
	FROM audit_trail_chained
	ORDER BY id DESC
	LIMIT ?;`

	rows, err := l.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var tsStr string
		if err := rows.Scan(&e.ID, &tsStr, &e.Agent, &e.Action, &e.Details, &e.PrevHash, &e.RecordHash); err == nil {
			if parsedTs, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
				e.Timestamp = parsedTs
			} else if parsedTs, err := time.Parse("2006-01-02 15:04:05", tsStr); err == nil {
				e.Timestamp = parsedTs
			} else if parsedTs, err := time.Parse(time.RFC3339, tsStr); err == nil {
				e.Timestamp = parsedTs
			}
			logs = append(logs, e)
		}
	}
	return logs, nil
}
