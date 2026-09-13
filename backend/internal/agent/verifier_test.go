package agent

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifier_ExitCode(t *testing.T) {
	v := NewVerifier()

	rule := `{"type":"exit_code","expected_code":0}`
	valid, _, err := v.Verify(rule, "ok", 0)
	if err != nil || !valid {
		t.Fatalf("Esperaba valid=true, obtuvo valid=%v, err=%v", valid, err)
	}

	valid, _, _ = v.Verify(rule, "error", 1)
	if valid {
		t.Fatal("Esperaba valid=false para exit_code distinto")
	}
}

func TestVerifier_FileExists(t *testing.T) {
	v := NewVerifier()
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "test.txt")

	_ = os.WriteFile(targetFile, []byte("contenido de prueba"), 0644)

	rule := `{"type":"file_exists","path":"` + filepath.ToSlash(targetFile) + `","min_size_bytes":5}`
	valid, _, err := v.Verify(rule, "", 0)
	if err != nil || !valid {
		t.Fatalf("Esperaba archivo existente válido, obtuvo: %v, err=%v", valid, err)
	}

	// Archivo inexistente
	ruleMissing := `{"type":"file_exists","path":"` + filepath.ToSlash(filepath.Join(tmpDir, "missing.txt")) + `"}`
	valid, _, _ = v.Verify(ruleMissing, "", 0)
	if valid {
		t.Fatal("Esperaba valid=false para archivo inexistente")
	}
}

func TestVerifier_FileContains(t *testing.T) {
	v := NewVerifier()
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "report.log")

	_ = os.WriteFile(targetFile, []byte("Build successful. Artifact generated at /bin/app"), 0644)

	rule := `{"type":"file_contains","path":"` + filepath.ToSlash(targetFile) + `","pattern":"Build successful"}`
	valid, _, err := v.Verify(rule, "", 0)
	if err != nil || !valid {
		t.Fatalf("Esperaba coincidencia en archivo, err=%v", err)
	}

	ruleMismatch := `{"type":"file_contains","path":"` + filepath.ToSlash(targetFile) + `","pattern":"Fatal error"}`
	valid, _, _ = v.Verify(ruleMismatch, "", 0)
	if valid {
		t.Fatal("Esperaba valid=false para patrón no encontrado")
	}
}

func TestVerifier_OutputRegex(t *testing.T) {
	v := NewVerifier()
	rule := `{"type":"output_regex","pattern":"PASS: \\d+ tests"}`

	valid, _, err := v.Verify(rule, "Running suite... PASS: 42 tests finished", 0)
	if err != nil || !valid {
		t.Fatalf("Esperaba coincidencia regex, err=%v", err)
	}

	valid, _, _ = v.Verify(rule, "FAIL: 2 tests failed", 0)
	if valid {
		t.Fatal("Esperaba fallo por no coincidir con expresión regular")
	}
}

func TestVerifier_HTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	v := NewVerifier()
	rule := `{"type":"http_status","url":"` + server.URL + `","expected_code":200}`

	valid, _, err := v.Verify(rule, "", 0)
	if err != nil || !valid {
		t.Fatalf("Esperaba HTTP status 200 válido, err=%v", err)
	}

	ruleBadStatus := `{"type":"http_status","url":"` + server.URL + `","expected_code":404}`
	valid, _, _ = v.Verify(ruleBadStatus, "", 0)
	if valid {
		t.Fatal("Esperaba valid=false para status code no coincidente")
	}
}
