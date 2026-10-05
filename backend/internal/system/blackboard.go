package system

import (
	"context"
	"fmt"
	"hash/fnv"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// OSSnapshot representa una fotografía instantánea del estado de la PC en memoria RAM.
type OSSnapshot struct {
	ActiveWindowTitle string
	ActiveWindowHWND  uintptr
	OpenWindows       []string
	FocusedFile       string
	ClipboardSnippet  string
	CPUPercent        int
	RAMUsedMB         uint64
	RAMTotalMB        uint64
	Hash              uint64
	Timestamp         time.Time
}

// OSBlackboard es un pizarrón en memoria RAM física de alta velocidad (< 0.01 ms)
// que mantiene el estado consolidado del sistema operativo pre-renderizado en caliente.
type OSBlackboard struct {
	mu       sync.RWMutex
	snapshot OSSnapshot
	nav      OSNavigator
	stopCh   chan struct{}
}

var (
	defaultBlackboard     *OSBlackboard
	defaultBlackboardOnce sync.Once
)

// DefaultBlackboard retorna la instancia singleton del pizarrón de estado en RAM.
func DefaultBlackboard() *OSBlackboard {
	defaultBlackboardOnce.Do(func() {
		b := &OSBlackboard{
			nav:    NewWindowsNavigator(),
			stopCh: make(chan struct{}),
		}
		b.Refresh(context.Background())
		b.startAutoRefresh()
		defaultBlackboard = b
	})
	return defaultBlackboard
}

func (b *OSBlackboard) startAutoRefresh() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
				b.Refresh(ctx)
				cancel()
			case <-b.stopCh:
				return
			}
		}
	}()
}

// Close detiene el worker de refresco en segundo plano.
func (b *OSBlackboard) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	select {
	case <-b.stopCh:
	default:
		close(b.stopCh)
	}
}

// Refresh actualiza la fotografía del sistema en RAM.
func (b *OSBlackboard) Refresh(ctx context.Context) {
	snap := OSSnapshot{
		Timestamp: time.Now(),
	}

	// 1. Ventanas activas y ventana en foco
	if wins, err := b.nav.GetActiveWindows(ctx); err == nil && len(wins) > 0 {
		var titles []string
		for i, w := range wins {
			t := strings.TrimSpace(w.Title)
			if t != "" && !strings.EqualFold(t, "Program Manager") && !strings.EqualFold(t, "Windows Input Experience") {
				if snap.ActiveWindowTitle == "" && i == 0 {
					snap.ActiveWindowTitle = t
					snap.ActiveWindowHWND = w.Handle
				}
				titles = append(titles, t)
				if len(titles) >= 6 {
					break
				}
			}
		}
		snap.OpenWindows = titles
	}

	// 2. Telemetría ligera de hardware (CPU / RAM)
	if vMem, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		snap.RAMTotalMB = vMem.Total / 1024 / 1024
		snap.RAMUsedMB = vMem.Used / 1024 / 1024
	}
	if cpuPerc, err := cpu.PercentWithContext(ctx, 50*time.Millisecond, false); err == nil && len(cpuPerc) > 0 {
		snap.CPUPercent = int(cpuPerc[0])
	}

	// 3. Portapapeles (resumen corto sanitizado)
	if clip, err := ReadClipboardContent(); err == nil {
		cleanClip := strings.TrimSpace(clip)
		if len(cleanClip) > 80 {
			cleanClip = cleanClip[:80] + "..."
		}
		snap.ClipboardSnippet = cleanClip
	}

	// 4. Calcular hash de estado para detección de cambios (Delta Check)
	h := fnv.New64a()
	h.Write([]byte(snap.ActiveWindowTitle))
	for _, w := range snap.OpenWindows {
		h.Write([]byte(w))
	}
	h.Write([]byte(snap.ClipboardSnippet))
	snap.Hash = h.Sum64()

	b.mu.Lock()
	snap.FocusedFile = b.snapshot.FocusedFile // Preservar archivo en foco
	b.snapshot = snap
	b.mu.Unlock()
}

// SetFocusedFile registra el archivo actualmente en foco de la conversación.
func (b *OSBlackboard) SetFocusedFile(file string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.snapshot.FocusedFile = file
}

// GetSnapshot retorna una copia del snapshot actual en RAM (< 0.001 ms).
func (b *OSBlackboard) GetSnapshot() OSSnapshot {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.snapshot
}

