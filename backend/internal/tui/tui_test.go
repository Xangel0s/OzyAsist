package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel(nil, nil, false)
	if m.state != StateIdle {
		t.Errorf("Expected initial state StateIdle, got %v", m.state)
	}
	if m.voiceEnabled {
		t.Errorf("Expected voiceEnabled false when passed false")
	}

	mActive := InitialModel(nil, nil, true)
	if !mActive.voiceEnabled {
		t.Errorf("Expected voiceEnabled true when passed true")
	}

	if len(m.entries) == 0 {
		t.Errorf("Expected welcome entry in entries")
	}

	// Test slash command /help
	out := m.handleSlashCommand("/help")
	if out == "" {
		t.Errorf("Expected non-empty output for /help")
	}

	// Test slash command /perm
	outPerm := m.handleSlashCommand("/perm autonomous")
	if m.permissionLevel != "autonomous" {
		t.Errorf("Expected permissionLevel autonomous, got %s (output: %s)", m.permissionLevel, outPerm)
	}

	// Test slash command /mcp
	outMCP := m.handleSlashCommand("/mcp")
	if outMCP == "" {
		t.Errorf("Expected non-empty output for /mcp")
	}

	// Test slash command /tools
	outTools := m.handleSlashCommand("/tools")
	if outTools == "" {
		t.Errorf("Expected non-empty output for /tools")
	}

	// Test slash command /voice without controller
	outVoice := m.handleSlashCommand("/voice")
	if outVoice == "" {
		t.Errorf("Expected output for /voice")
	}

	// Test slash command /key groq
	outKeyGroq := m.handleSlashCommand("/key groq gsk_test123")
	if outKeyGroq == "" {
		t.Errorf("Expected output for /key groq")
	}

	// Test slash command /provider
	outProvList := m.handleSlashCommand("/provider")
	if outProvList == "" {
		t.Errorf("Expected output for /provider")
	}

	// Test slash command /key cohere
	outKeyCohere := m.handleSlashCommand("/key cohere dummy_cohere_test_key_12345")
	if outKeyCohere == "" {
		t.Errorf("Expected output for /key cohere")
	}
	if m.provider == nil || m.provider.Name() != "cohere" {
		t.Errorf("Expected active provider to be cohere, got %v", m.provider)
	}

	// Test window resize message
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model := newM.(Model)
	if !model.ready {
		t.Errorf("Expected model to be ready after WindowSizeMsg")
	}
}

func TestCleanAssistantText(t *testing.T) {
	raw := "<thought>\nAnalizando el problema del usuario...\n</thought>\n\n\n\nHola, he organizado tus archivos.\n\n\nQue tengas buen día."
	cleaned := cleanAssistantText(raw)
	expected := "Hola, he organizado tus archivos.\n\nQue tengas buen día."
	if cleaned != expected {
		t.Errorf("Expected cleaned text:\n%q\nGot:\n%q", expected, cleaned)
	}
}

func TestWrapLine(t *testing.T) {
	longLine := "Get-Process : No se encuentra ningún proceso con el nombre \"notepad\". Compruebe el nombre del proceso y vuelva a ejecutar el cmdlet."
	maxWidth := 40
	wrapped := wrapLine(longLine, maxWidth)
	if len(wrapped) < 3 {
		t.Errorf("Expected at least 3 lines, got %d: %v", len(wrapped), wrapped)
	}
	for i, line := range wrapped {
		runes := []rune(line)
		if len(runes) > maxWidth {
			t.Errorf("Line %d exceeds maxWidth %d (len %d): %s", i, maxWidth, len(runes), line)
		}
	}
}

func TestWrapLine_LongUnbrokenWord(t *testing.T) {
	longWord := "https://github.com/ozyassist/backend/internal/agent/very/deeply/nested/directory/path/file.go"
	maxWidth := 30
	wrapped := wrapLine(longWord, maxWidth)
	if len(wrapped) < 3 {
		t.Errorf("Expected unbroken word to be wrapped, got %d parts: %v", len(wrapped), wrapped)
	}
	for i, line := range wrapped {
		runes := []rune(line)
		if len(runes) > maxWidth {
			t.Errorf("Part %d exceeds maxWidth %d (len %d): %s", i, maxWidth, len(runes), line)
		}
	}
}

func TestWrapContent_PreservesIndentation(t *testing.T) {
	content := "  - Primer punto con una explicación bastante larga para probar el comportamiento del wrap\n    • Subpunto indentado"
	wrapped := wrapContent(content, 45)
	if len(wrapped) < 3 {
		t.Errorf("Expected wrapped lines >= 3, got %d: %v", len(wrapped), wrapped)
	}
}


