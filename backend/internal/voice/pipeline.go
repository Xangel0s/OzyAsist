package voice

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Config struct {
	WakeWordBinary string // Ruta a sherpa-onnx-wake-word o listener ligero
	WhisperBinary  string // Ruta a whisper.exe / whisper-cli
	PiperBinary    string // Ruta a piper.exe
	PiperModel     string // Ruta al modelo ONNX de voz en español
	GroqAPIKey     string // API Key para transcripción ultrarrápida en la nube con Groq (~150ms)
	OpenAIAPIKey   string // API Key de OpenAI para Whisper-1 fallback
}

// HasSTT indica si hay algún motor de transcripción configurado
func (c Config) HasSTT() bool {
	return c.GroqAPIKey != "" || c.OpenAIAPIKey != "" || c.WhisperBinary != "" || os.Getenv("GROQ_API_KEY") != "" || os.Getenv("OPENAI_API_KEY") != ""
}

// AutoDetectConfig detecta automáticamente binarios locales y variables de entorno
func AutoDetectConfig() Config {
	cfg := Config{
		GroqAPIKey:   os.Getenv("GROQ_API_KEY"),
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
	}

	// Autodetectar Piper TTS binario
	piperCandidates := []string{
		filepath.Join("tools", "piper", "piper", "piper.exe"),
		filepath.Join("backend", "tools", "piper", "piper", "piper.exe"),
		filepath.Join("..", "backend", "tools", "piper", "piper", "piper.exe"),
	}
	for _, p := range piperCandidates {
		if _, err := os.Stat(p); err == nil {
			cfg.PiperBinary = p
			break
		}
	}

	// Autodetectar Piper TTS modelo en español
	modelCandidates := []string{
		filepath.Join("tools", "piper", "es_ES-davefx-medium.onnx"),
		filepath.Join("backend", "tools", "piper", "es_ES-davefx-medium.onnx"),
		filepath.Join("..", "backend", "tools", "piper", "es_ES-davefx-medium.onnx"),
	}
	for _, m := range modelCandidates {
		if _, err := os.Stat(m); err == nil {
			cfg.PiperModel = m
			break
		}
	}

	// Autodetectar Whisper binario local
	whisperCandidates := []string{
		"whisper.exe",
		"whisper-cli.exe",
		filepath.Join("tools", "whisper", "whisper.exe"),
		filepath.Join("backend", "tools", "whisper", "whisper.exe"),
	}
	for _, w := range whisperCandidates {
		if _, err := os.Stat(w); err == nil {
			cfg.WhisperBinary = w
			break
		}
	}

	return cfg
}

type AudioPipeline struct {
	cfg        Config
	mu         sync.Mutex
	isSpeaking bool
}

func NewAudioPipeline(cfg Config) *AudioPipeline {
	return &AudioPipeline{cfg: cfg}
}

// UpdateConfig actualiza la configuración de forma dinámica en tiempo de ejecución
func (ap *AudioPipeline) UpdateConfig(cfg Config) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	if cfg.GroqAPIKey != "" {
		ap.cfg.GroqAPIKey = cfg.GroqAPIKey
	}
	if cfg.OpenAIAPIKey != "" {
		ap.cfg.OpenAIAPIKey = cfg.OpenAIAPIKey
	}
	if cfg.PiperBinary != "" {
		ap.cfg.PiperBinary = cfg.PiperBinary
	}
	if cfg.PiperModel != "" {
		ap.cfg.PiperModel = cfg.PiperModel
	}
	if cfg.WhisperBinary != "" {
		ap.cfg.WhisperBinary = cfg.WhisperBinary
	}
}

// SynthesizeSpeech genera un archivo de audio WAV en memoria a partir de texto
func (ap *AudioPipeline) SynthesizeSpeech(ctx context.Context, text string) ([]byte, error) {
	ap.mu.Lock()
	piperBin := ap.cfg.PiperBinary
	piperModel := ap.cfg.PiperModel
	if piperBin == "" || piperModel == "" {
		ap.mu.Unlock()
		return nil, fmt.Errorf("piper TTS no configurado")
	}
	ap.isSpeaking = true
	ap.mu.Unlock()

	defer func() {
		ap.mu.Lock()
		ap.isSpeaking = false
		ap.mu.Unlock()
	}()

	cmd := exec.CommandContext(ctx, piperBin,
		"--model", piperModel,
		"--output_file", "-", // salida por stdout
	)
	cmd.Stdin = strings.NewReader(text)

	var wavOut bytes.Buffer
	cmd.Stdout = &wavOut

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("fallo al sintetizar audio con Piper: %w", err)
	}

	return wavOut.Bytes(), nil
}

// TranscribeAudio procesa un fragmento de audio PCM/WAV y devuelve el texto
func (ap *AudioPipeline) TranscribeAudio(ctx context.Context, wavData []byte) (string, error) {
	ap.mu.Lock()
	groqKey := ap.cfg.GroqAPIKey
	openaiKey := ap.cfg.OpenAIAPIKey
	whisperBin := ap.cfg.WhisperBinary
	ap.mu.Unlock()

	if groqKey == "" {
		groqKey = os.Getenv("GROQ_API_KEY")
	}
	if openaiKey == "" {
		openaiKey = os.Getenv("OPENAI_API_KEY")
	}

	// 1. Intentar transcripción ultrarrápida con Groq Whisper (~150ms)
	if groqKey != "" {
		text, err := TranscribeGroq(ctx, groqKey, wavData)
		if err == nil && text != "" {
			return text, nil
		}
		if err != nil && openaiKey == "" && whisperBin == "" {
			return "", fmt.Errorf("error en Groq Whisper: %w", err)
		}
	}

	// 2. Fallback a OpenAI Whisper si hay OPENAI_API_KEY
	if openaiKey != "" {
		text, err := TranscribeOpenAI(ctx, openaiKey, wavData)
		if err == nil && text != "" {
			return text, nil
		}
		if err != nil && whisperBin == "" {
			return "", fmt.Errorf("error en OpenAI Whisper: %w", err)
		}
	}

	// 3. Fallback a binario local de whisper
	if whisperBin != "" {
		cmd := exec.CommandContext(ctx, whisperBin,
			"-m", "models/ggml-small.bin",
			"-l", "es",
			"-nt", // no timestamps
			"-",
		)
		cmd.Stdin = bytes.NewReader(wavData)

		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("fallo en transcripción Whisper local: %w", err)
		}

		return strings.TrimSpace(string(out)), nil
	}

	return "", fmt.Errorf("no hay motor de transcripción (STT) configurado: configure GROQ_API_KEY o OPENAI_API_KEY con /key groq <key>")
}

// IsSpeaking indica si el agente está actualmente emitiendo sonido (para control de Barge-in)
func (ap *AudioPipeline) IsSpeaking() bool {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	return ap.isSpeaking
}
