package system

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestUndoDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Error abriendo SQLite: %v", err)
	}

	migration := `
	CREATE TABLE shadow_snapshots (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL,
		file_path TEXT NOT NULL,
		content_before BLOB NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(migration); err != nil {
		t.Fatalf("Error migración prueba: %v", err)
	}
	return db
}

func TestUndoEngine_CaptureAndRollback(t *testing.T) {
	db := setupTestUndoDB(t)
	defer db.Close()

	engine := NewUndoEngine(db)
	ctx := context.Background()

	// Crear archivo temporal
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config.json")
	initialContent := []byte(`{"version": 1, "active": true}`)
	if err := os.WriteFile(testFile, initialContent, 0644); err != nil {
		t.Fatalf("Error creando archivo test: %v", err)
	}

	// 1. Capturar snapshot antes de que el agente lo modifique
	snap, err := engine.CaptureSnapshot(ctx, "task-123", testFile)
	if err != nil {
		t.Fatalf("Error capturando snapshot: %v", err)
	}
	if string(snap.ContentBefore) != string(initialContent) {
		t.Errorf("Contenido snapshot incorrecto: %s", string(snap.ContentBefore))
	}

	// 2. Simular modificación por el agente
	modifiedContent := []byte(`{"version": 2, "active": false, "corrupted": true}`)
	if err := os.WriteFile(testFile, modifiedContent, 0644); err != nil {
		t.Fatalf("Error modificando archivo: %v", err)
	}

	// 3. Ejecutar RollbackTime-Travel
	restored, err := engine.RollbackTask(ctx, "task-123")
	if err != nil {
		t.Fatalf("Error en rollback: %v", err)
	}
	if len(restored) != 1 || restored[0] != testFile {
		t.Errorf("Lista de archivos restaurados inesperada: %v", restored)
	}

	// 4. Validar que el archivo volvió al estado exacto inicial
	restoredContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Error leyendo archivo restaurado: %v", err)
	}
	if string(restoredContent) != string(initialContent) {
		t.Fatalf("El archivo no se restauró correctamente. Esperado: %s, Obtenido: %s", string(initialContent), string(restoredContent))
	}
}
