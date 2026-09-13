package system

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type ShadowSnapshot struct {
	ID            string    `json:"id"`
	TaskID        string    `json:"task_id"`
	FilePath      string    `json:"file_path"`
	ContentBefore []byte    `json:"content_before"`
	CreatedAt     time.Time `json:"created_at"`
}

type UndoEngine struct {
	db *sql.DB
}

func NewUndoEngine(db *sql.DB) *UndoEngine {
	return &UndoEngine{db: db}
}

// CaptureSnapshot guarda una copia exacta del archivo antes de ser alterado por el agente
func (u *UndoEngine) CaptureSnapshot(ctx context.Context, taskID, filePath string) (*ShadowSnapshot, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("obtener ruta absoluta: %w", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// El archivo es nuevo, el contenido anterior es vacío
			content = []byte{}
		} else {
			return nil, fmt.Errorf("leer archivo para snapshot: %w", err)
		}
	}

	snapID := uuid.NewString()
	now := time.Now().UTC()

	query := `
	INSERT INTO shadow_snapshots (id, task_id, file_path, content_before, created_at)
	VALUES (?, ?, ?, ?, ?);`

	_, err = u.db.ExecContext(ctx, query, snapID, taskID, absPath, content, now)
	if err != nil {
		return nil, fmt.Errorf("guardar shadow snapshot: %w", err)
	}

	return &ShadowSnapshot{
		ID:            snapID,
		TaskID:        taskID,
		FilePath:      absPath,
		ContentBefore: content,
		CreatedAt:     now,
	}, nil
}

// RollbackTask revierte todos los archivos modificados durante una tarea a su estado original
func (u *UndoEngine) RollbackTask(ctx context.Context, taskID string) ([]string, error) {
	query := `
	SELECT id, task_id, file_path, content_before, created_at
	FROM shadow_snapshots
	WHERE task_id = ?
	ORDER BY created_at DESC;`

	rows, err := u.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("consultar snapshots de la tarea %s: %w", taskID, err)
	}
	defer rows.Close()

	var restoredFiles []string
	seen := make(map[string]bool)

	for rows.Next() {
		var s ShadowSnapshot
		var tsStr string
		if err := rows.Scan(&s.ID, &s.TaskID, &s.FilePath, &s.ContentBefore, &tsStr); err != nil {
			return restoredFiles, fmt.Errorf("scan snapshot: %w", err)
		}

		// Revertir a la versión más antigua si se capturaron múltiples del mismo archivo
		if seen[s.FilePath] {
			continue
		}
		seen[s.FilePath] = true

		if len(s.ContentBefore) == 0 {
			// El archivo no existía antes de la tarea, se elimina
			_ = os.Remove(s.FilePath)
		} else {
			// Restaurar contenido previo
			dir := filepath.Dir(s.FilePath)
			_ = os.MkdirAll(dir, 0755)
			if err := os.WriteFile(s.FilePath, s.ContentBefore, 0644); err != nil {
				return restoredFiles, fmt.Errorf("restaurar archivo %s: %w", s.FilePath, err)
			}
		}
		restoredFiles = append(restoredFiles, s.FilePath)
	}

	return restoredFiles, nil
}

// ListSnapshots lista los snapshots asociados a una tarea
func (u *UndoEngine) ListSnapshots(ctx context.Context, taskID string) ([]ShadowSnapshot, error) {
	query := `
	SELECT id, task_id, file_path, content_before, created_at
	FROM shadow_snapshots
	WHERE task_id = ?
	ORDER BY created_at ASC;`

	rows, err := u.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []ShadowSnapshot
	for rows.Next() {
		var s ShadowSnapshot
		var tsStr string
		if err := rows.Scan(&s.ID, &s.TaskID, &s.FilePath, &s.ContentBefore, &tsStr); err == nil {
			if parsed, err := time.Parse(time.RFC3339, tsStr); err == nil {
				s.CreatedAt = parsed
			}
			snapshots = append(snapshots, s)
		}
	}
	return snapshots, nil
}
