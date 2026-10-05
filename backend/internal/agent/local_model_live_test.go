package agent

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

// TestLocalModel_ComprehensiveLive ejecuta las pruebas solicitadas contra el modelo local de Ollama.
func TestLocalModel_ComprehensiveLive(t *testing.T) {
	// Verificar si Ollama está corriendo en localhost:11434
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://127.0.0.1:11434/api/tags")
	if err != nil {
		t.Skip("Ollama no está activo en 127.0.0.1:11434; omitiendo prueba live.")
		return
	}
	resp.Body.Close()

	setupTestDB(t)

	testDir, err := os.MkdirTemp("", "ozy-live-local-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	userID := db.DefaultUserID()
	if err := db.EnsureDefaultUser(); err != nil {
		t.Fatalf("ensure default user: %v", err)
	}

	projectID := uuid.NewString()
	proj := &models.Project{
		ID:              projectID,
		UserID:          userID,
		Name:            "Local Live Project",
		RootPath:        testDir,
		PermissionLevel: "sandboxed",
		AgentConsent:    "always",
		CreatedAt:       time.Now(),
	}
	if err := db.CreateProject(proj); err != nil {
		t.Fatalf("create project: %v", err)
	}

	provider := providers.NewOllama("http://127.0.0.1:11434/v1")
	provider.SetModel("llama3.1:latest")

	// 1. Prueba de Documentación Word (.docx)
	t.Run("CreateDocxReport", func(t *testing.T) {
		docxPath := filepath.Join(testDir, "test_doc.docx")
		tc := providers.ToolCall{
			ID:   "call_docx",
			Name: "os_create_docx",
			Input: mustJSON(`{
				"path": "` + strings.ReplaceAll(docxPath, `\`, `\\`) + `",
				"title": "Arquitectura Híbrida 3.0",
				"sections": [
					{"title": "Rust Core", "content": "Captura nativa de cámara con nokhwa y MSMF."},
					{"title": "MCTS Planning", "content": "Simulación de árboles de decisión en Python."}
				]
			}`),
		}

		out, ok := executeToolCall(context.Background(), tc, nil, nil)
		if !ok || !strings.Contains(out, "EXITOSAMENTE") {
			t.Fatalf("os_create_docx falló: %s", out)
		}

		fi, err := os.Stat(docxPath)
		if err != nil || fi.Size() < 500 {
			t.Fatalf("archivo docx no generado o corrupto: %v", err)
		}
	})

	// 2. Prueba de Informe PDF (.pdf)
	t.Run("CreatePDFReport", func(t *testing.T) {
		pdfPath := filepath.Join(testDir, "test_report.pdf")
		tc := providers.ToolCall{
			ID:   "call_pdf",
			Name: "os_create_pdf",
			Input: mustJSON(`{
				"path": "` + strings.ReplaceAll(pdfPath, `\`, `\\`) + `",
				"title": "Reporte Zero-Docker",
				"sections": [
					{"title": "Ontología GraphRAG", "content": "Grafo ontológico en SQLite y RAM."}
				]
			}`),
		}

		out, ok := executeToolCall(context.Background(), tc, nil, nil)
		if !ok || !strings.Contains(out, "EXITOSAMENTE") {
			t.Fatalf("os_create_pdf falló: %s", out)
		}

		fi, err := os.Stat(pdfPath)
		if err != nil || fi.Size() < 500 {
			t.Fatalf("archivo pdf no generado o corrupto: %v", err)
		}
	})

	// 3. Prueba de Hoja de Cálculo Excel (.xlsx) con tipos mixtos (números y strings)
	t.Run("CreateExcelSpreadsheet", func(t *testing.T) {
		xlsxPath := filepath.Join(testDir, "test_metrics.xlsx")
		tc := providers.ToolCall{
			ID:   "call_excel",
			Name: "os_create_excel",
			Input: mustJSON(`{
				"path": "` + strings.ReplaceAll(xlsxPath, `\`, `\\`) + `",
				"title": "Métricas",
				"headers": ["Módulo", "LatenciaMs", "Estado"],
				"rows": [
					["GraphRAG", 0.5, "Activo"],
					["MCTS Planner", 1.2, "Activo"],
					["Rust Camera", 15, "Activo"]
				]
			}`),
		}

		out, ok := executeToolCall(context.Background(), tc, nil, nil)
		if !ok || !strings.Contains(out, "EXITOSAMENTE") {
			t.Fatalf("os_create_excel falló: %s", out)
		}

		fi, err := os.Stat(xlsxPath)
		if err != nil || fi.Size() < 500 {
			t.Fatalf("archivo xlsx no generado o corrupto: %v", err)
		}
	})

	// 4. Prueba de Consulta Ontológica GraphRAG
	t.Run("QueryOntologyGraphRAG", func(t *testing.T) {
		tc := providers.ToolCall{
			ID:   "call_ontology",
			Name: "query_system_knowledge",
			Input: mustJSON(`{
				"query": "captura de camara rust nokhwa y audio wasapi",
				"topK": 2
			}`),
		}

		out, ok := executeToolCall(context.Background(), tc, nil, nil)
		if !ok || !strings.Contains(out, "CONOCIMIENTO ONTOLÓGICO") {
			t.Fatalf("query_system_knowledge falló: %s", out)
		}
	})

	// 5. Prueba de MCTS Error Recovery
	t.Run("MCTSErrorRecovery", func(t *testing.T) {
		tc := providers.ToolCall{
			ID:   "call_mcts",
			Name: "nine_mcts_solve",
			Input: mustJSON(`{
				"goal": "detener proceso bloqueado",
				"failedTool": "os_close_window",
				"errorMessage": "Acceso denegado error 740",
				"iterations": 50
			}`),
		}

		out, ok := executeToolCall(context.Background(), tc, nil, nil)
		if !ok || !strings.Contains(out, "NINE MCTS PLANNER") {
			t.Fatalf("nine_mcts_solve falló: %s", out)
		}
		if !strings.Contains(out, "RECUPERACIÓN") && !strings.Contains(out, "TRAYECTORIA") {
			t.Fatalf("nine_mcts_solve no devolvió recuperación esperada: %s", out)
		}
	})
}
