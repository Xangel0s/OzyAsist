package agent

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestQueryDB_SafeSelectAndBlocking(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Crear DB de prueba con SQLite
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("error abriendo db: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);
		INSERT INTO users (name, email) VALUES ('Alice', 'alice@test.com'), ('Bob', 'bob@test.com');
	`)
	if err != nil {
		t.Fatalf("error creando datos de prueba: %v", err)
	}
	db.Close()

	ctx := context.Background()

	// 1. Probar SELECT válido
	out, err := ExecuteSafeQuery(ctx, dbPath, "SELECT id, name, email FROM users ORDER BY id", 10)
	if err != nil {
		t.Fatalf("ExecuteSafeQuery falló en SELECT válido: %v", err)
	}
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "bob@test.com") {
		t.Errorf("El resultado debería contener 'Alice' y 'bob@test.com', obtuve:\n%s", out)
	}
	if !strings.Contains(out, "| id | name | email |") {
		t.Errorf("El resultado debería tener formato de tabla Markdown, obtuve:\n%s", out)
	}

	// 2. Probar PRAGMA
	outPragma, err := ExecuteSafeQuery(ctx, dbPath, "PRAGMA table_info(users)", 10)
	if err != nil {
		t.Fatalf("ExecuteSafeQuery falló en PRAGMA: %v", err)
	}
	if !strings.Contains(outPragma, "email") {
		t.Errorf("PRAGMA table_info debería listar la columna 'email', obtuve:\n%s", outPragma)
	}

	// 3. Probar Bloqueo de DROP TABLE
	_, err = ExecuteSafeQuery(ctx, dbPath, "DROP TABLE users", 10)
	if err == nil {
		t.Errorf("ExecuteSafeQuery debería bloquear DROP TABLE")
	}

	// 4. Probar Bloqueo de DELETE
	_, err = ExecuteSafeQuery(ctx, dbPath, "DELETE FROM users WHERE id = 1", 10)
	if err == nil {
		t.Errorf("ExecuteSafeQuery debería bloquear DELETE")
	}

	// 5. Probar Bloqueo de inyección con punto y coma
	_, err = ExecuteSafeQuery(ctx, dbPath, "SELECT * FROM users; DROP TABLE users;", 10)
	if err == nil {
		t.Errorf("ExecuteSafeQuery debería bloquear consultas con sentencias encadenadas")
	}
}
