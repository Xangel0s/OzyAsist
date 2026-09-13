package voice

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/ozyassist/backend/internal/audio"
)

type VoiceEngineState int

const (
	StateIdle VoiceEngineState = iota
	StateListeningHotword
	StateRecordingSpeech
	StateTranscribing
	StateExecutingAgent
	StateSpeaking
)

// NativeVoiceEngine orquesta la escucha nativa de audio de Windows (WASAPI)
// sin depender del navegador ni de Chrome Web Speech API.
type NativeVoiceEngine struct {
	mu                sync.Mutex
	state             VoiceEngineState
	pipeline          *AudioPipeline
	cancelFunc        context.CancelFunc
	isRunning         bool
	CommandDispatcher func(ctx context.Context, command string, onDelta func(string), onComplete func(string))

	// Callbacks para TUI o CLI
	OnStateChange func(state VoiceEngineState, desc string)
	OnTranscribed func(text string)
	OnAgentDelta  func(delta string)
	OnCompleted   func(response string)
}

func NewNativeVoiceEngine(cfg Config, dispatcher func(ctx context.Context, command string, onDelta func(string), onComplete func(string))) *NativeVoiceEngine {
	if cfg.GroqAPIKey == "" {
		cfg.GroqAPIKey = os.Getenv("GROQ_API_KEY")
	}
	if cfg.OpenAIAPIKey == "" {
		cfg.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	}

	return &NativeVoiceEngine{
		state:             StateIdle,
		pipeline:          NewAudioPipeline(cfg),
		CommandDispatcher: dispatcher,
	}
}

// HasSTT verifica si el motor cuenta con transcripción configurada
func (e *NativeVoiceEngine) HasSTT() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.pipeline.cfg.HasSTT()
}

// IsRunning indica si el bucle de audio está en ejecución activa
func (e *NativeVoiceEngine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isRunning
}

// UpdateConfig actualiza las claves y rutas de modelos en caliente
func (e *NativeVoiceEngine) UpdateConfig(cfg Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.pipeline.UpdateConfig(cfg)
}

// Start inicia el bucle en segundo plano
func (e *NativeVoiceEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return nil
	}
	if !e.pipeline.cfg.HasSTT() {
		e.mu.Unlock()
		return fmt.Errorf("no hay motor de transcripción STT configurado (falta GROQ_API_KEY o OPENAI_API_KEY)")
	}
	engineCtx, cancel := context.WithCancel(ctx)
	e.cancelFunc = cancel
	e.isRunning = true
	e.mu.Unlock()

	go e.runLoop(engineCtx)
	return nil
}

// Stop detiene el motor
func (e *NativeVoiceEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancelFunc != nil {
		e.cancelFunc()
		e.cancelFunc = nil
	}
	e.isRunning = false
	e.setState(StateIdle, "Motor de voz detenido")
}

func (e *NativeVoiceEngine) setState(s VoiceEngineState, desc string) {
	e.state = s
	if e.OnStateChange != nil {
		e.OnStateChange(s, desc)
	}
}

