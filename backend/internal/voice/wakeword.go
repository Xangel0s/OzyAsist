package voice

import (
	"context"
	"log"
	"os/exec"
	"sync"
	"time"
)

// WakeWordDetector gestiona la detección en segundo plano de "Hey Ozy" o palabras clave
type WakeWordDetector struct {
	mu        sync.Mutex
	isRunning bool
	cancel    context.CancelFunc
	onDetect  func()
}

// NewWakeWordDetector inicializa un detector de palabra clave
func NewWakeWordDetector(onDetect func()) *WakeWordDetector {
	return &WakeWordDetector{
		onDetect: onDetect,
	}
}

// Start inicia la escucha pasiva del Wake Word con bajo consumo de CPU
func (d *WakeWordDetector) Start(binaryPath string, keywords ...string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isRunning {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.isRunning = true

	go func() {
		defer func() {
			d.mu.Lock()
			d.isRunning = false
			d.mu.Unlock()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if binaryPath != "" {
					cmd := exec.CommandContext(ctx, binaryPath)
					if err := cmd.Run(); err != nil && ctx.Err() == nil {
						log.Printf("[WakeWord] Proceso reiniciando: %v", err)
						time.Sleep(1 * time.Second)
					} else if ctx.Err() == nil && d.onDetect != nil {
						d.onDetect()
					}
				} else {
					// Standby mode si el binario local no está configurado (delegado al cliente web)
					time.Sleep(2 * time.Second)
				}
			}
		}
	}()

	return nil
}

// Stop detiene la escucha pasiva
func (d *WakeWordDetector) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	d.isRunning = false
}

// IsRunning indica si el detector está activo
func (d *WakeWordDetector) IsRunning() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.isRunning
}
