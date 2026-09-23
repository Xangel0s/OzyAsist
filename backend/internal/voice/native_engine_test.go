package voice

import (
	"encoding/binary"
	"testing"
)

func TestEncodeWAV_ValidHeader(t *testing.T) {
	pcm := []int16{0, 100, -100, 300, -300, 500}
	sampleRate := 16000
	wav := EncodeWAV(pcm, sampleRate)

	if len(wav) != 44+len(pcm)*2 {
		t.Fatalf("Tamaño esperado %d, obtuve %d", 44+len(pcm)*2, len(wav))
	}

	if string(wav[0:4]) != "RIFF" {
		t.Errorf("Esperaba 'RIFF', obtuve '%s'", string(wav[0:4]))
	}

	if string(wav[8:12]) != "WAVE" {
		t.Errorf("Esperaba 'WAVE', obtuve '%s'", string(wav[8:12]))
	}

	if string(wav[12:16]) != "fmt " {
		t.Errorf("Esperaba 'fmt ', obtuve '%s'", string(wav[12:16]))
	}

	channels := binary.LittleEndian.Uint16(wav[22:24])
	if channels != 1 {
		t.Errorf("Esperaba 1 canal mono, obtuve %d", channels)
	}

	rate := binary.LittleEndian.Uint32(wav[24:28])
	if rate != uint32(sampleRate) {
		t.Errorf("Esperaba rate %d, obtuve %d", sampleRate, rate)
	}

	if string(wav[36:40]) != "data" {
		t.Errorf("Esperaba 'data', obtuve '%s'", string(wav[36:40]))
	}

	dataLen := binary.LittleEndian.Uint32(wav[40:44])
	if dataLen != uint32(len(pcm)*2) {
		t.Errorf("Esperaba dataLen %d, obtuve %d", len(pcm)*2, dataLen)
	}
}

func TestCleanTranscribedText_WakeWords(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hey Ozy abre la calculadora", "abre la calculadora"},
		{"oye ozy, cierra el bloc de notas", "cierra el bloc de notas"},
		{"hola ozy: qué hora es", "qué hora es"},
		{"abre la calculadora", "abre la calculadora"},
		{"cierra bloc de notas.", "cierra bloc de notas."},
		// Alucinación repetitiva de Whisper en silencio:
		{"¿Qué es la verdad? ¿Qué es la verdad? ¿Qué es la verdad? ¿Qué es la verdad?", ""},
	}

	for _, tt := range tests {
		got := cleanTranscribedText(tt.input)
		if got != tt.expected {
			t.Errorf("cleanTranscribedText(%q) = %q; esperaba %q", tt.input, got, tt.expected)
		}
	}
}

func TestNativeVoiceEngine_Lifecycle(t *testing.T) {
	engine := NewNativeVoiceEngine(Config{}, nil)
	if engine.IsRunning() {
		t.Fatal("Esperaba motor no ejecutándose inicialmente")
	}

	hasSTT := engine.HasSTT()
	t.Logf("engine.HasSTT() = %v", hasSTT)

	engine.Stop()
	if engine.IsRunning() {
		t.Fatal("Esperaba motor detenido tras Stop()")
	}
}
