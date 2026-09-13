package audio

import (
	"testing"
)

func TestAudioDevicesEnumeration(t *testing.T) {
	devices, err := GetAudioDevices()
	if err != nil {
		t.Fatalf("Error enumerando dispositivos de audio: %v", err)
	}
	if len(devices) == 0 {
		t.Fatal("Se esperaba al menos 1 dispositivo de audio (real o fallback)")
	}

	hasInput := false
	hasOutput := false
	for _, d := range devices {
		if d.Type == "input" {
			hasInput = true
		}
		if d.Type == "output" {
			hasOutput = true
		}
	}

	if !hasInput || !hasOutput {
		t.Logf("Advertencia: inputs=%v outputs=%v", hasInput, hasOutput)
	}
}

func TestCalculateAudioEnergy(t *testing.T) {
	// 1. Buffer silencioso
	silence := make([]byte, 1024)
	energy := CalculateAudioEnergy(silence)
	if energy != 0.0 {
		t.Errorf("Energía esperada 0.0 para silencio, obtenido: %f", energy)
	}

	// 2. Buffer con señal de audio
	signal := make([]byte, 1024)
	for i := 0; i < len(signal)-1; i += 2 {
		signal[i] = 0x00
		signal[i+1] = 0x40 // ~16384 (amplitud media)
	}
	energySignal := CalculateAudioEnergy(signal)
	if energySignal <= 0.0 {
		t.Errorf("Energía esperada > 0.0 para señal de audio, obtenido: %f", energySignal)
	}

	if !IsVoiceActive(energySignal, 0.05) {
		t.Errorf("Se esperaba que la señal de audio supere el umbral de VAD")
	}
}

func TestDetectWakeWord(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"Hey Ozy", true},
		{"oye ozy cómo estás", true},
		{"ey ozi crea una carpeta", true},
		{"jei ozy abre el navegador", true},
		{"HOLA OZY", true},
		{"ozy", true},
		{"ozi", true},
		{"buenos días", false},
		{"esto es una prueba de texto normal", false},
		{"", false},
	}

	for _, c := range cases {
		res := DetectWakeWord(c.input)
		if res.Matched != c.expected {
			t.Errorf("DetectWakeWord(%q) = %v, esperado %v (Keyword: %s)", c.input, res.Matched, c.expected, res.Keyword)
		}
	}
}
