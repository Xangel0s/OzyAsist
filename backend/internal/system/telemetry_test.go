package system

import (
	"context"
	"testing"
	"time"
)

func TestCollector_CollectMetrics(t *testing.T) {
	collector := NewCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	metrics, err := collector.Collect(ctx)
	if err != nil {
		t.Fatalf("Error recolectando telemetría del sistema: %v", err)
	}

	if metrics.CPUCores <= 0 {
		t.Errorf("CPUCores inválido: %d", metrics.CPUCores)
	}

	if metrics.RAMTotalMB == 0 {
		t.Errorf("RAMTotalMB reporta 0 MB")
	}

	throttling, reason := collector.ShouldThrottleBackground(metrics)
	t.Logf("Telemetría obtenida: CPU=%.1f%%, RAM Usada=%d/%d MB, Throttling=%v (%s)",
		metrics.CPUUsagePercent, metrics.RAMUsedMB, metrics.RAMTotalMB, throttling, reason)
}
