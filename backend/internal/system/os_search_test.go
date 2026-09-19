package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchContent(t *testing.T) {
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "config.env")
	file2 := filepath.Join(tempDir, "main.go")
	binFile := filepath.Join(tempDir, "image.png")

	_ = os.WriteFile(file1, []byte("API_SECRET_KEY=ozy_99182312\nDEBUG=true\n"), 0644)
	_ = os.WriteFile(file2, []byte("package main\n\nfunc Run() {\n\t// token secreto aquí\n}\n"), 0644)
	_ = os.WriteFile(binFile, []byte{0x89, 'P', 'N', 'G', 0x00, 0x01, 0x02}, 0644)

	// Test literal search
	matches, err := SearchContent(SearchContentParams{
		RootDir:    tempDir,
		Query:      "API_SECRET_KEY",
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("SearchContent failed: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("se esperaba 1 match, se encontraron %d", len(matches))
	}
	if matches[0].LineNumber != 1 {
		t.Errorf("línea esperada 1, obtenida: %d", matches[0].LineNumber)
	}

	// Test regex search
	regexMatches, err := SearchContent(SearchContentParams{
		RootDir:    tempDir,
		Query:      "ozy_[0-9]+",
		IsRegex:    true,
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("SearchContent regex failed: %v", err)
	}
	if len(regexMatches) != 1 {
		t.Errorf("se esperaba 1 match regex, obtenidos: %d", len(regexMatches))
	}
}
