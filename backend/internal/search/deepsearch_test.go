package search

import (
	"context"
	"strings"
	"testing"

	"github.com/ozyassist/backend/internal/providers"
)

type mockSearchProvider struct {
	response string
}

func (m *mockSearchProvider) StreamCompletion(ctx context.Context, messages []providers.Message, opts providers.CompletionOptions) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk, 2)
	ch <- providers.StreamChunk{Type: "text", Content: m.response}
	ch <- providers.StreamChunk{Type: "done"}
	close(ch)
	return ch, nil
}

func (m *mockSearchProvider) Name() string        { return "mock-search" }
func (m *mockSearchProvider) SupportsTools() bool { return false }
func (m *mockSearchProvider) Models() []string    { return []string{"mock-model"} }

func TestDeepSearchEngine_Execute(t *testing.T) {
	mockLLM := &mockSearchProvider{
		response: "SQLite con WAL mode ofrece mayor concurrencia [1].",
	}

	mockSearch := func(ctx context.Context, query string) ([]SourceCitation, error) {
		return []SourceCitation{
			{
				URL:     "https://sqlite.org/wal.html",
				Title:   "Write-Ahead Logging",
				Snippet: "WAL allows concurrent readers and writers.",
			},
		}, nil
	}

	engine := NewDeepSearchEngine(mockLLM, mockSearch)
	res, err := engine.Execute(context.Background(), "Mejores prácticas SQLite")
	if err != nil {
		t.Fatalf("Error en DeepSearch: %v", err)
	}

	if len(res.Sources) != 1 {
		t.Fatalf("Esperaba 1 fuente, obtuvo %d", len(res.Sources))
	}

	if res.Sources[0].Domain != "sqlite.org" {
		t.Errorf("Dominio incorrecto: %s", res.Sources[0].Domain)
	}

	if res.Answer != "SQLite con WAL mode ofrece mayor concurrencia [1]." {
		t.Errorf("Respuesta inesperada: %s", res.Answer)
	}
}

func TestExtractDomain(t *testing.T) {
	cases := []struct {
		url      string
		expected string
	}{
		{"https://www.google.com/search?q=test", "google.com"},
		{"https://github.com/ozyassist/backend", "github.com"},
		{"http://localhost:8080/api", "localhost"},
	}

	for _, c := range cases {
		got := ExtractDomain(c.url)
		if got != c.expected {
			t.Errorf("ExtractDomain(%s) = %s, esperado %s", c.url, got, c.expected)
		}
	}
}

func TestDefaultDuckDuckGoSearch_Live(t *testing.T) {
	results, err := DefaultDuckDuckGoSearch(context.Background(), "golang official site")
	if err != nil {
		t.Logf("DuckDuckGo request error (network/timeout): %v", err)
		return
	}
	if len(results) == 0 {
		t.Errorf("Esperaba resultados para 'golang official site', obtuvo 0")
	}
	for i, r := range results {
		t.Logf("[%d] %s -> %s", i+1, r.Title, r.URL)
	}
}

func TestCleanHTMLToText_Advanced(t *testing.T) {
	rawHTML := `<html>
		<head><title>Test Page</title><style>.cls{color:red;}</style></head>
		<body>
			<nav><a href="/">Home</a></nav>
			<h1>Título Principal</h1>
			<p>Este es un <b>párrafo</b> con información técnica importante sobre inferencia de modelos.</p>
			<ul>
				<li>Primer punto clave</li>
				<li>Segundo punto clave</li>
			</ul>
			<footer>Copyright 2026</footer>
		</body>
	</html>`

	cleaned := cleanHTMLToText(rawHTML, 1000)
	if strings.Contains(cleaned, "<nav>") || strings.Contains(cleaned, "<style>") {
		t.Errorf("cleanHTMLToText no eliminó las etiquetas de navegación o estilo")
	}
	if !strings.Contains(cleaned, "### Título Principal") {
		t.Errorf("cleanHTMLToText no formateó el encabezado: %s", cleaned)
	}
	if !strings.Contains(cleaned, "• Primer punto clave") {
		t.Errorf("cleanHTMLToText no formateó las listas: %s", cleaned)
	}
}

func TestSaveResearchReport(t *testing.T) {
	res := &DeepSearchResult{
		Query:  "Inferencia rápida LLM",
		Answer: "Groq y Baseten ofrecen inferencia optimizada [1].",
		Sources: []SourceCitation{
			{ID: 1, Title: "Groq LPU", URL: "https://groq.com", Domain: "groq.com", Snippet: "Inferencia ultra-rápida a 800 t/s."},
		},
	}

	tempDir := t.TempDir()
	filePath, err := SaveResearchReport(res, tempDir)
	if err != nil {
		t.Fatalf("SaveResearchReport falló: %v", err)
	}

	if filePath == "" {
		t.Fatalf("Esperaba una ruta de archivo no vacía")
	}
}

