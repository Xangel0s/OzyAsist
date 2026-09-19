package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="test_doc.pdf"`)
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-1.3 fake pdf content for testing downloader"))
	}))
	defer ts.Close()

	tempDir := t.TempDir()
	destFile := filepath.Join(tempDir, "custom_doc.pdf")

	res, err := DownloadFile(context.Background(), ts.URL+"/sample", destFile)
	if err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}

	if res.SizeBytes == 0 {
		t.Errorf("expected downloaded size > 0, got %d", res.SizeBytes)
	}

	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("no se pudo leer archivo descargado: %v", err)
	}
	if string(data) != "%PDF-1.3 fake pdf content for testing downloader" {
		t.Errorf("contenido descargado incorrecto: %s", string(data))
	}
}
