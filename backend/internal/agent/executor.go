package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

// Executor es el runtime de ejecución de herramientas del agente.
// Se mantiene para backward-compat con las rutas HTTP de agent tasks.
type Executor struct {
	Provider     providers.Provider
	Sandbox      *Sandbox
	Level        PermissionLevel
	ChatID       string
	ProjectID    string
	UserID       string
	TaskID       string
	StepCallback func(StepResult)
}

// StepResult es el resultado de un paso de ejecución (usado por el API de agent tasks HTTP).
type StepResult struct {
	StepID  int    `json:"stepId"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// ExecutePlan ejecuta un plan de pasos estáticos (mantenido para backward-compat).
func (e *Executor) ExecutePlan(ctx context.Context, plan *Plan) ([]StepResult, error) {
	if e.TaskID == "" {
		e.TaskID = uuid.NewString()
	}

	planJSON, _ := json.Marshal(plan.Steps)
	task := &models.AgentTask{
		ID:              e.TaskID,
		ProjectID:       e.ProjectID,
		ChatID:          e.ChatID,
		Goal:            plan.Goal,
		PlanJSON:        string(planJSON),
		Status:          "running",
		PermissionLevel: string(e.Level),
		CreatedAt:       time.Now(),
	}

	if err := db.CreateAgentTask(task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	completed := make(map[int]string)
	var results []StepResult

	for _, step := range plan.Steps {
		depsOk := true
		for _, depID := range step.DependsOn {
			if _, ok := completed[depID]; !ok {
				depsOk = false
				break
			}
		}
		if !depsOk {
			results = append(results, StepResult{StepID: step.ID, Success: false, Error: "dependency not met"})
			continue
		}

		result := e.executeStep(ctx, step, completed)
		results = append(results, result)
		if e.StepCallback != nil {
			e.StepCallback(result)
		}

		if result.Success {
			completed[step.ID] = result.Output
		} else if result.Error == "requires confirmation" {
			db.UpdateTaskStatus(e.TaskID, "awaiting_confirmation", result.Error)
			return results, nil
		} else {
			db.UpdateTaskStatus(e.TaskID, "failed", result.Error)
			return results, fmt.Errorf("step %d failed: %s", step.ID, result.Error)
		}
	}

	db.UpdateTaskStatus(e.TaskID, "completed", "")
	return results, nil
}

func (e *Executor) executeStep(ctx context.Context, step PlanStep, completed map[int]string) StepResult {
	log.Printf("Agent step %d: %s (%s)", step.ID, step.Description, step.ActionType)

	action := &models.AgentAction{
		ID:         uuid.NewString(),
		TaskID:     e.TaskID,
		ActionType: step.ActionType,
		Target:     step.Target,
		CreatedAt:  time.Now(),
	}

	auth, err := ValidateAndAuthorize(step, e.Level, e.Sandbox)
	if err != nil {
		action.Result = err.Error()
		db.CreateAgentAction(action)
		return StepResult{StepID: step.ID, Success: false, Error: err.Error()}
	}

	if auth.RequiresConfirmation {
		action.RequiresConfirmation = true
		action.Result = "awaiting confirmation"
		db.CreateAgentAction(action)
		return StepResult{StepID: step.ID, Success: false, Error: "requires confirmation"}
	}

	switch step.ActionType {
	case "file_read":
		return e.execFileRead(ctx, auth, action)
	case "file_write":
		return e.execFileWrite(ctx, auth, action)
	case "command_exec":
		return e.execCommand(ctx, auth, action)
	default:
		action.Result = "unknown action type: " + step.ActionType
		db.CreateAgentAction(action)
		return StepResult{StepID: step.ID, Success: false, Error: action.Result}
	}
}

func (e *Executor) execFileRead(ctx context.Context, auth *AuthorizedAction, action *models.AgentAction) StepResult {
	data, err := os.ReadFile(auth.ResolvedPath)
	if err != nil {
		action.Result = fmt.Sprintf("read file: %v", err)
		db.CreateAgentAction(action)
		return StepResult{StepID: auth.Step.ID, Success: false, Error: action.Result}
	}
	action.Result = string(data)
	db.CreateAgentAction(action)
	return StepResult{StepID: auth.Step.ID, Success: true, Output: string(data)}
}

func (e *Executor) execFileWrite(ctx context.Context, auth *AuthorizedAction, action *models.AgentAction) StepResult {
	dir := filepath.Dir(auth.ResolvedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		action.Result = fmt.Sprintf("create dir: %v", err)
		db.CreateAgentAction(action)
		return StepResult{StepID: auth.Step.ID, Success: false, Error: action.Result}
	}
	if err := os.WriteFile(auth.ResolvedPath, []byte(auth.Step.Input), 0644); err != nil {
		action.Result = fmt.Sprintf("write file: %v", err)
		db.CreateAgentAction(action)
		return StepResult{StepID: auth.Step.ID, Success: false, Error: action.Result}
	}
	action.Result = fmt.Sprintf("wrote %d bytes to %s", len(auth.Step.Input), auth.ResolvedPath)
	db.CreateAgentAction(action)
	return StepResult{StepID: auth.Step.ID, Success: true, Output: action.Result}
}

func (e *Executor) execCommand(ctx context.Context, auth *AuthorizedAction, action *models.AgentAction) StepResult {
	dir := ""
	if e.Sandbox != nil {
		dir = e.Sandbox.ProjectRoot
	}
	result, err := ExecuteShellInDir(ctx, "powershell", auth.Step.Target, dir)
	if err != nil {
		action.Result = fmt.Sprintf("command error: %v", err)
		db.CreateAgentAction(action)
		return StepResult{StepID: auth.Step.ID, Success: false, Error: action.Result}
	}
	output := result.Stdout
	if result.Stderr != "" {
		output += "\nSTDERR: " + result.Stderr
	}
	action.Result = output
	db.CreateAgentAction(action)
	return StepResult{StepID: auth.Step.ID, Success: result.ExitCode == 0, Output: output}
}

// =============================================================================
// executeToolCall — Dispatcher central para el ReAct Loop
// =============================================================================

// executeToolCall ejecuta una herramienta del agente (llamado desde loop.go).
// Devuelve (output string, success bool).
func executeToolCall(ctx context.Context, tc providers.ToolCall, auth *AuthorizedAction, sandbox *Sandbox) (string, bool) {
	switch tc.Name {
	case "read_file":
		return execReadFile(ctx, tc, sandbox)
	case "write_file":
		return execWriteFile(ctx, tc, sandbox)
	case "run_command":
		return execRunCommand(ctx, tc, sandbox)
	case "list_files":
		return execListFiles(ctx, tc, sandbox)
	case "search_text":
		return execSearchText(ctx, tc, sandbox)
	case "apply_diff":
		return execApplyDiff(ctx, tc, sandbox)
	default:
		return fmt.Sprintf("herramienta desconocida: %s", tc.Name), false
	}
}

// toolCallToPlanStep convierte un ToolCall del ReAct loop al PlanStep legacy
// para que ValidateAndAuthorize pueda evaluarlo con los mismos permisos.
func toolCallToPlanStep(tc providers.ToolCall) PlanStep {
	var params map[string]string
	json.Unmarshal(tc.Input, &params)
	target := params["path"]
	if target == "" {
		target = params["command"]
	}
	if target == "" {
		target = params["query"]
	}
	actionType := toolNameToActionType(tc.Name)
	return PlanStep{
		ID:          0,
		Description: tc.Name,
		ActionType:  actionType,
		Target:      target,
		Input:       params["content"],
	}
}

func toolNameToActionType(name string) string {
	switch name {
	case "read_file":
		return "file_read"
	case "write_file", "apply_diff":
		return "file_write"
	case "run_command":
		return "command_exec"
	case "list_files", "search_text":
		return "file_read"
	default:
		return "file_read"
	}
}

// --- Implementaciones de herramientas individuales ---

func execReadFile(_ context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	path, err := resolveSandboxPath(sandbox, params.Path)
	if err != nil {
		return err.Error(), false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("error leyendo %s: %v", params.Path, err), false
	}
	return string(data), true
}

func execWriteFile(_ context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	path, err := resolveSandboxPath(sandbox, params.Path)
	if err != nil {
		return err.Error(), false
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Sprintf("error creando directorio: %v", err), false
	}
	if err := os.WriteFile(path, []byte(params.Content), 0644); err != nil {
		return fmt.Sprintf("error escribiendo %s: %v", params.Path, err), false
	}
	return fmt.Sprintf("✓ Archivo escrito: %s (%d bytes)", params.Path, len(params.Content)), true
}

func execRunCommand(ctx context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	dir := ""
	if sandbox != nil {
		dir = sandbox.ProjectRoot
	}
	result, err := ExecuteStreamingCommand(ctx, params.Command, dir)
	if err != nil {
		return fmt.Sprintf("error ejecutando comando: %v", err), false
	}
	var out strings.Builder
	if result.Stdout != "" {
		out.WriteString(result.Stdout)
	}
	if result.Stderr != "" {
		if out.Len() > 0 {
			out.WriteString("\n")
		}
		out.WriteString("STDERR:\n" + result.Stderr)
	}
	out.WriteString(fmt.Sprintf("\n[exit code: %d, duración: %s]", result.ExitCode, result.Duration))
	return out.String(), result.ExitCode == 0
}

func execListFiles(_ context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Pattern    string `json:"pattern"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}

	root := "."
	if sandbox != nil {
		root = sandbox.ProjectRoot
	}

	var matches []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "vendor") {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, path)
		matched, _ := filepath.Match(strings.ReplaceAll(params.Pattern, "**", "*"), rel)
		if !matched {
			// También intentar match solo por extensión
			if strings.HasPrefix(params.Pattern, "**/*.") {
				ext := strings.TrimPrefix(params.Pattern, "**/*")
				matched = strings.HasSuffix(path, ext)
			}
		}
		if matched && !d.IsDir() {
			matches = append(matches, rel)
		}
		if len(matches) >= maxResults {
			return fmt.Errorf("max")
		}
		return nil
	})
	_ = err // ignorar error de max

	if len(matches) == 0 {
		return fmt.Sprintf("No se encontraron archivos con el patrón: %s", params.Pattern), true
	}
	return strings.Join(matches, "\n") + fmt.Sprintf("\n\n[%d archivos]", len(matches)), true
}

