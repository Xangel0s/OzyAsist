package audio

import "log"

// WakeWordEngine representa el motor de detección continua en segundo plano (Stubbed)
type WakeWordEngine struct {
	onWake func()
}

// NewWakeWordEngine inicializa el motor en modo stub
func NewWakeWordEngine(accessKey string, onWake func()) (*WakeWordEngine, error) {
	log.Println("WakeWordEngine (Stub) inicializado. Detección de Wake word deshabilitada por CLI.")
	return &WakeWordEngine{
		onWake: onWake,
	}, nil
}

// Start simula el comienzo de la escucha continua
func (e *WakeWordEngine) Start() error {
	log.Println("WakeWordEngine (Stub) iniciado.")
	return nil
}

// Stop simula la detención de la escucha
func (e *WakeWordEngine) Stop() {
	log.Println("WakeWordEngine (Stub) detenido.")
}
