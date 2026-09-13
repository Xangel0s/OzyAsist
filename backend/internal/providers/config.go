package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var configMu sync.Mutex

// LoadEnvFiles intenta cargar variables de entorno desde .env si existe
func LoadEnvFiles() {
	configMu.Lock()
	defer configMu.Unlock()

	candidates := []string{
		".env",
		filepath.Join("backend", ".env"),
		filepath.Join("..", ".env"),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			parseEnvContent(string(data))
			break
		}
	}
}

func parseEnvContent(content string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.Trim(strings.TrimSpace(parts[1]), "\"'`")
			if os.Getenv(k) == "" && v != "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}

// SaveConfigKey guarda o actualiza una variable en el archivo .env local
func SaveConfigKey(key, value string) error {
	configMu.Lock()
	defer configMu.Unlock()

	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)

	_ = os.Setenv(key, value)

	targetPath := ".env"
	if _, err := os.Stat("backend"); err == nil {
		targetPath = filepath.Join("backend", ".env")
	}

	existing, _ := os.ReadFile(targetPath)
	lines := strings.Split(string(existing), "\n")
	found := false
	var newLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			newLines = append(newLines, fmt.Sprintf("%s=%s", key, value))
			found = true
		} else if trimmed != "" || len(newLines) > 0 {
			newLines = append(newLines, line)
		}
	}

	if !found {
		newLines = append(newLines, fmt.Sprintf("%s=%s", key, value))
	}

	return os.WriteFile(targetPath, []byte(strings.Join(newLines, "\n")+"\n"), 0644)
}
