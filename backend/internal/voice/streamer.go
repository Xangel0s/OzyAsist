package voice

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

var (
	reMarkdownLinks = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	reCodeBlocks    = regexp.MustCompile("(?s)```.*?```")
	reInlineCode    = regexp.MustCompile("`([^`]+)`")
	reBoldItalic    = regexp.MustCompile(`[*_]{1,3}([^*_]+)[*_]{1,3}`)
	reHeaders       = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reBullets       = regexp.MustCompile(`(?m)^[\s*\-+]\s+`)
	reURLs          = regexp.MustCompile(`https?://\S+`)
	reNumberedList  = regexp.MustCompile(`(?m)^\s*\d+[\.)]\s+`)
	reMultiSpace    = regexp.MustCompile(`[ \t]+`)
)

// CleanForSpeech normaliza y limpia texto con formato Markdown para que la síntesis
// de voz suene completamente natural y fluida en español sin pronunciar símbolos.
func CleanForSpeech(text string) string {
	s := text
	// Reemplazar bloques de código extensos
	s = reCodeBlocks.ReplaceAllString(s, "bloque de código omitido")
	// Extraer texto legible de enlaces markdown [Texto](URL)
	s = reMarkdownLinks.ReplaceAllString(s, "$1")
	// Reemplazar URLs directas
	s = reURLs.ReplaceAllString(s, "enlace web")
	// Código en línea
	s = reInlineCode.ReplaceAllString(s, "$1")
	// Negrita y cursiva
	s = reBoldItalic.ReplaceAllString(s, "$1")
	// Encabezados
	s = reHeaders.ReplaceAllString(s, "")
	// Viñetas y listas numeradas
	s = reBullets.ReplaceAllString(s, "")
	s = reNumberedList.ReplaceAllString(s, "")

	// Eliminar caracteres y símbolos no hablables
	s = strings.ReplaceAll(s, "•", "")
	s = strings.ReplaceAll(s, "~", "")
	s = strings.ReplaceAll(s, "|", " ")
	s = strings.ReplaceAll(s, "#", "")
	s = strings.ReplaceAll(s, "[", "")
	s = strings.ReplaceAll(s, "]", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, ">", "")
	s = strings.ReplaceAll(s, "`", "")

	// Limpiar saltos de línea repetidos
	lines := strings.Split(s, "\n")
	var cleanedLines []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}
	s = strings.Join(cleanedLines, " ")
	s = reMultiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// SentenceStreamer agrupa deltas de tokens en tiempo real y detecta límites de oraciones
// para emitir fragmentos de voz de baja latencia (~300-400ms) sin esperar la respuesta completa.
type SentenceStreamer struct {
	mu         sync.Mutex
	buffer     strings.Builder
	onSentence func(sentence string)
}

// NewSentenceStreamer inicializa un nuevo procesador de oraciones en streaming
func NewSentenceStreamer(onSentence func(sentence string)) *SentenceStreamer {
	return &SentenceStreamer{
		onSentence: onSentence,
	}
}

// Feed procesa un nuevo fragmento de texto (token delta) y dispara onSentence cuando
// una frase u oración se completa según signos de puntuación.
func (ss *SentenceStreamer) Feed(delta string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.buffer.WriteString(delta)
	content := ss.buffer.String()

	for {
		splitIdx := -1
		// Detectar terminadores de oración
		for i := 0; i < len(content)-1; i++ {
			ch := content[i]
			next := content[i+1]

			// Fin de oración estándar: punto, interrogación, exclamación seguido de espacio o salto de línea
			if (ch == '.' || ch == '?' || ch == '!') && (next == ' ' || next == '\n' || next == '\t') {
				splitIdx = i + 1
				break
			}
			// Salto de línea doble o fin de párrafo
			if ch == '\n' && next == '\n' {
				splitIdx = i + 1
				break
			}
			// Dos puntos seguido de salto de línea (introducción a una lista)
			if ch == ':' && next == '\n' {
				splitIdx = i + 1
				break
			}
		}

		// Si el buffer ya es extenso (> 100 caracteres) y hay una coma seguida de espacio, dividir para mantener baja latencia
		if splitIdx == -1 && len(content) > 100 {
			for i := 40; i < len(content)-1; i++ {
				if content[i] == ',' && (content[i+1] == ' ' || content[i+1] == '\n') {
					splitIdx = i + 1
					break
				}
			}
		}

		if splitIdx == -1 {
			break
		}

		sentence := content[:splitIdx]
		content = strings.TrimLeft(content[splitIdx:], " \t\n\r")
		ss.buffer.Reset()
		ss.buffer.WriteString(content)

		cleaned := CleanForSpeech(sentence)
		if len(cleaned) >= 2 && ss.onSentence != nil {
			ss.onSentence(cleaned)
		}
	}
}

