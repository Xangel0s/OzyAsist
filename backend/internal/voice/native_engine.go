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
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
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

// NativeVoiceEngine gestiona la captura continua de micrófono, detección VAD y transcripción STT
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
	return &NativeVoiceEngine{
		state:             StateIdle,
		pipeline:          NewAudioPipeline(cfg),
		CommandDispatcher: dispatcher,
	}
}

func (e *NativeVoiceEngine) HasSTT() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pipeline != nil && e.pipeline.cfg.HasSTT() {
		return true
	}
	return hasLocalSTT()
}

func (e *NativeVoiceEngine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isRunning
}

func (e *NativeVoiceEngine) UpdateConfig(cfg Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pipeline != nil {
		e.pipeline.UpdateConfig(cfg)
	}
}

func (e *NativeVoiceEngine) setState(s VoiceEngineState, desc string) {
	e.mu.Lock()
	e.state = s
	onStateChange := e.OnStateChange
	e.mu.Unlock()

	if onStateChange != nil {
		onStateChange(s, desc)
	}
}

// detectDefaultMicDevice detecta el primer micrófono disponible mediante ffmpeg dshow
func detectDefaultMicDevice() string {
	cmd := exec.Command("ffmpeg", "-nostdin", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	cmd.Stdin = strings.NewReader("")
	out, _ := cmd.CombinedOutput()
	lines := strings.Split(string(out), "\n")
	re := regexp.MustCompile(`(?i)"([^"]+)"\s+\(audio\)`)
	for _, line := range lines {
		if m := re.FindStringSubmatch(line); len(m) > 1 {
			return m[1]
		}
	}
	return "Varios micrófonos (Intel® Smart Sound Technology (Intel® SST))"
}

func (e *NativeVoiceEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return nil
	}

	engineCtx, cancel := context.WithCancel(ctx)
	e.cancelFunc = cancel
	e.isRunning = true
	e.mu.Unlock()

	e.setState(StateListeningHotword, "Escuchando micrófono...")

	go e.micListenLoop(engineCtx)
	return nil
}

func (e *NativeVoiceEngine) Stop() {
	e.mu.Lock()
	if !e.isRunning {
		e.mu.Unlock()
		return
	}
	if e.cancelFunc != nil {
		e.cancelFunc()
		e.cancelFunc = nil
	}
	e.isRunning = false
	e.state = StateIdle
	e.mu.Unlock()

	e.setState(StateIdle, "Modo voz detenido")
}

// micListenLoop captura audio continuo desde el micrófono con ffmpeg y aplica VAD por energía
func (e *NativeVoiceEngine) micListenLoop(ctx context.Context) {
	micName := detectDefaultMicDevice()
	log.Printf("[NativeVoice] Iniciando captura de micrófono: %s", micName)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-nostdin",
			"-loglevel", "quiet",
			"-y",
			"-f", "dshow",
			"-i", fmt.Sprintf("audio=%s", micName),
			"-ar", "16000",
			"-ac", "1",
			"-f", "s16le",
			"pipe:1",
		)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
		cmd.Stdin = strings.NewReader("")

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("[NativeVoice] Error abriendo stdout pipe de ffmpeg: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if err := cmd.Start(); err != nil {
			log.Printf("[NativeVoice] Error iniciando ffmpeg: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Buffer para leer 40ms = 640 samples = 1280 bytes
		chunkSize := 1280
		buf := make([]byte, chunkSize)
		var speechSamples []int16
		isSpeaking := false
		silenceFrames := 0

		// Threshold de energía RMS para detectar voz humana
		rmsThreshold := 550.0

		for {
			select {
			case <-ctx.Done():
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				return
			default:
			}

			// Eco-supresión: Si Ozy está hablando por altavoz (TTS), descartar mic
			if e.pipeline != nil && e.pipeline.IsSpeaking() {
				_, _ = io.ReadFull(stdout, buf)
				speechSamples = nil
				isSpeaking = false
				silenceFrames = 0
				continue
			}

			_, err := io.ReadFull(stdout, buf)
			if err != nil {
				break
			}

			// Convertir bytes a int16 PCM
			samples := make([]int16, len(buf)/2)
			sumSquares := 0.0
			for i := 0; i < len(samples); i++ {
				s := int16(binary.LittleEndian.Uint16(buf[i*2 : i*2+2]))
				samples[i] = s
				sumSquares += float64(s) * float64(s)
			}
			rms := math.Sqrt(sumSquares / float64(len(samples)))

			if rms > rmsThreshold {
				if !isSpeaking {
					isSpeaking = true
					e.setState(StateRecordingSpeech, "Detectando voz...")
				}
				speechSamples = append(speechSamples, samples...)
				silenceFrames = 0
				// Límite de seguridad: máx 10 segundos continuos
				if len(speechSamples) > 16000*10 {
					isSpeaking = false
				}
			} else if isSpeaking {
				silenceFrames++
				speechSamples = append(speechSamples, samples...)
				// 18 frames * 40ms = 720ms de silencio tras hablar
				if silenceFrames >= 18 {
					isSpeaking = false
				}
			}

			if !isSpeaking && len(speechSamples) > 0 {
				// Al menos 0.6s de audio (9600 samples)
				if len(speechSamples) >= 9600 {
					recorded := make([]int16, len(speechSamples))
					copy(recorded, speechSamples)
					go e.processSpeech(ctx, recorded)
				}
				speechSamples = nil
				silenceFrames = 0
				e.setState(StateListeningHotword, "Escuchando...")
			}
		}

		_ = cmd.Wait()
		time.Sleep(500 * time.Millisecond)
	}
}

