package audio

import (
	"fmt"
	"log"

	"github.com/Picovoice/porcupine/binding/go/v3"
	"github.com/gen2brain/malgo"
)

// WakeWordEngine representa el motor de detección continua en segundo plano
type WakeWordEngine struct {
	porcupine *porcupine.Porcupine
	ctx       *malgo.AllocatedContext
	device    *malgo.Device
	onWake    func()
}

// NewWakeWordEngine inicializa el motor nativo de hardware
func NewWakeWordEngine(accessKey string, onWake func()) (*WakeWordEngine, error) {
	if accessKey == "" {
		return nil, fmt.Errorf("se requiere una PICOVOICE_ACCESS_KEY para Porcupine")
	}

	// 1. Inicializar Porcupine (Palabra clave: "porcupine" o usar modelo personalizado)
	p := porcupine.Porcupine{
		AccessKey: accessKey,
		BuiltInKeywords: []porcupine.BuiltInKeyword{porcupine.PORCUPINE}, 
	}
	err := p.Init()
	if err != nil {
		return nil, fmt.Errorf("error inicializando Porcupine: %w", err)
	}

	// 2. Inicializar MiniAudio (Malgo)
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		log.Printf("malgo log: %v\n", message)
	})
	if err != nil {
		p.Delete()
		return nil, fmt.Errorf("error inicializando contexto de audio: %w", err)
	}

	// 3. Configurar el formato exacto que Porcupine necesita
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = uint32(porcupine.SampleRate)
	deviceConfig.Alsa.NoMMap = 1

	engine := &WakeWordEngine{
		porcupine: &p,
		ctx:       ctx,
		onWake:    onWake,
	}

	// Callback que se ejecuta con audio directamente del micrófono
	onRecvFrames := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		sampleCount := framecount * deviceConfig.Capture.Channels
		pcm := make([]int16, sampleCount)
		for i := uint32(0); i < sampleCount; i++ {
			pcm[i] = int16(pInputSamples[i*2]) | (int16(pInputSamples[i*2+1]) << 8)
		}

		if len(pcm) >= porcupine.FrameLength {
			keywordIndex, err := engine.porcupine.Process(pcm[:porcupine.FrameLength])
			if err != nil {
				log.Printf("Error procesando audio: %v", err)
				return
			}
			if keywordIndex >= 0 {
				log.Println("🎯 ¡WAKE WORD DETECTADO NATIVAMENTE!")
				if engine.onWake != nil {
					engine.onWake()
				}
			}
		}
	}

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}

	device, err := malgo.InitDevice(ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		p.Delete()
		ctx.Free()
		return nil, fmt.Errorf("error inicializando dispositivo de audio: %w", err)
	}

	engine.device = device
	return engine, nil
}

// Start comienza la escucha continua en segundo plano
func (e *WakeWordEngine) Start() error {
	return e.device.Start()
}

// Stop detiene la escucha y libera los recursos de hardware
func (e *WakeWordEngine) Stop() {
	if e.device != nil {
		e.device.Uninit()
	}
	if e.ctx != nil {
		e.ctx.Free()
	}
	if e.porcupine != nil {
		e.porcupine.Delete()
	}
}
