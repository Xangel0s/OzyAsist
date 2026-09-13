package system

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemMetrics struct {
	CPUUsagePercent   float64   `json:"cpu_usage_percent"`
	CPUCores          int       `json:"cpu_cores"`
	RAMTotalMB        uint64    `json:"ram_total_mb"`
	RAMUsedMB         uint64    `json:"ram_used_mb"`
	RAMFreeMB         uint64    `json:"ram_free_mb"`
	RAMUsagePercent   float64   `json:"ram_usage_percent"`
	DiskTotalGB       uint64    `json:"disk_total_gb"`
	DiskFreeGB        uint64    `json:"disk_free_gb"`
	DiskUsagePercent  float64   `json:"disk_usage_percent"`
	UptimeSeconds     uint64    `json:"uptime_seconds"`
	Platform          string    `json:"platform"`
	ThermalThrottling bool      `json:"thermal_throttling"`
	CollectedAt       time.Time `json:"collected_at"`
}

type Collector struct{}

func NewCollector() *Collector {
	return &Collector{}
}

// Collect toma una captura instantánea de los recursos del sistema operativo
func (c *Collector) Collect(ctx context.Context) (SystemMetrics, error) {
	metrics := SystemMetrics{
		CPUCores:    runtime.NumCPU(),
		CollectedAt: time.Now(),
	}

	// 1. Uso de CPU
	cpuPercentages, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, false)
	if err == nil && len(cpuPercentages) > 0 {
		metrics.CPUUsagePercent = cpuPercentages[0]
	}

	// 2. Memoria RAM
	vMem, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		metrics.RAMTotalMB = vMem.Total / 1024 / 1024
		metrics.RAMUsedMB = vMem.Used / 1024 / 1024
		metrics.RAMFreeMB = vMem.Available / 1024 / 1024
		metrics.RAMUsagePercent = vMem.UsedPercent
	}

	// 3. Almacenamiento raíz / principal
	rootPath := "/"
	if runtime.GOOS == "windows" {
		rootPath = "C:\\"
	}
	dUsage, err := disk.UsageWithContext(ctx, rootPath)
	if err == nil {
		metrics.DiskTotalGB = dUsage.Total / 1024 / 1024 / 1024
		metrics.DiskFreeGB = dUsage.Free / 1024 / 1024 / 1024
		metrics.DiskUsagePercent = dUsage.UsedPercent
	}

	// 4. Host Uptime y Plataforma
	hInfo, err := host.InfoWithContext(ctx)
	if err == nil {
		metrics.UptimeSeconds = hInfo.Uptime
		metrics.Platform = fmt.Sprintf("%s %s", hInfo.Platform, hInfo.PlatformVersion)
	}

	// 5. Heurística de Throttling térmico / sobrecarga
	if metrics.CPUUsagePercent > 90.0 && metrics.RAMUsagePercent > 90.0 {
		metrics.ThermalThrottling = true
	}

	return metrics, nil
}

// ShouldThrottleBackground indica si el sistema debe pausar tareas no esenciales
func (c *Collector) ShouldThrottleBackground(m SystemMetrics) (bool, string) {
	if m.RAMFreeMB < 1500 {
		return true, fmt.Sprintf("Memoria RAM disponible crítica: %d MB", m.RAMFreeMB)
	}
	if m.CPUUsagePercent > 88.0 {
		return true, fmt.Sprintf("Uso de CPU elevado: %.1f%%", m.CPUUsagePercent)
	}
	if m.DiskUsagePercent > 95.0 {
		return true, fmt.Sprintf("Espacio en disco casi agotado: %.1f%%", m.DiskUsagePercent)
	}
	return false, ""
}
