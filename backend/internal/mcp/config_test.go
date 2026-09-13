package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp_servers.json")

	rawJSON := `{
		"mcpServers": {
			"sqlite": {
				"command": "uvx",
				"args": ["mcp-server-sqlite", "--db-path", "test.db"],
				"env": {
					"TEST_ENV": "active"
				}
			},
			"github": {
				"command": "npx",
				"args": ["-y", "@modelcontextprotocol/server-github"],
				"env": {
					"GITHUB_TOKEN": "secret_token"
				}
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(rawJSON), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	configs, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFile failed: %v", err)
	}

	if len(configs) != 2 {
		t.Fatalf("expected 2 server configs, got %d", len(configs))
	}

	foundSQLite := false
	foundGitHub := false
	for _, c := range configs {
		if c.ID == "sqlite" {
			foundSQLite = true
			if len(c.Args) != 3 {
				t.Errorf("expected 3 args for sqlite, got %d", len(c.Args))
			}
			if len(c.Env) != 1 || c.Env[0] != "TEST_ENV=active" {
				t.Errorf("unexpected env for sqlite: %v", c.Env)
			}
		}
		if c.ID == "github" {
			foundGitHub = true
			if len(c.Args) != 2 {
				t.Errorf("expected 2 args for github, got %d", len(c.Args))
			}
			if len(c.Env) != 1 || c.Env[0] != "GITHUB_TOKEN=secret_token" {
				t.Errorf("unexpected env for github: %v", c.Env)
			}
		}
	}

	if !foundSQLite {
		t.Errorf("sqlite config not found")
	}
	if !foundGitHub {
		t.Errorf("github config not found")
	}
}
