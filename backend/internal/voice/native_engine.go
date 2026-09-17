package voice

import (
	"context"
	"log"
	"sync"
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

// NativeVoiceEngine stub para evitar dependencias de cgo
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
	return false
}

func (e *NativeVoiceEngine) IsRunning() bool {
	return false
}

func (e *NativeVoiceEngine) UpdateConfig(cfg Config) {
}

func (e *NativeVoiceEngine) Start(ctx context.Context) error {
	log.Println("[NativeVoice] Motor de voz deshabilitado (stub).")
	return nil
}

func (e *NativeVoiceEngine) Stop() {
}

func (e *NativeVoiceEngine) setState(s VoiceEngineState, desc string) {
}

func EncodeWAV(pcm []int16, sampleRate int) []byte {
	return nil
}

func TranscribeGroq(ctx context.Context, apiKey string, wavData []byte) (string, error) {
	return "", nil
}

func TranscribeOpenAI(ctx context.Context, apiKey string, wavData []byte) (string, error) {
	return "", nil
}

func TranscribeFast(ctx context.Context, apiKey string, wavData []byte) (string, error) {
	return "", nil
}
