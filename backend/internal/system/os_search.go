package system

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ContentMatch representa una coincidencia de búsqueda dentro de un archivo
type ContentMatch struct {
	Path        string `json:"path"`
	LineNumber  int    `json:"line_number"`
	LineContent string `json:"line_content"`
}

// SearchContentParams configura la búsqueda de contenido
type SearchContentParams struct {
	RootDir     string   `json:"root_dir"`
	Query       string   `json:"query"`
	IsRegex     bool     `json:"is_regex"`
	Extensions  []string `json:"extensions,omitempty"`
	MaxResults  int      `json:"max_results"`
}

// SearchContent busca palabras o patrones dentro de los archivos de texto de una carpeta
func SearchContent(params SearchContentParams) ([]ContentMatch, error) {
	if params.Query == "" {
		return nil, fmt.Errorf("el término de búsqueda no puede estar vacío")
	}

	rootDir := ResolveUserPath(params.RootDir)
	if rootDir == "" || rootDir == "." {
		rootDir, _ = os.Getwd()
	}

	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}

	var re *regexp.Regexp
	queryLower := strings.ToLower(params.Query)
	if params.IsRegex {
		var err error
		re, err = regexp.Compile("(?i)" + params.Query)
		if err != nil {
			return nil, fmt.Errorf("expresión regular inválida: %w", err)
		}
	}

	extFilter := make(map[string]bool)
	for _, ext := range params.Extensions {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extFilter[ext] = true
	}

	var matches []ContentMatch

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if len(matches) >= maxResults {
			return filepath.SkipAll
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == "dist" || name == "build" || name == ".next" ||
				name == "target" || name == "AppData" || name == "venv" ||
				name == "__pycache__" || name == ".idea" || name == ".vscode" ||
				name == ".tauri" {
				return filepath.SkipDir
			}
			return nil
		}

		// Filtro por extensión si se definió
		fileExt := strings.ToLower(filepath.Ext(path))
		if len(extFilter) > 0 && !extFilter[fileExt] {
			return nil
		}

		// Descartar extensiones claramente binarias
		binaryExts := map[string]bool{
			".exe": true, ".dll": true, ".so": true, ".bin": true, ".zip": true,
			".tar": true, ".gz": true, ".7z": true, ".png": true, ".jpg": true,
			".jpeg": true, ".gif": true, ".mp3": true, ".mp4": true, ".pdf": true,
			".docx": true, ".xlsx": true, ".pptx": true, ".db": true, ".sqlite": true,
		}
		if binaryExts[fileExt] {
			return nil
		}

		// Verificar que el archivo no sea binario (primeros 512 bytes)
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		if n > 0 && bytes.IndexByte(buf[:n], 0) != -1 {
			// Es binario, saltar
			return nil
		}

		// Rebobinar archivo
		_, _ = file.Seek(0, io.SeekStart)

		scanner := bufio.NewScanner(file)
		// Permitir líneas de hasta 1MB
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		lineNum := 1

		for scanner.Scan() {
			if len(matches) >= maxResults {
				return filepath.SkipAll
			}

			lineText := scanner.Text()
			matched := false

			if params.IsRegex {
				matched = re.MatchString(lineText)
			} else {
				matched = strings.Contains(strings.ToLower(lineText), queryLower)
			}

			if matched {
				trimmedLine := strings.TrimSpace(lineText)
				if len(trimmedLine) > 250 {
					trimmedLine = trimmedLine[:250] + "..."
				}

				matches = append(matches, ContentMatch{
					Path:        path,
					LineNumber:  lineNum,
					LineContent: trimmedLine,
				})
			}
			lineNum++
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, fmt.Errorf("error durante la búsqueda de contenido: %w", err)
	}

	return matches, nil
}
