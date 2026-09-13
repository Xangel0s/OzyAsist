package security

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestAuditDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Error abriendo SQLite: %v", err)
	}

	migration := `
	CREATE TABLE audit_trail_chained (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		agent TEXT NOT NULL,
		action TEXT NOT NULL,
		details TEXT NOT NULL,
		prev_hash TEXT NOT NULL,
		record_hash TEXT NOT NULL
	);`

	if _, err := db.Exec(migration); err != nil {
		t.Fatalf("Error migración prueba: %v", err)
	}
	return db
}

func TestAuditTrail_HashChainingAndIntegrity(t *testing.T) {
	db := setupTestAuditDB(t)
	defer db.Close()

	logger := NewAuditLogger(db)
	ctx := context.Background()

	// 1. Registrar primer bloque
	entry1, err := logger.Record(ctx, "ozy", "shell_exec", "go test ./...")
	if err != nil {
		t.Fatalf("Fallo en registro 1: %v", err)
	}
	if entry1.PrevHash != GenesisHash {
		t.Errorf("Esperaba genesis hash en bloque 1, obtuve %s", entry1.PrevHash)
	}

	// 2. Registrar segundo bloque encadenado
	entry2, err := logger.Record(ctx, "charc", "safety_audit", "comando aprobado")
	if err != nil {
		t.Fatalf("Fallo en registro 2: %v", err)
	}
	if entry2.PrevHash != entry1.RecordHash {
		t.Errorf("El registro 2 no apunta al hash del registro 1")
	}

	// 3. Verificar integridad de la cadena
	valid, count, err := logger.VerifyIntegrity(ctx)
	if err != nil || !valid || count != 2 {
		t.Fatalf("La verificación de integridad falló: valid=%v, count=%d, err=%v", valid, count, err)
	}

	// 4. Test de adulteración: modificar un registro arbitrariamente en la base de datos
	_, err = db.Exec(`UPDATE audit_trail_chained SET details = 'comando malicioso inyectado' WHERE id = 1`)
	if err != nil {
		t.Fatalf("Error alterando BD: %v", err)
	}

	// La verificación debe detectar la alteración inmediatamente
	validTampered, _, _ := logger.VerifyIntegrity(ctx)
	if validTampered {
		t.Fatalf("La verificación no detectó la adulteración del registro de auditoría")
	}
}
