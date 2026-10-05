package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ozyassist/backend/internal/providers"
)

func TestRustCore_BinaryDiscoveryAndCameraList(t *testing.T) {
	bin, err := FindRustCoreBinary()
	if err != nil {
		t.Fatalf("no se encontró binario ozy-core.exe: %v", err)
	}

	if !strings.HasSuffix(strings.ToLower(bin), "ozy-core.exe") {
		t.Errorf("ruta de binario inesperada: %s", bin)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listOut, err := CallRustCameraList(ctx)
	if err != nil {
		t.Fatalf("fallo consultando lista de cámaras en ozy-core: %v", err)
	}

	if !strings.Contains(listOut, "Cámaras") && !strings.Contains(listOut, "No se detectaron cámaras") {
		t.Errorf("salida de cámara inesperada: %s", listOut)
	}

	// Probar tool call execOSCameraList
	tc := providers.ToolCall{
		ID:    "call_cam_list",
		Name:  "os_camera_list",
		Input: []byte(`{}`),
	}
	tcOut, ok := execOSCameraList(ctx, tc)
	if !ok {
		t.Errorf("execOSCameraList retornó ok=false: %s", tcOut)
	}
}
