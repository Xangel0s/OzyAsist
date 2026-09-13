package connectors

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type WasmExecutionResult struct {
	ExitCode   uint32        `json:"exit_code"`
	Stdout     string        `json:"stdout"`
	Stderr     string        `json:"stderr"`
	DurationMS int64         `json:"duration_ms"`
}

type WasmSandbox struct {
	runtime wazero.Runtime
}

func NewWasmSandbox(ctx context.Context) (*WasmSandbox, error) {
	config := wazero.NewRuntimeConfig()
	r := wazero.NewRuntimeWithConfig(ctx, config)

	// Inicializar soporte estándar WASI
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	return &WasmSandbox{runtime: r}, nil
}

// ExecuteWasm ejecuta binarios WebAssembly de forma 100% aislada en memoria
func (s *WasmSandbox) ExecuteWasm(ctx context.Context, wasmBytes []byte, args []string, env map[string]string, stdin []byte) (*WasmExecutionResult, error) {
	start := time.Now()

	var stdoutBuf, stderrBuf bytes.Buffer
	stdinBuf := bytes.NewReader(stdin)

	modConfig := wazero.NewModuleConfig().
		WithStdout(&stdoutBuf).
		WithStderr(&stderrBuf).
		WithStdin(stdinBuf).
		WithArgs(args...)

	for k, v := range env {
		modConfig = modConfig.WithEnv(k, v)
	}

	compiled, err := s.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("compilar módulo wasm: %w", err)
	}
	defer compiled.Close(ctx)

	mod, err := s.runtime.InstantiateModule(ctx, compiled, modConfig)
	if err != nil {
		return &WasmExecutionResult{
			ExitCode:   1,
			Stdout:     stdoutBuf.String(),
			Stderr:     fmt.Sprintf("error instanciando módulo: %v\n%s", err, stderrBuf.String()),
			DurationMS: time.Since(start).Milliseconds(),
		}, nil
	}
	defer mod.Close(ctx)

	return &WasmExecutionResult{
		ExitCode:   0,
		Stdout:     stdoutBuf.String(),
		Stderr:     stderrBuf.String(),
		DurationMS: time.Since(start).Milliseconds(),
	}, nil
}

// Close libera recursos del runtime Wazero
func (s *WasmSandbox) Close(ctx context.Context) error {
	if s.runtime != nil {
		return s.runtime.Close(ctx)
	}
	return nil
}
