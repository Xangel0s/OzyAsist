package mcp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Transport defines the interface for communicating with an MCP server
type Transport interface {
	Start(ctx context.Context) error
	Send(message []byte) error
	Receive() ([]byte, error)
	Close() error
}

// StdioTransport implements Transport using standard input/output over a spawned process
type StdioTransport struct {
	Command string
	Args    []string
	Env     []string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader
}

func NewStdioTransport(command string, args []string, env []string) *StdioTransport {
	return &StdioTransport{
		Command: command,
		Args:    args,
		Env:     env,
	}
}

func (t *StdioTransport) Start(ctx context.Context) error {
	t.cmd = exec.CommandContext(ctx, t.Command, t.Args...)
	
	if len(t.Env) > 0 {
		t.cmd.Env = append(t.cmd.Environ(), t.Env...)
	}

	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	t.stdout = stdout

	// Start the process
	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server process: %w", err)
	}

	t.reader = bufio.NewReader(t.stdout)
	return nil
}

func (t *StdioTransport) Send(message []byte) error {
	if t.stdin == nil {
		return fmt.Errorf("transport not started")
	}

	// JSON-RPC messages over stdio must be newline-terminated
	msg := string(message)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}

	_, err := t.stdin.Write([]byte(msg))
	return err
}

func (t *StdioTransport) Receive() ([]byte, error) {
	if t.reader == nil {
		return nil, fmt.Errorf("transport not started")
	}

	// Read until newline
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	return line, nil
}

func (t *StdioTransport) Close() error {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		pid := t.cmd.Process.Pid
		if runtime.GOOS == "windows" && pid > 0 {
			_ = exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
		} else {
			_ = t.cmd.Process.Kill()
		}
		_ = t.cmd.Wait()
	}
	return nil
}
