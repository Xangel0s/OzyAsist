package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

type QueryDBParams struct {
	DBPath  string `json:"db_path"`            // Ruta a la base de datos (ej: 'backend/data/ozyassist.db', 'crmgeofal/app.db')
	Query   string `json:"query"`              // Consulta SQL (SELECT / PRAGMA / EXPLAIN)
	MaxRows int    `json:"max_rows,omitempty"` // Límite de filas a devolver (default: 50)
}

var destructiveSQLRegex = regexp.MustCompile(`(?i)\b(DROP|DELETE|UPDATE|INSERT|ALTER|TRUNCATE|VACUUM|ATTACH|DETACH|CREATE|REPLACE|GRANT|REVOKE)\b`)

// ValidateSafeSQL verifica que la consulta sea estrictamente de solo lectura y no destructiva.
func ValidateSafeSQL(query string) error {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return fmt.Errorf("la consulta SQL no puede estar vacía")
	}

	// Verificar si hay múltiples sentencias encadenadas con punto y coma
	statements := strings.Split(trimmed, ";")
	nonEmptyCount := 0
	for _, st := range statements {
		if strings.TrimSpace(st) != "" {
			nonEmptyCount++
		}
	}
	if nonEmptyCount > 1 {
		return fmt.Errorf("no se permite la ejecución de múltiples sentencias SQL encadenadas")
	}

	// Comprobar que inicie con una operación de consulta permitida
	upper := strings.ToUpper(trimmed)
	if !strings.HasPrefix(upper, "SELECT") &&
		!strings.HasPrefix(upper, "PRAGMA") &&
		!strings.HasPrefix(upper, "EXPLAIN") &&
		!strings.HasPrefix(upper, "WITH") {
		return fmt.Errorf("operación no permitida: solo se autorizan consultas de solo lectura (SELECT, PRAGMA, EXPLAIN, WITH)")
	}

	// Comprobar que no contenga palabras clave destructivas
	if destructiveSQLRegex.MatchString(trimmed) {
		return fmt.Errorf("la consulta contiene operaciones destructivas o de modificación bloqueadas")
	}

	return nil
}

// ExecuteSafeQuery ejecuta una consulta SQL de solo lectura sobre una base de datos SQLite local.
func ExecuteSafeQuery(ctx context.Context, dbPath, query string, maxRows int) (string, error) {
	if err := ValidateSafeSQL(query); err != nil {
		return "", err
	}

	resolvedPath := system.ResolveUserPath(dbPath)
	if _, err := os.Stat(resolvedPath); err != nil {
		return "", fmt.Errorf("no se encontró el archivo de base de datos en: %s", resolvedPath)
	}

	if maxRows <= 0 {
		maxRows = 50
	}

	dsn := fmt.Sprintf("file:%s?mode=ro", filepath.ToSlash(resolvedPath))
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return "", fmt.Errorf("error abriendo conexión SQLite en modo lectura: %v", err)
	}
	defer db.Close()

	start := time.Now()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("error ejecutando consulta SQL: %v", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("error obteniendo columnas: %v", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🗄️ === CONSULTA SQL EJECUTADA: %s ===\n", filepath.Base(resolvedPath)))
	sb.WriteString(fmt.Sprintf("SQL: `%s`\n\n", query))

	// Encabezado Markdown
	sb.WriteString("| " + strings.Join(cols, " | ") + " |\n")
	var sep []string
	for range cols {
		sep = append(sep, "---")
	}
	sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")

	rowCount := 0
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if rowCount >= maxRows {
			sb.WriteString(fmt.Sprintf("\n... [Límite de %d filas alcanzado. Consulta truncada para visualización]", maxRows))
			break
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return "", fmt.Errorf("error leyendo fila: %v", err)
		}

		var rowStr []string
		for _, val := range values {
			if val == nil {
				rowStr = append(rowStr, "NULL")
			} else {
				switch v := val.(type) {
				case []byte:
					rowStr = append(rowStr, string(v))
				default:
					rowStr = append(rowStr, fmt.Sprintf("%v", v))
				}
			}
		}
		sb.WriteString("| " + strings.Join(rowStr, " | ") + " |\n")
		rowCount++
	}

	duration := time.Since(start).Milliseconds()
	sb.WriteString(fmt.Sprintf("\n• Filas obtenidas: %d | Tiempo: %d ms", rowCount, duration))

	return sb.String(), nil
}

func execOSQueryDB(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params QueryDBParams
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_query_db: %v", err), false
	}

	out, err := ExecuteSafeQuery(ctx, params.DBPath, params.Query, params.MaxRows)
	if err != nil {
		return fmt.Sprintf("Error en os_query_db: %v", err), false
	}

	return out, true
}
