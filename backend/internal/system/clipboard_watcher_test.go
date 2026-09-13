package system

import (
	"testing"
)

type mockClipboardBroadcaster struct {
	lastEvent   string
	lastPayload interface{}
}

func (m *mockClipboardBroadcaster) BroadcastJSON(event string, payload interface{}) {
	m.lastEvent = event
	m.lastPayload = payload
}

func TestClassifyContent(t *testing.T) {
	tests := []struct {
		input       string
		expectedTyp string
		expectedAct string
		relevant    bool
	}{
		{
			input:       "Traceback (most recent call last):\n  File 'main.py', line 10, in <module>\nZeroDivisionError: division by zero",
			expectedTyp: "stack_trace",
			expectedAct: "analyze_and_heal",
			relevant:    true,
		},
		{
			input:       "https://github.com/ozyassist/core/pull/42",
			expectedTyp: "url",
			expectedAct: "deep_search",
			relevant:    true,
		},
		{
			input:       "func CalculateHash(prev string) string { return '' }",
			expectedTyp: "code_snippet",
			expectedAct: "explain_or_refactor",
			relevant:    true,
		},
		{
			input:       "hola mundo",
			expectedTyp: "text",
			expectedAct: "",
			relevant:    false,
		},
	}

	for _, tt := range tests {
		typ, act, rel := ClassifyContent(tt.input)
		if typ != tt.expectedTyp || act != tt.expectedAct || rel != tt.relevant {
			t.Errorf("Para input %q: obtenido (%s, %s, %v), esperado (%s, %s, %v)", tt.input, typ, act, rel, tt.expectedTyp, tt.expectedAct, tt.relevant)
		}
	}
}
