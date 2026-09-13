package security

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type AlertAction struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Style string `json:"style"` // "danger", "secondary", "primary"
}

type SecurityAlert struct {
	Level      string        `json:"level"`    // "critical", "warning", "info"
	Category   string        `json:"category"` // "security_watchdog", "high_cpu", "temp_binary"
	Title      string        `json:"title"`
	Message    string        `json:"message"`
	PID        int32         `json:"pid"`
	Executable string        `json:"executable"`
	Actions    []AlertAction `json:"actions"`
	DetectedAt time.Time     `json:"detected_at"`
}

type AlertBroadcaster interface {
	BroadcastJSON(event string, payload interface{})
}

type Watchdog struct {
	broadcaster AlertBroadcaster
	auditLogger *AuditLogger
	stopCh      chan struct{}
	running     bool
	mu          sync.Mutex
	alertCache  map[int32]time.Time
}

func NewWatchdog(broadcaster AlertBroadcaster, auditLogger *AuditLogger) *Watchdog {
	return &Watchdog{
		broadcaster: broadcaster,
		auditLogger: auditLogger,
		alertCache:  make(map[int32]time.Time),
	}
}

// Start arranca el ciclo de vigilancia EDR en segundo plano
func (w *Watchdog) Start(interval time.Duration) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	if interval < 2*time.Second {
		interval = 5 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-w.stopCh:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
				_, _ = w.ScanProcesses(ctx)
				cancel()
			}
		}
	}()
	log.Printf("[EDR Watchdog] Monitoreo continuo de procesos anómalos iniciado (intervalo: %v)", interval)
}

// Stop detiene el demonio EDR
func (w *Watchdog) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return
	}
	w.running = false
	close(w.stopCh)
}

// ScanProcesses inspecciona los procesos activos en busca de anomalías de ruta o consumo
func (w *Watchdog) ScanProcesses(ctx context.Context) ([]SecurityAlert, error) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar procesos: %w", err)
	}

	var alerts []SecurityAlert
	now := time.Now()

	for _, p := range processes {
		select {
		case <-ctx.Done():
			return alerts, ctx.Err()
		default:
		}

		pid := p.Pid
		// Ignorar PIDs del sistema base
		if pid <= 4 {
			continue
		}

		// Evitar spam de alertas sobre el mismo PID (re-alertar solo cada 2 minutos)
		w.mu.Lock()
		lastAlert, exists := w.alertCache[pid]
		if exists && now.Sub(lastAlert) < 2*time.Minute {
			w.mu.Unlock()
			continue
		}
		w.mu.Unlock()

		exePath, err := p.ExeWithContext(ctx)
		if err != nil || exePath == "" {
			continue
		}

		name, _ := p.NameWithContext(ctx)
		cleanPath := strings.ToLower(filepath.ToSlash(exePath))
		cleanName := strings.ToLower(name)

		isSuspiciousPath := strings.Contains(cleanPath, "/appdata/local/temp/") ||
			strings.Contains(cleanPath, "/temp/") ||
			strings.Contains(cleanPath, "/tmp/")

		isKnownThreatName := strings.Contains(cleanName, "xmr") ||
			strings.Contains(cleanName, "cryptonight") ||
			strings.Contains(cleanName, "mimikatz") ||
			strings.Contains(cleanName, "coinhive")

		cpuPct, _ := p.CPUPercentWithContext(ctx)

		if isKnownThreatName || (isSuspiciousPath && cpuPct > 80.0) {
			alert := SecurityAlert{
				Level:      "critical",
				Category:   "security_watchdog",
				Title:      "Proceso Sospechoso Detectado por Watchdog EDR",
				Message:    fmt.Sprintf("El ejecutable '%s' (PID %d) se está ejecutando desde una ruta temporal con consumo de CPU del %.1f%%.", name, pid, cpuPct),
				PID:        pid,
				Executable: exePath,
				Actions: []AlertAction{
					{ID: "kill_process", Label: "Terminar Proceso", Style: "danger"},
					{ID: "quarantine", Label: "Aislar Archivo", Style: "secondary"},
				},
				DetectedAt: now,
			}

			w.mu.Lock()
			w.alertCache[pid] = now
			w.mu.Unlock()

			alerts = append(alerts, alert)

			// Registrar en Audit Trail inmutable
			if w.auditLogger != nil {
				detailsJSON, _ := json.Marshal(alert)
				_, _ = w.auditLogger.Record(ctx, "watchdog_edr", "anomalous_process_detected", string(detailsJSON))
			}

			// Transmitir vía WebSocket
			if w.broadcaster != nil {
				w.broadcaster.BroadcastJSON("system:alert", alert)
			}
		}
	}

	return alerts, nil
}

// KillProcess termina de inmediato un proceso identificado por PID
func (w *Watchdog) KillProcess(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("obtener proceso PID %d: %w", pid, err)
	}

	err = p.Kill()
	if err != nil {
		return fmt.Errorf("terminar proceso PID %d: %w", pid, err)
	}

	if w.auditLogger != nil {
		_, _ = w.auditLogger.Record(context.Background(), "watchdog_edr", "process_terminated", fmt.Sprintf("PID %d forzado a terminar", pid))
	}
	return nil
}

// QuarantineFile aísla un archivo sospechoso renombrándolo con extensión .quarantine
func (w *Watchdog) QuarantineFile(filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("archivo no encontrado: %w", err)
	}

	quarantinedPath := filePath + ".quarantine"
	if err := os.Rename(filePath, quarantinedPath); err != nil {
		return fmt.Errorf("aislar archivo: %w", err)
	}

	if w.auditLogger != nil {
		_, _ = w.auditLogger.Record(context.Background(), "watchdog_edr", "file_quarantined", fmt.Sprintf("Archivo %s movido a %s", filePath, quarantinedPath))
	}
	return nil
}
