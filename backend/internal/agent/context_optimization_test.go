package agent

import (
	"fmt"
	"strings"
	"testing"
)

func TestStripReasoningTokens(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "<think>El usuario quiere ver las ventanas activas.</think>Listo, aquí están las ventanas.",
			expected: "Listo, aquí están las ventanas.",
		},
		{
			input:    "<thought>Pensamiento interno profundo...</thought>He generado el reporte.",
			expected: "He generado el reporte.",
		},
		{
			input:    "Texto inicial. <think>monólogo</think> Respuesta final.",
			expected: "Texto inicial.  Respuesta final.",
		},
		{
			input:    "<think>Tag sin cerrar al final...",
			expected: "",
		},
	}

	for i, c := range cases {
		got := StripReasoningTokens(c.input)
		if got != c.expected {
			t.Errorf("case %d: expected %q, got %q", i, c.expected, got)
		}
	}
}

func TestWindowToolOutputForContext(t *testing.T) {
	// 1. Output corto permanece idéntico
	short := "Línea 1\nLínea 2\nLínea 3"
	if got := WindowToolOutputForContext(short, 25, 15); got != short {
		t.Errorf("expected short output to remain identical")
	}

	// 2. Output masivo (150 líneas)
	var lines []string
	for i := 1; i <= 150; i++ {
		lines = append(lines, fmt.Sprintf("Log line %d: detail execution trace timestamp 2026-10-04", i))
	}
	massive := strings.Join(lines, "\n")

	windowed := WindowToolOutputForContext(massive, 25, 15)
	if !strings.Contains(windowed, "Log line 1:") {
		t.Errorf("expected windowed output to contain head line 1")
	}
	if !strings.Contains(windowed, "Log line 150:") {
		t.Errorf("expected windowed output to contain tail line 150")
	}
	if !strings.Contains(windowed, "líneas intermedias omitidas para optimizar la ventana de contexto") {
		t.Errorf("expected windowed output to contain omission notice")
	}
	if len(windowed) >= len(massive) {
		t.Errorf("expected windowed output to be significantly smaller than massive output")
	}
}