// Flush vacía el contenido restante en el buffer al finalizar la generación del LLM
func (ss *SentenceStreamer) Flush() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	content := ss.buffer.String()
	ss.buffer.Reset()

	cleaned := CleanForSpeech(content)
	if len(cleaned) >= 2 && ss.onSentence != nil {
		ss.onSentence(cleaned)
	}
}

// LocalSpeakerQueue gestiona una cola secuencial de oraciones para reproducirlas
// en tiempo real por los altavoces locales mediante Windows SAPI o Piper TTS con capacidad de interrupción (barge-in).
type LocalSpeakerQueue struct {
	queueCh       chan string
	parentCtx     context.Context
	parentCancel  context.CancelFunc
	mu            sync.Mutex
	currentCancel context.CancelFunc
	isPlaying     bool
}

// NewLocalSpeakerQueue crea una nueva cola de reproducción local en segundo plano
func NewLocalSpeakerQueue(parentCtx context.Context) *LocalSpeakerQueue {
	ctx, cancel := context.WithCancel(parentCtx)
	sq := &LocalSpeakerQueue{
		queueCh:      make(chan string, 50),
		parentCtx:    ctx,
		parentCancel: cancel,
	}

	go sq.worker()
	return sq
}

// Enqueue agrega una oración a la cola de pronunciación
func (sq *LocalSpeakerQueue) Enqueue(sentence string) {
	s := strings.TrimSpace(sentence)
	if s == "" {
		return
	}
	select {
	case sq.queueCh <- s:
	default:
		// Cola llena, descartar para evitar desborde
	}
}

// Cancel detiene inmediatamente la reproducción actual y vacía la cola pendiente (Barge-in)
func (sq *LocalSpeakerQueue) Cancel() {
	sq.mu.Lock()
	if sq.currentCancel != nil {
		sq.currentCancel()
		sq.currentCancel = nil
	}
	sq.isPlaying = false
	// Vaciar canal de oraciones pendientes
drain:
	for {
		select {
		case <-sq.queueCh:
		default:
			break drain
		}
	}
	sq.mu.Unlock()
}

// Close finaliza el worker de reproducción
func (sq *LocalSpeakerQueue) Close() {
	sq.Cancel()
	sq.parentCancel()
}

// IsPlaying indica si hay audio reproduciéndose activamente
func (sq *LocalSpeakerQueue) IsPlaying() bool {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	return sq.isPlaying
}

func (sq *LocalSpeakerQueue) worker() {
	for {
		select {
		case <-sq.parentCtx.Done():
			return
		case sentence, ok := <-sq.queueCh:
			if !ok {
				return
			}
			if sentence == "" {
				continue
			}

			speakCtx, cancel := context.WithCancel(sq.parentCtx)
			sq.mu.Lock()
			sq.currentCancel = cancel
			sq.isPlaying = true
			sq.mu.Unlock()

			sq.speak(speakCtx, sentence)

			sq.mu.Lock()
			sq.currentCancel = nil
			sq.isPlaying = false
			sq.mu.Unlock()
		}
	}
}

func (sq *LocalSpeakerQueue) speak(ctx context.Context, text string) {
	if ctx.Err() != nil {
		return
	}

	// 1. Intentar con Piper TTS local si está configurado
	cfg := AutoDetectConfig()
	if cfg.PiperBinary != "" && cfg.PiperModel != "" {
		pipe := NewAudioPipeline(cfg)
		wav, err := pipe.SynthesizeSpeech(ctx, text)
		if err == nil && len(wav) > 0 {
			_ = PlayWAV(wav)
			return
		}
	}

	// 2. Fallback nativo universal a Windows SAPI (Microsoft Helena Desktop / voces en español)
	escText := strings.ReplaceAll(text, "'", "''")
	psCmd := fmt.Sprintf(`
		Add-Type -AssemblyName System.Speech
		$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer
		$esVoice = $synth.GetInstalledVoices() | Where-Object { $_.VoiceInfo.Culture -like 'es*' } | Select-Object -First 1
		if ($esVoice) {
			$synth.SelectVoice($esVoice.VoiceInfo.Name)
		}
		$synth.Speak('%s')
	`, escText)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	_ = cmd.Run()
}
