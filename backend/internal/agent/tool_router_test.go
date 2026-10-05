package agent

import (
	"testing"
	"time"
)

func TestRAMToolRouter_OfficeDomain(t *testing.T) {
	router := DefaultRAMToolRouter()
	tools := router.RouteTools("por favor crea un reporte en word docx sobre las ventas", false, true)

	if len(tools) == 0 {
		t.Fatalf("expected tools for office query, got 0")
	}

	foundDocx := false
	for _, tool := range tools {
		if tool.Name == "os_create_docx" {
			foundDocx = true
			break
		}
	}

	if !foundDocx {
		t.Errorf("expected os_create_docx in routed tools, but not found in %v", tools)
	}
}

func TestRAMToolRouter_FilesDomain(t *testing.T) {
	router := DefaultRAMToolRouter()
	tools := router.RouteTools("busca los archivos en la carpeta de descargas", false, true)

	if len(tools) == 0 {
		t.Fatalf("expected tools for files query, got 0")
	}

	foundFileTool := false
	for _, tool := range tools {
		if tool.Name == "os_find_files" || tool.Name == "os_explore" {
			foundFileTool = true
			break
		}
	}

	if !foundFileTool {
		t.Errorf("expected os_find_files or os_explore in routed tools")
	}
}

func TestRAMToolRouter_LatencyUnder1ms(t *testing.T) {
	router := DefaultRAMToolRouter()
	query := "investiga en la web y redacta un correo para el cliente"

	start := time.Now()
	for i := 0; i < 100; i++ {
		_ = router.RouteTools(query, false, true)
	}
	elapsed := time.Since(start)
	avgMicroseconds := elapsed.Microseconds() / 100

	t.Logf("100 RAM router executions took %v (avg: %d µs per query)", elapsed, avgMicroseconds)
	if avgMicroseconds > 1000 {
		t.Errorf("expected RAM router avg latency under 1000µs (1ms), got %d µs", avgMicroseconds)
	}
}
