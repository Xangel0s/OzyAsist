package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/providers"
)

type GitFileItem struct {
	Path    string `json:"path"`
	Status  string `json:"status"` // "M", "A", "D", "U", "R"
	Added   int    `json:"added"`
	Deleted int    `json:"deleted"`
	Staged  bool   `json:"staged"`
}

type GitStatusResponse struct {
	Initialized bool          `json:"initialized"`
	Branch      string        `json:"branch"`
	Ahead       int           `json:"ahead"`
	Behind      int           `json:"behind"`
	Files       []GitFileItem `json:"files"`
	TotalAdded  int           `json:"totalAdded"`
	TotalDel    int           `json:"totalDeleted"`
	Error       string        `json:"error,omitempty"`
}

func getProjectRoot(projectID string) (string, error) {
	project, err := db.GetProject(projectID)
	if err != nil {
		return "", fmt.Errorf("proyecto no encontrado")
	}
	if project.RootPath == "" {
		return "", fmt.Errorf("el proyecto no tiene root_path definido")
	}
	if _, err := os.Stat(project.RootPath); err != nil {
		return "", fmt.Errorf("root_path no accesible: %w", err)
	}
	return project.RootPath, nil
}

func runGitCmd(dir string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// GetGitStatus checks if .git exists and returns branch, staged/unstaged changes, and diff stats.
func GetGitStatus(c *gin.Context) {
	projectID := c.Param("id")
	rootDir, err := getProjectRoot(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gitDir := filepath.Join(rootDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, GitStatusResponse{
			Initialized: false,
			Branch:      "",
			Files:       []GitFileItem{},
		})
		return
	}

	// 1. Get branch info
	branchOut, _, err := runGitCmd(rootDir, "branch", "--show-current")
	branch := branchOut
	if branch == "" {
		branch = "main"
	}

	// 2. Get status --porcelain=v1 -b
	statusOut, _, _ := runGitCmd(rootDir, "status", "--porcelain=v1", "-b", "-u")

	var files []GitFileItem
	ahead, behind := 0, 0

	// 3. Get numstat for staged diff
	stagedStats := make(map[string][2]int)
	if numstatStaged, _, err := runGitCmd(rootDir, "diff", "--cached", "--numstat"); err == nil && numstatStaged != "" {
		for _, line := range strings.Split(numstatStaged, "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				add, _ := strconv.Atoi(parts[0])
				del, _ := strconv.Atoi(parts[1])
				stagedStats[filepath.ToSlash(parts[2])] = [2]int{add, del}
			}
		}
	}

	// 4. Get numstat for unstaged diff
	unstagedStats := make(map[string][2]int)
	if numstatUnstaged, _, err := runGitCmd(rootDir, "diff", "--numstat"); err == nil && numstatUnstaged != "" {
		for _, line := range strings.Split(numstatUnstaged, "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				add, _ := strconv.Atoi(parts[0])
				del, _ := strconv.Atoi(parts[1])
				unstagedStats[filepath.ToSlash(parts[2])] = [2]int{add, del}
			}
		}
	}

	totalAdded, totalDel := 0, 0

	lines := strings.Split(statusOut, "\n")
	for _, rawLine := range lines {
		if strings.TrimSpace(rawLine) == "" {
			continue
		}
		if strings.HasPrefix(rawLine, "##") {
			if strings.Contains(rawLine, "ahead ") {
				fmt.Sscanf(rawLine[strings.Index(rawLine, "ahead ")+6:], "%d", &ahead)
			}
			if strings.Contains(rawLine, "behind ") {
				fmt.Sscanf(rawLine[strings.Index(rawLine, "behind ")+7:], "%d", &behind)
			}
			continue
		}

		if len(rawLine) < 3 {
			continue
		}

		indexStatus := string(rawLine[0])
		workTreeStatus := string(rawLine[1])
		filePath := strings.TrimSpace(rawLine[3:])
		filePath = filepath.ToSlash(filePath)

		// Handle renamed files: "R  old -> new"
		if strings.Contains(filePath, " -> ") {
			parts := strings.Split(filePath, " -> ")
			if len(parts) == 2 {
				filePath = parts[1]
			}
		}

		// If staged in index
		if indexStatus != " " && indexStatus != "?" {
			stats := stagedStats[filePath]
			files = append(files, GitFileItem{
				Path:    filePath,
				Status:  indexStatus,
				Added:   stats[0],
				Deleted: stats[1],
				Staged:  true,
			})
			totalAdded += stats[0]
			totalDel += stats[1]
		}

		// If modified in work tree
		if workTreeStatus != " " {
			stats := unstagedStats[filePath]
			statusLetter := workTreeStatus
			if workTreeStatus == "?" {
				statusLetter = "U"
			}
			files = append(files, GitFileItem{
				Path:    filePath,
				Status:  statusLetter,
				Added:   stats[0],
				Deleted: stats[1],
				Staged:  false,
			})
			totalAdded += stats[0]
			totalDel += stats[1]
		}
	}

	c.JSON(http.StatusOK, GitStatusResponse{
		Initialized: true,
		Branch:      branch,
		Ahead:       ahead,
		Behind:      behind,
		Files:       files,
		TotalAdded:  totalAdded,
		TotalDel:    totalDel,
	})
}

