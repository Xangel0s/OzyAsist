package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

type WatchdogTargetType string

const (
	WatchdogPort    WatchdogTargetType = "port"
	WatchdogProcess WatchdogTargetType = "process"
)

type WatchdogTask struct {
	ID        string             `json:"id"`
	Type      WatchdogTargetType `json:"type"`
	Target    string             `json:"target"` // Ej: '8080', 'docker.exe'
	AlertMsg  string             `json:"alert_msg"`
	Interval  time.Duration      `json:"interval"`
	CreatedAt time.Time          `json:"created_at"`
	cancel    context.CancelFunc
}

type WatchdogRegistry struct {
	mu    sync.RWMutex
	tasks map[string]*WatchdogTask
}

var GlobalWatchdogs = &WatchdogRegistry{
	tasks: make(map[string]*WatchdogTask),
}

func (r *WatchdogRegistry) Start(targetType WatchdogTargetType, target string, interval time.Duration, alertMsg string) *WatchdogTask {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	id := "watch_" + uuid.NewString()[:8]

	if interval < 5*time.Second {
		interval = 10 * time.Second
	}

	task := &WatchdogTask{
		ID:        id,
		Type:      targetType,
		Target:    target,
		AlertMsg:  alertMsg,
		Interval:  interval,
		CreatedAt: time.Now(),
		cancel:    cancel,
	}
	r.tasks[id] = task

	go r.runTask(ctx, task)
	return task
}

func (r *WatchdogRegistry) Stop(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if t, ok := r.tasks[id]; ok {
		t.cancel()
		delete(r.tasks, id)
		return true
	}
	return false
}

func (r *WatchdogRegistry) List() []*WatchdogTask {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []*WatchdogTask
	for _, t := range r.tasks {
		list = append(list, t)
	}
	return list
}

func (r *WatchdogRegistry) runTask(ctx context.Context, task *WatchdogTask) {
	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	lastStateWasDown := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			isDown := false
			switch task.Type {
			case WatchdogPort:
				addr := task.Target
				if !strings.Contains(addr, ":") {
					addr = "localhost:" + addr
				}
				conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
				if err != nil {
					isDown = true
				} else {
					conn.Close()
				}

			case WatchdogProcess:
				nav := system.NewWindowsNavigator()
				procs, err := nav.GetInstalledSoftware(ctx, task.Target)
				found := false
				if err == nil && len(procs) > 0 {
					found = true
				}
				if !found {
					windows, err := nav.GetActiveWindows(ctx)
					if err == nil {
						for _, w := range windows {
							if strings.Contains(strings.ToLower(w.Title), strings.ToLower(task.Target)) {
								found = true
								break
							}
						}
					}
				}
				isDown = !found
			}

			if isDown && !lastStateWasDown {
				lastStateWasDown = true
				title := "⚠️ OzyAssist Watchdog Alert"
				msg := task.AlertMsg
				if msg == "" {
					msg = fmt.Sprintf("El objetivo %s (%s) ya no responde o ha dejado de ejecutarse.", task.Target, task.Type)
				}
				SendToastNotification(title, msg)
			} else if !isDown && lastStateWasDown {
				lastStateWasDown = false
				SendToastNotification("✓ OzyAssist Watchdog", fmt.Sprintf("El objetivo %s (%s) se ha recuperado.", task.Target, task.Type))
			}
		}
	}
}

type WatchdogParams struct {
	Action   string `json:"action"`              // 'start', 'stop', 'list'
	ID       string `json:"id,omitempty"`        // Para 'stop'
	Type     string `json:"type,omitempty"`      // 'port', 'process'
	Target   string `json:"target,omitempty"`    // '8080', 'docker.exe', 'backend.exe'
	Interval int    `json:"interval,omitempty"`  // Segundos entre chequeos (default 10)
	AlertMsg string `json:"alert_msg,omitempty"` // Mensaje para la notificación Toast
}

func execOSWatchdog(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params WatchdogParams
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_watchdog: %v", err), false
	}

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		action = "list"
	}

	switch action {
	case "start":
		if params.Target == "" {
			return "Error: debes especificar 'target' (ej: '8080', 'docker.exe')", false
		}
		targetType := WatchdogPort
		if strings.ToLower(params.Type) == "process" {
			targetType = WatchdogProcess
		}
		interval := time.Duration(params.Interval) * time.Second
		if interval <= 0 {
			interval = 10 * time.Second
		}
		task := GlobalWatchdogs.Start(targetType, params.Target, interval, params.AlertMsg)
		return fmt.Sprintf("🛡️ === WATCHDOG INICIADO EXITOSAMENTE ===\n"+
			"• ID Monitor:   %s\n"+
			"• Tipo:         %s\n"+
			"• Objetivo:     %s\n"+
			"• Intervalo:    %v\n"+
			"OzyAssist vigilará este servicio proactivamente y te notificará por Toast si cae.",
			task.ID, task.Type, task.Target, task.Interval), true

	case "stop":
		if params.ID == "" {
			return "Error: debes proporcionar el 'id' del monitor a detener.", false
		}
		if GlobalWatchdogs.Stop(params.ID) {
			return fmt.Sprintf("✓ Monitor %s detenido correctamente.", params.ID), true
		}
		return fmt.Sprintf("No se encontró ningún monitor activo con el ID %s.", params.ID), false

	case "list":
		tasks := GlobalWatchdogs.List()
		if len(tasks) == 0 {
			return "ℹ️ No hay monitores Watchdog activos en este momento.", true
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🛡️ === MONITORES WATCHDOG ACTIVOS (%d) ===\n", len(tasks)))
		for _, t := range tasks {
			sb.WriteString(fmt.Sprintf("- [%s] %s -> %s (cada %v)\n", t.ID, t.Type, t.Target, t.Interval))
		}
		return sb.String(), true

	default:
		return fmt.Sprintf("Acción desconocida '%s'. Opciones: 'start', 'stop', 'list'.", action), false
	}
}