func (e *NativeVoiceEngine) processSpeech(ctx context.Context, pcm []int16) {
	e.setState(StateTranscribing, "Transcribiendo voz...")
	wavData := EncodeWAV(pcm, 16000)

	text, err := e.pipeline.TranscribeAudio(ctx, wavData)
	if err != nil {
		log.Printf("[NativeVoice] Error transcribiendo: %v", err)
		e.setState(StateListeningHotword, "Escuchando...")
		return
	}

	cleaned := cleanTranscribedText(text)
	if cleaned == "" {
		e.setState(StateListeningHotword, "Escuchando...")
		return
	}

	log.Printf("[NativeVoice] Texto reconocido: %s", cleaned)
	if e.OnTranscribed != nil {
		e.OnTranscribed(cleaned)
	}

	if e.CommandDispatcher != nil {
		e.setState(StateExecutingAgent, "Ejecutando orden...")
		e.CommandDispatcher(ctx, cleaned, e.OnAgentDelta, e.OnCompleted)
	}

	e.setState(StateListeningHotword, "Escuchando...")
}

func cleanTranscribedText(text string) string {
	cleaned := strings.TrimSpace(text)
	lower := strings.ToLower(cleaned)

	// Filtrar alucinaciones repetitivas de Whisper en silencio o ruido (frases idénticas repetidas)
	parts := strings.FieldsFunc(lower, func(r rune) bool {
		return r == '?' || r == '!' || r == '.' || r == ',' || r == ';'
	})
	if len(parts) >= 3 {
		seen := make(map[string]int)
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if len(trimmed) > 3 {
				seen[trimmed]++
				if seen[trimmed] >= 3 {
					return ""
				}
			}
		}
	}

	// También comprobar repetición de palabras individuales (ej: "s s s s s")
	words := strings.Fields(lower)
	if len(words) >= 4 {
		counts := make(map[string]int)
		for _, w := range words {
			counts[w]++
		}
		for _, c := range counts {
			if float64(c)/float64(len(words)) > 0.4 {
				return ""
			}
		}
	}

	// Limpiar Wake Words si fueron pronunciadas al inicio
	prefixes := []string{
		"hey ozy", "oye ozy", "hey osi", "oye osi", "hola ozy",
		"hey ozi", "oye ozi", "ozy",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			cleaned = strings.TrimSpace(cleaned[len(p):])
			cleaned = strings.TrimLeft(cleaned, ",.?!:; ")
			break
		}
	}

	return strings.TrimSpace(cleaned)
}

func EncodeWAV(pcm []int16, sampleRate int) []byte {
	numChannels := 1
	bitsPerSample := 16
	byteRate := sampleRate * numChannels * (bitsPerSample / 8)
	blockAlign := numChannels * (bitsPerSample / 8)
	dataSize := len(pcm) * 2
	chunkSize := 36 + dataSize

	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(chunkSize))
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(numChannels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(byteRate))
	_ = binary.Write(buf, binary.LittleEndian, uint16(blockAlign))
	_ = binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))

	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataSize))
	for _, sample := range pcm {
		_ = binary.Write(buf, binary.LittleEndian, sample)
	}

	return buf.Bytes()
}

func TranscribeGroq(ctx context.Context, apiKey string, wavData []byte) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("groq API key requerida")
	}
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

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
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/audio/transcriptions", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var res struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Text), nil
}

func TranscribeOpenAI(ctx context.Context, apiKey string, wavData []byte) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("openai API key requerida")
	}
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

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
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/transcriptions", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var res struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Text), nil
}

// TranscribeLocal ejecuta el script Python local de Whisper en training/
func TranscribeLocal(ctx context.Context, wavData []byte) (string, error) {
	pythonCandidates := []string{
		filepath.Join("training", "venv", "Scripts", "python.exe"),
		filepath.Join("..", "training", "venv", "Scripts", "python.exe"),
		filepath.Join("..", "..", "training", "venv", "Scripts", "python.exe"),
	}
	var pythonPath string
	for _, p := range pythonCandidates {
		if _, err := os.Stat(p); err == nil {
			pythonPath = p
			break
		}
	}
	if pythonPath == "" {
		return "", fmt.Errorf("entorno de python local no encontrado")
	}

	scriptCandidates := []string{
		filepath.Join("training", "stt_transcribe.py"),
		filepath.Join("..", "training", "stt_transcribe.py"),
		filepath.Join("..", "..", "training", "stt_transcribe.py"),
	}
	var scriptPath string
	for _, s := range scriptCandidates {
		if _, err := os.Stat(s); err == nil {
			scriptPath = s
			break
		}
	}
	if scriptPath == "" {
		return "", fmt.Errorf("script de transcripción local no encontrado")
	}

	tempWav := filepath.Join(os.TempDir(), fmt.Sprintf("ozy_voice_%d.wav", time.Now().UnixNano()))
	if err := os.WriteFile(tempWav, wavData, 0644); err != nil {
		return "", err
	}
	defer os.Remove(tempWav)

	cmd := exec.CommandContext(ctx, pythonPath, scriptPath, tempWav)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error ejecutando STT local: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func TranscribeFast(ctx context.Context, apiKey string, wavData []byte) (string, error) {
	if apiKey != "" {
		return TranscribeGroq(ctx, apiKey, wavData)
	}
	return TranscribeLocal(ctx, wavData)
}
