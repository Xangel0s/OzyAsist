package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
)

func setupTestMemoryAgent(t *testing.T) func() {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "ozy-memory-test-*")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test_ozy.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("error initializing test db: %v", err)
	}
	_ = db.EnsureDefaultUser()

	memory.InitContinuousMemory(db.DB, nil)

	return func() {
		db.Close()
		_ = os.RemoveAll(tempDir)
	}
}

func TestExecRememberFact(t *testing.T) {
	cleanup := setupTestMemoryAgent(t)
	defer cleanup()

	ctx := context.Background()
	inputJSON, _ := json.Marshal(map[string]interface{}{
		"category": "stack",
		"content":  "El proyecto crmgeofal usa Go 1.22 y arquitectura hexagonal",
	})

	tc := providers.ToolCall{
		ID:    uuid.NewString(),
		Name:  "remember_fact",
		Input: inputJSON,
	}

	out, success := execRememberFact(ctx, tc)
	if !success {
		t.Fatalf("execRememberFact failed: %s", out)
	}

	if !strings.Contains(out, "[MEMORIA]") || !strings.Contains(out, "STACK") {
		t.Errorf("expected [MEMORIA] and STACK in output, got: %s", out)
	}

	// Verificar que no contenga emojis
	for _, r := range out {
		if r > 127 && (r < 0x2000 || r > 0x2BFF) && (r < 0x2700 || r > 0x27BF) {
			// standard unicode text is fine, but check typical emoji ranges
			if r >= 0x1F300 && r <= 0x1F9FF {
				t.Errorf("output should not contain emojis, found rune: %U", r)
			}
		}
	}
}

func TestExecSearchMemory(t *testing.T) {
	cleanup := setupTestMemoryAgent(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Recordar un hecho
	store := memory.DefaultStore()
	err := store.UpsertFact(ctx, db.DefaultUserID(), memory.CategoryStack, "La base de datos de producción es SQLite WAL", 1.0)
	if err != nil {
		t.Fatalf("error upserting fact: %v", err)
	}

	// 2. Buscar hecho
	inputJSON, _ := json.Marshal(map[string]interface{}{
		"query": "SQLite",
		"limit": 5,
	})

	tc := providers.ToolCall{
		ID:    uuid.NewString(),
		Name:  "search_memory",
		Input: inputJSON,
	}

	out, success := execSearchMemory(ctx, tc)
	if !success {
		t.Fatalf("execSearchMemory failed: %s", out)
	}

	if !strings.Contains(out, "[MEMORIA]") || !strings.Contains(out, "SQLite WAL") {
		t.Errorf("expected [MEMORIA] and SQLite WAL in output, got: %s", out)
	}
}

func TestExecUpdateUserProfile(t *testing.T) {
	cleanup := setupTestMemoryAgent(t)
	defer cleanup()

	ctx := context.Background()
	profileContent := "# Perfil\n- Rol: Arquitecto de Software\n- Enfoque: Geofal CRM"

	inputJSON, _ := json.Marshal(map[string]interface{}{
		"profile_md": profileContent,
	})

	tc := providers.ToolCall{
		ID:    uuid.NewString(),
		Name:  "update_user_profile",
		Input: inputJSON,
	}

	out, success := execUpdateUserProfile(ctx, tc)
	if !success {
		t.Fatalf("execUpdateUserProfile failed: %s", out)
	}

	if !strings.Contains(out, "[PERFIL]") {
		t.Errorf("expected [PERFIL] in output, got: %s", out)
	}

	user, err := db.GetUser(db.DefaultUserID())
	if err != nil {
		t.Fatalf("error retrieving user: %v", err)
	}

	if user.ProfileMd != profileContent {
		t.Errorf("expected ProfileMd '%s', got '%s'", profileContent, user.ProfileMd)
	}
}
