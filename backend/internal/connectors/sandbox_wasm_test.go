package connectors

import (
	"context"
	"testing"
)

func TestWasmSandbox_Minimal(t *testing.T) {
	ctx := context.Background()
	sandbox, err := NewWasmSandbox(ctx)
	if err != nil {
		t.Fatalf("Error creando WasmSandbox: %v", err)
	}
	defer sandbox.Close(ctx)

	// Módulo WASM mínimo válido (cabecera standard WebAssembly binary magic + version 1)
	minimalWasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

	res, err := sandbox.ExecuteWasm(ctx, minimalWasm, []string{"test"}, nil, nil)
	if err != nil {
		t.Fatalf("Fallo en ejecución wasm: %v", err)
	}

	if res.ExitCode != 0 {
		t.Errorf("ExitCode inesperado: %d", res.ExitCode)
	}
}
