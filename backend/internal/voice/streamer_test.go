package voice

import (
	"testing"
)

func TestCleanForSpeech(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Markdown links and formatting",
			input:    "Hola, consulta la [documentación oficial](https://example.com) para más **detalles** y *consejos*.",
			expected: "Hola, consulta la documentación oficial para más detalles y consejos.",
		},
		{
			name:     "Code block and inline code",
			input:    "Puedes ejecutar `ls -la` o revisar el archivo:\n```go\nfunc main() {}\n```\nEso es todo.",
			expected: "Puedes ejecutar ls -la o revisar el archivo: bloque de código omitido Eso es todo.",
		},
		{
			name:     "Headers, bullets, and numbered lists",
			input:    "# Diagnóstico\n- Primero revisa el disco\n* Luego la memoria\n1. Paso uno\n2. Paso dos",
			expected: "Diagnóstico Primero revisa el disco Luego la memoria Paso uno Paso dos",
		},
		{
			name:     "Status brackets and special characters",
			input:    "[OK] Sistema operativo estable • RAM libre: 8GB ~ uso normal | CPU normal #test",
			expected: "OK Sistema operativo estable RAM libre: 8GB uso normal CPU normal test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CleanForSpeech(tc.input)
			if got != tc.expected {
				t.Errorf("CleanForSpeech() mismatch\nGot:      %q\nExpected: %q", got, tc.expected)
			}
		})
	}
}

func TestSentenceStreamer(t *testing.T) {
	var collected []string
	streamer := NewSentenceStreamer(func(sentence string) {
		collected = append(collected, sentence)
	})

	// Simular streaming de tokens delta
	tokens := []string{
		"Hola", " a", " todos.", " ",
		"Este", " es", " un", " mensaje", " de", " prueba!", " ",
		"¿Cómo", " estás", " hoy?", " ",
		"Todo", " se", " encuentra", " en", " orden",
	}

	for _, tok := range tokens {
		streamer.Feed(tok)
	}

	// Deberían haberse detectado 3 oraciones completas
	if len(collected) != 3 {
		t.Fatalf("Se esperaban 3 oraciones completas antes de Flush, se obtuvieron %d: %v", len(collected), collected)
	}

	if collected[0] != "Hola a todos." {
		t.Errorf("Oración 0 errónea: %q", collected[0])
	}
	if collected[1] != "Este es un mensaje de prueba!" {
		t.Errorf("Oración 1 errónea: %q", collected[1])
	}
	if collected[2] != "¿Cómo estás hoy?" {
		t.Errorf("Oración 2 errónea: %q", collected[2])
	}

	// Flush del remanente
	streamer.Flush()
	if len(collected) != 4 {
		t.Fatalf("Se esperaban 4 oraciones en total tras Flush, se obtuvieron %d: %v", len(collected), collected)
	}
	if collected[3] != "Todo se encuentra en orden" {
		t.Errorf("Oración 3 tras flush errónea: %q", collected[3])
	}
}

func TestSentenceStreamerLongSentenceSplit(t *testing.T) {
	var collected []string
	streamer := NewSentenceStreamer(func(sentence string) {
		collected = append(collected, sentence)
	})

	longClause := "Este es un fragmento de texto muy largo que continúa sin ningún punto final pero contiene varias comas intermedias, de modo que el sistema de voz pueda dividirlo y hablarlo sin esperar indefinidamente."
	
	streamer.Feed(longClause)
	streamer.Flush()

	if len(collected) < 2 {
		t.Errorf("Se esperaba que la oración larga con comas se dividiera en al menos 2 oraciones, se obtuvieron: %d (%v)", len(collected), collected)
	}
}