func execSearchText(_ context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Query         string `json:"query"`
		Include       string `json:"include"`
		CaseSensitive bool   `json:"case_sensitive"`
		MaxResults    int    `json:"max_results"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 30
	}

	root := "."
	if sandbox != nil {
		root = sandbox.ProjectRoot
	}

	flags := 0
	_ = flags
	var re *regexp.Regexp
	var err error
	if !params.CaseSensitive {
		re, err = regexp.Compile("(?i)" + params.Query)
	} else {
		re, err = regexp.Compile(params.Query)
	}
	if err != nil {
		return fmt.Sprintf("regex inválido '%s': %v", params.Query, err), false
	}

	type Match struct {
		File string
		Line int
		Text string
	}
	var matches []Match

	filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			if d != nil && d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if params.Include != "" {
			matched, _ := filepath.Match(params.Include, filepath.Base(path))
			if !matched {
				return nil
			}
		}
		// Limitar a archivos de texto razonables
		if info, err := d.Info(); err == nil && info.Size() > 1<<20 { // >1MB skip
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)

		scanner := bufio.NewScanner(bytes.NewReader(data))
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if re.MatchString(line) {
				matches = append(matches, Match{File: rel, Line: lineNum, Text: strings.TrimSpace(line)})
				if len(matches) >= maxResults {
					return fmt.Errorf("max")
				}
			}
		}
		return nil
	})

	if len(matches) == 0 {
		return fmt.Sprintf("No se encontraron coincidencias para: %s", params.Query), true
	}

	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("%s:%d: %s\n", m.File, m.Line, m.Text))
	}
	sb.WriteString(fmt.Sprintf("\n[%d coincidencias]", len(matches)))
	return sb.String(), true
}

func execApplyDiff(_ context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Path string `json:"path"`
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	path, err := resolveSandboxPath(sandbox, params.Path)
	if err != nil {
		return err.Error(), false
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("error leyendo %s: %v", params.Path, err), false
	}
	patched, err := applyUnifiedDiff(string(original), params.Diff)
	if err != nil {
		return fmt.Sprintf("error aplicando diff: %v", err), false
	}
	if err := os.WriteFile(path, []byte(patched), 0644); err != nil {
		return fmt.Sprintf("error escribiendo %s: %v", params.Path, err), false
	}
	return fmt.Sprintf("✓ Diff aplicado a %s", params.Path), true
}

// resolveSandboxPath resuelve una ruta relativa al sandbox y previene path traversal.
func resolveSandboxPath(sandbox *Sandbox, relPath string) (string, error) {
	if sandbox == nil {
		return relPath, nil
	}
	clean := filepath.Clean(relPath)
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("ruta inválida (path traversal): %s", relPath)
	}
	return filepath.Join(sandbox.ProjectRoot, clean), nil
}

// applyUnifiedDiff aplica un diff unificado simple línea a línea.
// Para diffs complejos con múltiples hunks.
func applyUnifiedDiff(original, diff string) (string, error) {
	origLines := strings.Split(original, "\n")
	diffLines := strings.Split(diff, "\n")

	result := make([]string, 0, len(origLines))
	origIdx := 0

	i := 0
	for i < len(diffLines) {
		line := diffLines[i]

		// Saltar headers del diff (--- +++)
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			i++
			continue
		}

		// Hunk header: @@ -a,b +c,d @@
		if strings.HasPrefix(line, "@@") {
			var startOrig, countOrig int
			fmt.Sscanf(line, "@@ -%d,%d", &startOrig, &countOrig)
			startOrig-- // 0-indexed

			// Copiar líneas originales hasta el hunk
			for origIdx < startOrig && origIdx < len(origLines) {
				result = append(result, origLines[origIdx])
				origIdx++
			}
			i++
			continue
		}

		if strings.HasPrefix(line, "+") {
			result = append(result, strings.TrimPrefix(line, "+"))
			i++
		} else if strings.HasPrefix(line, "-") {
			origIdx++ // saltar línea eliminada
			i++
		} else if strings.HasPrefix(line, " ") {
			// Contexto — usar línea original
			if origIdx < len(origLines) {
				result = append(result, origLines[origIdx])
				origIdx++
			}
			i++
		} else {
			i++
		}
	}

	// Copiar el resto de líneas originales
	for origIdx < len(origLines) {
		result = append(result, origLines[origIdx])
		origIdx++
	}

	return strings.Join(result, "\n"), nil
}
