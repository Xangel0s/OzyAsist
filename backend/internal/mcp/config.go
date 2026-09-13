package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// MCPServersConfig representa el formato estándar universal mcp_servers.json (Claude Desktop, Cursor, Antigravity).
type MCPServersConfig struct {
	MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

// MCPServerEntry define la configuración de un servidor individual dentro de mcpServers.
type MCPServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// FindDefaultConfigFile busca el archivo de configuración mcp_servers.json en las rutas canónicas:
// 1. Directorio actual o backend/mcp_servers.json
// 2. Directorio personal del usuario (~/.ozy/mcp_servers.json)
func FindDefaultConfigFile() string {
	candidates := []string{
		"mcp_servers.json",
		filepath.Join("backend", "mcp_servers.json"),
		filepath.Join("..", "mcp_servers.json"),
	}

	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		userProfile = os.Getenv("HOME")
	}
	if userProfile != "" {
		candidates = append(candidates, filepath.Join(userProfile, ".ozy", "mcp_servers.json"))
	}

	for _, cand := range candidates {
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			abs, err := filepath.Abs(cand)
			if err == nil {
				return abs
			}
			return cand
		}
	}

	return ""
}

// LoadConfigFile carga y parsea un archivo de configuración mcp_servers.json.
func LoadConfigFile(path string) ([]ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error leyendo archivo de configuración MCP %s: %w", path, err)
	}

	var root MCPServersConfig
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("error parseando JSON de %s: %w", path, err)
	}

	var configs []ServerConfig
	for id, entry := range root.MCPServers {
		cleanID := strings.TrimSpace(id)
		if cleanID == "" {
			continue
		}

		cmd := ResolveMCPCommand(strings.TrimSpace(entry.Command))
		if cmd == "" {
			continue
		}

		// Convertir variables de entorno de map a formato KEY=VALUE
		var envList []string
		for k, v := range entry.Env {
			// Expandir variables de entorno si contienen sintaxis %VAR% o $VAR
			expandedVal := os.ExpandEnv(v)
			envList = append(envList, fmt.Sprintf("%s=%s", k, expandedVal))
		}

		configs = append(configs, ServerConfig{
			ID:      cleanID,
			Name:    cleanID,
			Command: cmd,
			Args:    entry.Args,
			Env:     envList,
		})
	}

	return configs, nil
}

// ResolveMCPCommand resuelve ejecutables como 'npx', 'uvx' o 'node' en Windows
// comprobando si requieren extensiones .cmd, .bat o .exe en el PATH.
func ResolveMCPCommand(cmd string) string {
	if cmd == "" {
		return ""
	}

	if runtime.GOOS == "windows" {
		lower := strings.ToLower(cmd)
		// Si es npx, npm, pnpm o yarn sin extensión, buscar .cmd
		if !strings.HasSuffix(lower, ".exe") && !strings.HasSuffix(lower, ".cmd") && !strings.HasSuffix(lower, ".bat") {
			for _, ext := range []string{".cmd", ".exe", ".bat"} {
				if lp, err := exec.LookPath(cmd + ext); err == nil {
					return lp
				}
			}
		}
	}

	if lp, err := exec.LookPath(cmd); err == nil {
		return lp
	}

	return cmd
}