func (e *NativeVoiceEngine) runLoop(ctx context.Context) {
	log.Println("[NativeVoice] Iniciando motor de escucha continua...")

	// Inicializar contexto de audio MiniAudio nativo
	audioCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		log.Printf("[NativeVoice] Error inicializando contexto de audio: %v\n", err)
		return
	}
	defer audioCtx.Free()

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = 16000 // Whisper nativo (16 kHz mono)
	deviceConfig.Alsa.NoMMap = 1

	var capturedBuffer []int16
	var bufMu sync.Mutex
	isRecording := false
	silenceCounter := 0

	onRecvFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		samples := framecount * deviceConfig.Capture.Channels
		pcm := make([]int16, samples)
		var totalEnergy float64

		for i := uint32(0); i < samples; i++ {
			val := int16(pInputSamples[i*2]) | (int16(pInputSamples[i*2+1]) << 8)
			pcm[i] = val
			norm := float64(val) / 32768.0
			totalEnergy += norm * norm
		}

		rms := 0.0
		if samples > 0 {
			rms = math.Sqrt(totalEnergy / float64(samples))
		}

		bufMu.Lock()
		defer bufMu.Unlock()

		if isRecording {
			capturedBuffer = append(capturedBuffer, pcm...)
			// Detección de fin de habla (VAD de silencio)
			if rms < 0.02 {
				silenceCounter++
			} else {
				silenceCounter = 0
			}
		} else {
			// Ventana deslizante para detección de Wake Word
			capturedBuffer = append(capturedBuffer, pcm...)
			if len(capturedBuffer) > 16000*3 { // mantener últimos 3 segundos
				capturedBuffer = capturedBuffer[len(capturedBuffer)-16000*3:]
			}
		}
	}

	device, err := malgo.InitDevice(audioCtx.Context, deviceConfig, malgo.DeviceCallbacks{Data: onRecvFrames})
	if err != nil {
		log.Printf("[NativeVoice] Error inicializando dispositivo de captura: %v\n", err)
		e.setState(StateIdle, "Error: No se detectó micrófono de entrada")
		return
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		log.Printf("[NativeVoice] Error arrancando captura: %v\n", err)
		e.setState(StateIdle, "Error al iniciar captura de micrófono")
		return
	}

	e.setState(StateListeningHotword, "Escuchando Wake Word ('Hey Ozy')...")

	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			device.Stop()
			return

		case <-ticker.C:
			if e.state == StateExecutingAgent || e.state == StateSpeaking {
				continue // Ignorar mientras el agente procesa o habla
			}

			bufMu.Lock()
			currentSamples := make([]int16, len(capturedBuffer))
			copy(currentSamples, capturedBuffer)
			bufMu.Unlock()

			if len(currentSamples) < 16000 {
				continue
			}

			// VAD: Comprobar si hay energía vocal real en el fragmento capturado
			var sumSquares float64
			for _, s := range currentSamples {
				norm := float64(s) / 32768.0
				sumSquares += norm * norm
			}
			chunkRMS := math.Sqrt(sumSquares / float64(len(currentSamples)))
			if chunkRMS < 0.015 { // Silencio o ruido leve ambiente, no saturar la API
				continue
			}

			wavBytes := EncodeWAV(currentSamples, 16000)

			// Transcribir fragmento para buscar el Wake Word
			text, err := e.pipeline.TranscribeAudio(ctx, wavBytes)
			if err != nil {
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "no hay motor") {
					log.Printf("[NativeVoice] ⚠️ Error crítico STT: %v\n", err)
					e.setState(StateIdle, fmt.Sprintf("Voz detenida: %v", err))
					return
				}
				continue
			}
			if text == "" {
				continue
			}

			match := audio.DetectWakeWord(text)
			if match.Matched {
				log.Printf("[NativeVoice] 🎯 Wake Word detectado: '%s' (Frase: '%s')\n", match.Keyword, match.CleanPhrase)
				e.setState(StateRecordingSpeech, "¡Ozy despierto! Escuchando comando...")

				// Si el comando ya venía en la misma frase (ej: "Hey Ozy ordena mis carpetas")
				command := extractCommandAfterWakeWord(match.CleanPhrase, match.Keyword)
				if command != "" {
					e.processVoiceCommand(ctx, command)
					bufMu.Lock()
					capturedBuffer = nil
					bufMu.Unlock()
					e.setState(StateListeningHotword, "Escuchando Wake Word ('Hey Ozy')...")
					continue
				}

				// Si solo dijo "Hey Ozy", esperar 3 segundos adicionales para el comando
				bufMu.Lock()
				capturedBuffer = nil
				isRecording = true
				silenceCounter = 0
				bufMu.Unlock()

				time.Sleep(3 * time.Second)

				bufMu.Lock()
				isRecording = false
				cmdSamples := make([]int16, len(capturedBuffer))
				copy(cmdSamples, capturedBuffer)
				capturedBuffer = nil
				bufMu.Unlock()

				if len(cmdSamples) > 8000 {
					cmdWav := EncodeWAV(cmdSamples, 16000)
					cmdText, _ := e.pipeline.TranscribeAudio(ctx, cmdWav)
					if cmdText != "" {
						e.processVoiceCommand(ctx, cmdText)
					}
				}

				e.setState(StateListeningHotword, "Escuchando Wake Word ('Hey Ozy')...")
			}
		}
	}
}

