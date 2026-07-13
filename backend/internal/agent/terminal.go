package agent

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const defaultCmdTimeout = 30 * time.Second

type CmdResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Duration string `json:"duration"`
}

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

func ExecuteShell(ctx context.Context, shell, script string) (*CmdResult, error) {
	return ExecuteShellInDir(ctx, shell, script, "")
}

func ExecuteShellInDir(ctx context.Context, shell, script, dir string) (*CmdResult, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultCmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, "-c", script)
	if dir != "" {
		cmd.Dir = dir
	}
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
