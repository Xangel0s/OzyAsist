package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompressAndExtractZip(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Crear estructura de prueba
	srcFolder := filepath.Join(tempDir, "proyecto")
	subFolder := filepath.Join(srcFolder, "sub")
	if err := os.MkdirAll(subFolder, 0755); err != nil {
		t.Fatalf("error creando carpetas: %v", err)
	}

	file1 := filepath.Join(srcFolder, "hola.txt")
	file2 := filepath.Join(subFolder, "datos.json")
	if err := os.WriteFile(file1, []byte("contenido 1"), 0644); err != nil {
		t.Fatalf("error escribiendo file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(`{"clave":"valor"}`), 0644); err != nil {
		t.Fatalf("error escribiendo file2: %v", err)
	}

	// 2. Comprimir
	zipPath := filepath.Join(tempDir, "paquete.zip")
	err := CompressZip([]string{srcFolder}, zipPath)
	if err != nil {
		t.Fatalf("CompressZip falló: %v", err)
	}

	fi, err := os.Stat(zipPath)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("el archivo zip no fue creado o está vacío: %v", err)
	}

	// 3. Descomprimir
	extractDir := filepath.Join(tempDir, "extraido")
	files, err := ExtractZip(zipPath, extractDir)
	if err != nil {
		t.Fatalf("ExtractZip falló: %v", err)
	}

	if len(files) < 2 {
		t.Errorf("se esperaban al menos 2 archivos extraídos, se obtuvieron: %d", len(files))
	}

	// Verificar contenido
	extractedFile1 := filepath.Join(extractDir, "proyecto", "hola.txt")
	data, err := os.ReadFile(extractedFile1)
	if err != nil || string(data) != "contenido 1" {
		t.Errorf("contenido de archivo extraído incorrecto: %s, error: %v", string(data), err)
	}
}