func (e *NativeVoiceEngine) processVoiceCommand(ctx context.Context, command string) {
	log.Printf("[NativeVoice] Procesando orden por voz: %s\n", command)
	e.setState(StateExecutingAgent, fmt.Sprintf("Ejecutando orden: %s", command))

	if e.OnTranscribed != nil {
		e.OnTranscribed(command)
	}

	if e.CommandDispatcher != nil {
		e.CommandDispatcher(ctx, command, func(delta string) {
			if e.OnAgentDelta != nil {
				e.OnAgentDelta(delta)
			}
		}, func(fullResponse string) {
			if e.OnCompleted != nil {
				e.OnCompleted(fullResponse)
			}
			e.setState(StateSpeaking, "Hablando respuesta...")
			if wav, err := e.pipeline.SynthesizeSpeech(ctx, fullResponse); err == nil && len(wav) > 0 {
				_ = PlayWAV(wav)
			}
		})
	}
}

func extractCommandAfterWakeWord(phrase, keyword string) string {
	lowerPhrase := strings.ToLower(phrase)
	lowerKw := strings.ToLower(keyword)
	idx := strings.Index(lowerPhrase, lowerKw)
	if idx >= 0 {
		rem := strings.TrimSpace(lowerPhrase[idx+len(lowerKw):])
		rem = strings.TrimLeft(rem, ",.:;!? ")
		return rem
	}
	return ""
}

// EncodeWAV empaqueta muestras PCM int16 de 16kHz a contenedor WAV canónico
func EncodeWAV(pcm []int16, sampleRate int) []byte {
	dataLen := len(pcm) * 2
	totalLen := 36 + dataLen
	buf := new(bytes.Buffer)

	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(totalLen))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate*2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataLen))

	for _, s := range pcm {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}

	return buf.Bytes()
}

// TranscribeGroq realiza transcripción ultrarrápida vía Groq Whisper API (~150ms)
func TranscribeGroq(ctx context.Context, apiKey string, wavData []byte) (string, error) {
	if apiKey == "" {
		apiKey = os.Getenv("GROQ_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("no hay GROQ_API_KEY configurada")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(wavData); err != nil {
		return "", err
	}

	_ = writer.WriteField("model", "whisper-large-v3-turbo")
	_ = writer.WriteField("language", "es")
	_ = writer.WriteField("response_format", "json")
	_ = writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error groq API (%d): %s", resp.StatusCode, string(b))
	}

	var res struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return strings.TrimSpace(res.Text), nil
}

// TranscribeOpenAI realiza transcripción vía OpenAI Whisper-1
func TranscribeOpenAI(ctx context.Context, apiKey string, wavData []byte) (string, error) {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("no hay OPENAI_API_KEY configurada")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(wavData); err != nil {
		return "", err
	}

	_ = writer.WriteField("model", "whisper-1")
	_ = writer.WriteField("language", "es")
	_ = writer.WriteField("response_format", "json")
	_ = writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error openai whisper API (%d): %s", resp.StatusCode, string(b))
	}

	var res struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return strings.TrimSpace(res.Text), nil
}

// TranscribeFast mantiene compatibilidad hacia atrás
func TranscribeFast(ctx context.Context, apiKey string, wavData []byte) (string, error) {
	return TranscribeGroq(ctx, apiKey, wavData)
}
