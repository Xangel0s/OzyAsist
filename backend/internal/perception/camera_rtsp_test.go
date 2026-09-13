package perception

import (
	"context"
	"testing"
	"time"
)

func TestRTSPClient_ConfigValidation(t *testing.T) {
	config := RTSPStreamConfig{
		URL:     "",
		Timeout: 2 * time.Second,
	}
	client := NewRTSPClient(config)
	if client == nil {
		t.Fatalf("Error creando RTSPClient")
	}

	_, err := client.CaptureFrame(context.Background())
	if err == nil {
		t.Errorf("Esperaba error por URL vacía, pero no ocurrió")
	}
}