// RenderHUD genera una ficha visual compacta (< 65 tokens) optimizada para la atención del LLM.
func (b *OSBlackboard) RenderHUD() string {
	b.mu.RLock()
	snap := b.snapshot
	b.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("\n\n┌─[ OZY OS-HUD: ESTADO EN VIVO ]───────────────────────────────────┐\n")

	if snap.ActiveWindowTitle != "" {
		sb.WriteString(fmt.Sprintf("│ • FOCO: %q (HWND: 0x%X)\n", snap.ActiveWindowTitle, snap.ActiveWindowHWND))
	} else {
		sb.WriteString("│ • FOCO: Escritorio de Windows\n")
	}

	if len(snap.OpenWindows) > 0 {
		sb.WriteString(fmt.Sprintf("│ • VENTANAS ACTIVAS: [%s]\n", strings.Join(snap.OpenWindows, " | ")))
	}

	if snap.FocusedFile != "" {
		sb.WriteString(fmt.Sprintf("│ • ARCHIVO EN FOCO: %q\n", snap.FocusedFile))
	}

	if snap.ClipboardSnippet != "" {
		sb.WriteString(fmt.Sprintf("│ • PORTAPAPELES: %q\n", snap.ClipboardSnippet))
	}

	sb.WriteString(fmt.Sprintf("│ • HARDWARE: CPU %d%% | RAM %d/%d MB (%s)\n",
		snap.CPUPercent, snap.RAMUsedMB, snap.RAMTotalMB, runtime.GOARCH))
	sb.WriteString("└──────────────────────────────────────────────────────────────────┘")

	return sb.String()
}

// RenderDelta compara con el hash previo: si no hubo cambios, retorna un marcador de 8 tokens.
func (b *OSBlackboard) RenderDelta(prevHash uint64) (string, uint64) {
	b.mu.RLock()
	snap := b.snapshot
	b.mu.RUnlock()

	if prevHash != 0 && snap.Hash == prevHash {
		return "\n[PC-ESTADO: Sin cambios en ventanas ni foco respecto al turno previo]", snap.Hash
	}

	return b.RenderHUD(), snap.Hash
}

// IsOSRelevantQuery determina si la consulta del usuario amerita inyectar contexto del SO.
// Si es conceptual, teórica, de charla o saludo, retorna false (0 tokens de overhead).
func IsOSRelevantQuery(query string) bool {
	lower := strings.ToLower(strings.TrimSpace(query))
	if lower == "" {
		return false
	}

	// 1. Verificación por tokens exactos (evita falsos positivos como 'red' en 'redacta')
	tokens := strings.Fields(lower)
	tokenSet := make(map[string]bool)
	for _, t := range tokens {
		clean := strings.Trim(t, "¿?¡!.,;:\"'()[]{}")
		tokenSet[clean] = true
	}

	osWords := map[string]bool{
		"abre": true, "abrir": true, "cierra": true, "cerrar": true,
		"ventana": true, "ventanas": true, "app": true, "apps": true,
		"programa": true, "programas": true,
		"pantalla": true, "foco": true, "minimiza": true, "maximiza": true,
		"brillo": true, "volumen": true, "audio": true, "sonido": true,
		"mute": true, "archivo": true, "archivos": true, "carpeta": true,
		"carpetas": true, "directorio": true, "disco": true, "ram": true,
		"cpu": true, "bateria": true, "servicio": true, "servicios": true,
		"proceso": true, "procesos": true, "kill": true, "hardware": true,
		"wifi": true, "red": true, "puerto": true, "puertos": true,
		"notifica": true, "toast": true, "calculadora": true, "notepad": true,
		"bloc": true, "chrome": true, "spotify": true, "vscode": true,
		"antigravity": true, "terminal": true, "powershell": true, "git": true,
		"escritorio": true, "dock": true, "taskmgr": true, "pc": true,
		"abierto": true, "abiertos": true, "abierta": true, "abiertas": true,
	}

	for w := range tokenSet {
		if osWords[w] {
			return true
		}
	}

	// 2. Referencias deícticas (señalar algo del entorno o frases compuestas)
	deicticPhrases := []string{
		"ábrelo", "ábrela", "ciérralo", "ciérrala", "léelo", "léela",
		"qué hay", "qué se ve", "el error", "en pantalla", "portapapeles",
		"que tengo abierto", "que esta abierto", "que ventana", "copiado",
		"esta ventana", "este programa",
	}
	for _, phrase := range deicticPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}

	return false
}