// InitGit initializes a real git repository in project root.
func InitGit(c *gin.Context) {
	projectID := c.Param("id")
	rootDir, err := getProjectRoot(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stdout, stderr, err := runGitCmd(rootDir, "init", "-b", "main")
	if err != nil {
		// Fallback for older git versions without -b flag
		stdout, stderr, err = runGitCmd(rootDir, "init")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error initializing git: %s (stderr: %s)", err.Error(), stderr)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Repositorio Git inicializado correctamente",
		"output":  stdout,
	})
}

type StageReq struct {
	Paths []string `json:"paths"`
	All   bool     `json:"all"`
}

// StageGit stages files in git.
func StageGit(c *gin.Context) {
	projectID := c.Param("id")
	rootDir, err := getProjectRoot(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req StageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
		return
	}

	args := []string{"add"}
	if req.All || len(req.Paths) == 0 {
		args = append(args, ".")
	} else {
		args = append(args, req.Paths...)
	}

	stdout, stderr, err := runGitCmd(rootDir, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al hacer stage: %s (stderr: %s)", err.Error(), stderr)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Archivos preparados (staged)", "output": stdout})
}

// UnstageGit unstages files from index.
func UnstageGit(c *gin.Context) {
	projectID := c.Param("id")
	rootDir, err := getProjectRoot(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req StageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
		return
	}

	args := []string{"restore", "--staged"}
	if req.All || len(req.Paths) == 0 {
		args = append(args, ".")
	} else {
		args = append(args, req.Paths...)
	}

	stdout, stderr, err := runGitCmd(rootDir, args...)
	if err != nil {
		// Fallback to git reset HEAD
		resetArgs := []string{"reset", "HEAD"}
		if !req.All && len(req.Paths) > 0 {
			resetArgs = append(resetArgs, req.Paths...)
		}
		stdout, stderr, err = runGitCmd(rootDir, resetArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al quitar stage: %s (stderr: %s)", err.Error(), stderr)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Archivos retirados de stage", "output": stdout})
}

type CommitReq struct {
	Message string `json:"message"`
	Sync    bool   `json:"sync"`
}

// CommitGit makes a git commit with the given message.
func CommitGit(c *gin.Context) {
	projectID := c.Param("id")
	rootDir, err := getProjectRoot(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req CommitReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mensaje de commit requerido"})
		return
	}

	stdout, stderr, err := runGitCmd(rootDir, "commit", "-m", req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al hacer commit: %s (stderr: %s)", err.Error(), stderr)})
		return
	}

	syncOutput := ""
	if req.Sync {
		pullOut, _, _ := runGitCmd(rootDir, "pull", "--rebase")
		pushOut, pushErrOut, pushErr := runGitCmd(rootDir, "push")
		if pushErr != nil {
			syncOutput = fmt.Sprintf("Commit exitoso, pero push falló: %s (%s)", pushErr.Error(), pushErrOut)
		} else {
			syncOutput = fmt.Sprintf("Pull: %s | Push: %s", pullOut, pushOut)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Commit realizado con éxito",
		"output":     stdout,
		"syncOutput": syncOutput,
	})
}

