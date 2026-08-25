package agent

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	defaultCmdTimeout  = 30 * time.Second
	streamingCmdTimeout = 120 * time.Second // comandos largos (build, install, etc.)
)

// CmdResult es el resultado de ejecutar un comando.
type CmdResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Duration string `json:"duration"`
}

// ExecuteCommand ejecuta un comando con argumentos discretos (sin shell).
func ExecuteCommand(ctx context.Context, command string, args ...string) (*CmdResult, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultCmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start).String()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("command execution error: %w", err)
		}
	}

	return &CmdResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: duration,
	}, nil
}

// ExecuteShell ejecuta un script en un shell (sin directorio específico).
func ExecuteShell(ctx context.Context, shell, script string) (*CmdResult, error) {
	return ExecuteShellInDir(ctx, shell, script, "")
}

// ExecuteShellInDir ejecuta un script shell en un directorio dado.
// En Windows usa powershell.exe. Captura stdout y stderr en buffers (no streaming).
// Usa timeout de 30s.
func ExecuteShellInDir(ctx context.Context, shell, script, dir string) (*CmdResult, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultCmdTimeout)
	defer cancel()

	cmd := buildShellCmd(ctx, shell, script, dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start).String()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() != nil {
			// Asegurar que el proceso y su árbol mueren al cancelar el contexto
			killProcessTree(cmd)
			return nil, fmt.Errorf("comando cancelado: %w", ctx.Err())
		} else {
			return nil, fmt.Errorf("command execution error: %w", err)
		}
	}

	return &CmdResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: duration,
	}, nil
}

// ExecuteStreamingCommand ejecuta un comando con streaming de líneas en tiempo real.
// Usa un timeout de 120s. Útil para comandos largos (build, npm install, go test, etc.).
// En Windows mata el proceso y su árbol al cancelar el contexto.
func ExecuteStreamingCommand(ctx context.Context, script, dir string) (*CmdResult, error) {
	ctx, cancel := context.WithTimeout(ctx, streamingCmdTimeout)
	defer cancel()

	cmd := buildShellCmd(ctx, "powershell", script, dir)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	var (
		stdoutLines []string
		stderrLines []string
		mu          sync.Mutex
		wg          sync.WaitGroup
	)

	// Goroutine para leer stdout línea a línea
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			stdoutLines = append(stdoutLines, line)
			mu.Unlock()
		}
	}()

	// Goroutine para leer stderr línea a línea
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			stderrLines = append(stderrLines, line)
			mu.Unlock()
		}
	}()

	start := time.Now()

	// Esperar a que el proceso termine o el contexto se cancele
	done := make(chan error, 1)
	go func() {
		wg.Wait()
		done <- cmd.Wait()
	}()

	var cmdErr error
	select {
	case cmdErr = <-done:
		// Proceso terminó normalmente
	case <-ctx.Done():
		// Contexto cancelado — matar árbol de procesos en Windows
		killProcessTree(cmd)
		<-done // esperar que Wait retorne
		return nil, fmt.Errorf("comando cancelado: %w", ctx.Err())
	}

	duration := time.Since(start).String()

	exitCode := 0
	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return &CmdResult{
		Stdout:   strings.Join(stdoutLines, "\n"),
		Stderr:   strings.Join(stderrLines, "\n"),
		ExitCode: exitCode,
		Duration: duration,
	}, nil
}

// buildShellCmd construye el comando para el shell correcto según la plataforma.
// En Windows usa powershell.exe con -NonInteractive -Command.
func buildShellCmd(ctx context.Context, shell, script, dir string) *exec.Cmd {
	var cmd *exec.Cmd
	switch shell {
	case "powershell", "pwsh":
		// -NonInteractive: no pedir input interactivo
		// -NoProfile: arranque más rápido
		// -Command: ejecutar el script directamente
		cmd = exec.CommandContext(ctx, "powershell.exe", "-NonInteractive", "-NoProfile", "-Command", script)
	default:
		cmd = exec.CommandContext(ctx, shell, "-c", script)
	}
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd
}

// killProcessTree mata el proceso y todos sus hijos en Windows.
// Usa taskkill /F /T /PID para matar el árbol completo de procesos.
// Esto evita procesos huérfanos cuando se cancela el contexto.
func killProcessTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	// /F = forzar, /T = matar árbol completo de subprocesos
	killer := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	killer.Run() // ignorar error — puede que el proceso ya terminó
}
