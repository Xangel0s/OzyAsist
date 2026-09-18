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

func TestBuildAgentSystemPrompt_IncludesProfileAndMemory(t *testing.T) {
	cleanup := setupTestMemoryAgent(t)
	defer cleanup()

	ctx := context.Background()
	userID := db.DefaultUserID()

	// 1. Configurar Perfil
	profileContent := "# Datos Clave\n- Lead Developer en crmgeofal\n- Prefiere Go puro"
	if err := db.UpdateUserProfile(userID, profileContent); err != nil {
		t.Fatalf("error updating profile: %v", err)
	}

	// 2. Configurar Recuerdos
	store := memory.DefaultStore()
	if err := store.UpsertFact(ctx, userID, memory.CategoryContext, "El puerto de desarrollo es 4000", 1.0); err != nil {
		t.Fatalf("error upserting fact: %v", err)
	}

	// 3. Generar Prompt
	params := AgentLoopParams{
		UserID:      userID,
		UserMessage: "Consulta sobre el puerto de desarrollo",
		VoiceMode:   false,
	}
	prompt := buildAgentSystemPrompt(params)

	if !strings.Contains(prompt, "=== PERFIL Y ROL DEL USUARIO ===") {
		t.Errorf("expected profile section in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Lead Developer en crmgeofal") {
		t.Errorf("expected profile content in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "=== RECUERDOS Y PREFERENCIAS APRENDIDAS DEL USUARIO ===") {
		t.Errorf("expected memories section in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "El puerto de desarrollo es 4000") {
		t.Errorf("expected memory content in prompt, got: %s", prompt)
	}
}

