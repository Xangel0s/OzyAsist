package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ExecTerminalReq struct {
	Command string `json:"command"`
	Timeout int    `json:"timeoutSecs"` // optional timeout in seconds
}

type ExecTerminalRes struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Duration string `json:"duration"`
	Error    string `json:"error,omitempty"`
}

// ExecuteTerminalCommand runs a shell command inside the project's root directory.
func ExecuteTerminalCommand(c *gin.Context) {
	projectID := c.Param("id")
	rootDir, err := getProjectRoot(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req ExecTerminalReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Command) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo command es requerido"})
		return
	}

	timeoutSecs := req.Timeout
	if timeoutSecs <= 0 || timeoutSecs > 120 {
		timeoutSecs = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", req.Command)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-c", req.Command)
	}

	cmd.Dir = rootDir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	startTime := time.Now()
	runErr := cmd.Run()
	duration := time.Since(startTime)

	exitCode := 0
	if runErr != nil {
		if exitError, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	errStr := ""
	if runErr != nil && exitCode != 0 {
		errStr = runErr.Error()
	}

	c.JSON(http.StatusOK, ExecTerminalRes{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: exitCode,
		Duration: fmt.Sprintf("%.2fs", duration.Seconds()),
		Error:    errStr,
	})
}