// SyncGit runs git fetch, pull and push.
func SyncGit(c *gin.Context) {
	projectID := c.Param("id")
	rootDir, err := getProjectRoot(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pullOut, pullErr, err := runGitCmd(rootDir, "pull", "--rebase")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error en git pull: %s (stderr: %s)", err.Error(), pullErr)})
		return
	}

	pushOut, pushErr, err := runGitCmd(rootDir, "push")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Git pull exitoso, push no pudo completarse (puede que no haya upstream remoto configurado)",
			"pull":    pullOut,
			"pushErr": pushErr,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sincronización Git completada exitosamente",
		"pull":    pullOut,
		"push":    pushOut,
	})
}

// GenerateGitCommitMessage inspects diff and generates a clear conventional commit message via AI.
func GenerateGitCommitMessage(c *gin.Context) {
	projectID := c.Param("id")
	rootDir, err := getProjectRoot(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	diffOut, _, _ := runGitCmd(rootDir, "diff", "--cached")
	if strings.TrimSpace(diffOut) == "" {
		diffOut, _, _ = runGitCmd(rootDir, "diff")
	}

	if strings.TrimSpace(diffOut) == "" {
		c.JSON(http.StatusOK, gin.H{
			"message":    "chore: update workspace files",
			"highlights": []string{"No modified code diffs detected"},
		})
		return
	}

	// Limit diff length for prompt
	if len(diffOut) > 4000 {
		diffOut = diffOut[:4000] + "\n... [diff truncado]"
	}

	avail := providers.Available()
	if len(avail) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "feat: update project files and code structure",
			"highlights": []string{
				"Modified workspace components",
				"Updated codebase structure",
			},
		})
		return
	}

	provider, err := providers.Get(avail[0])
	if err != nil || provider == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "feat: update project files and components",
			"highlights": []string{
				"Applied code adjustments across project files",
			},
		})
		return
	}

	prompt := fmt.Sprintf(`Analiza el siguiente git diff y genera un mensaje de commit en formato Conventional Commits (ej: feat(scope): descripcion concisa).
Devuelve SOLO una línea con el commit message y luego 2-3 viñetas breves con puntos clave.

Diff:
%s`, diffOut)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	stream, err := provider.StreamCompletion(ctx, []providers.Message{
		{Role: "user", Content: prompt},
	}, providers.CompletionOptions{Stream: true})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "feat: update project files",
			"highlights": []string{"Updated project files"},
		})
		return
	}

	var fullResp strings.Builder
	for chunk := range stream {
		if chunk.Type == "text" {
			fullResp.WriteString(chunk.Content)
		}
	}
	resp := fullResp.String()
	if err != nil || resp == "" {
		c.JSON(http.StatusOK, gin.H{
			"message": "feat: update project files and components",
			"highlights": []string{
				"Applied code adjustments across project files",
			},
		})
		return
	}

	lines := strings.Split(strings.TrimSpace(resp), "\n")
	firstLine := strings.TrimSpace(lines[0])
	firstLine = strings.TrimPrefix(firstLine, "Commit:")
	firstLine = strings.TrimPrefix(firstLine, "`")
	firstLine = strings.TrimSuffix(firstLine, "`")
	firstLine = strings.TrimSpace(firstLine)

	var highlights []string
	for _, l := range lines[1:] {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "-") || strings.HasPrefix(l, "*") || strings.HasPrefix(l, "•") {
			clean := strings.TrimSpace(strings.TrimLeft(l, "-*• "))
			if clean != "" {
				highlights = append(highlights, clean)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    firstLine,
		"highlights": highlights,
	})
}
