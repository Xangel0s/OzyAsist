package voice

import (
	"context"
	"testing"
)

func TestAudioPipeline_Unconfigured(t *testing.T) {
	ap := NewAudioPipeline(Config{})

	_, err := ap.SynthesizeSpeech(context.Background(), "Hola mundo")
	if err == nil {
		t.Fatal("Esperaba error con Piper no configurado")
	}

	_, err = ap.TranscribeAudio(context.Background(), []byte("fake_audio"))
	if err == nil {
		t.Fatal("Esperaba error con Whisper no configurado")
	}

	if ap.IsSpeaking() {
		t.Fatal("Esperaba IsSpeaking=false")
	}
}

func TestConfig_HasSTT(t *testing.T) {
	cEmpty := Config{}
	if cEmpty.HasSTT() {
		t.Errorf("Esperaba HasSTT=false para configuración vacía")
	}

	cGroq := Config{GroqAPIKey: "gsk_test123"}
	if !cGroq.HasSTT() {
		t.Errorf("Esperaba HasSTT=true con GroqAPIKey")
	}

	cOpenAI := Config{OpenAIAPIKey: "sk-test123"}
	if !cOpenAI.HasSTT() {
		t.Errorf("Esperaba HasSTT=true con OpenAIAPIKey")
	}

	cWhisper := Config{WhisperBinary: "whisper.exe"}
	if !cWhisper.HasSTT() {
		t.Errorf("Esperaba HasSTT=true con WhisperBinary")
	}
}

func TestAutoDetectConfig(t *testing.T) {
	cfg := AutoDetectConfig()
	// Si estamos en el directorio backend, piper.exe debe ser detectado
	if cfg.PiperBinary != "" {
		t.Logf("Piper autodetectado: %s", cfg.PiperBinary)
	}
	if cfg.PiperModel != "" {
		t.Logf("Modelo Piper autodetectado: %s", cfg.PiperModel)
	}
}

