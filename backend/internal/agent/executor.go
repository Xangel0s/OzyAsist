package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/mcp"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/search"
	"github.com/ozyassist/backend/internal/system"
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

	var projID *string
	if e.ProjectID != "" {
		projID = &e.ProjectID
	}
	userID := e.UserID
	if userID == "" {
		userID = db.DefaultUserID()
	}
	task := &models.AgentTask{
		ID:          e.TaskID,
		UserID:      userID,
		ProjectID:   projID,
		Title:       plan.Goal,
		Prompt:      plan.Goal,
		Status:      models.TaskStatusRunning,
		TotalSteps:  len(plan.Steps),
		CurrentStep: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
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
	if strings.HasPrefix(tc.Name, "mcp_") {
		var args interface{}
		if err := json.Unmarshal(tc.Input, &args); err != nil {
			return fmt.Sprintf("Error parsing MCP arguments: %v", err), false
		}
		res, err := mcp.DefaultRegistry.CallTool(ctx, tc.Name, args)
		if err != nil {
			return fmt.Sprintf("MCP Tool Error: %v", err), false
		}
		
		// Serialize content to string
		out, _ := json.MarshalIndent(res.Content, "", "  ")
		return string(out), !res.IsError
	}

	switch tc.Name {
	case "read_file":
		return execReadFile(ctx, tc, sandbox)
	case "write_file":
		return execWriteFile(ctx, tc, sandbox)
	case "os_search_index":
		return execOSSearchIndex(ctx, tc)
	case "os_save_custom_tool":
		return execOSSaveCustomTool(ctx, tc)
	case "run_command":
		return execRunCommand(ctx, tc, sandbox)
	case "list_files":
		return execListFiles(ctx, tc, sandbox)
	case "search_text":
		return execSearchText(ctx, tc, sandbox)
	case "apply_diff":
		return execApplyDiff(ctx, tc, sandbox)
	case "web_search":
		return execWebSearch(ctx, tc)
	case "deep_search":
		return execDeepSearch(ctx, tc)
	case "web_fetch":
		return execWebFetch(ctx, tc)
	case "web_dns_lookup":
		return execWebDNSLookup(ctx, tc)
	case "os_get_desktop":
		return execOSGetDesktop(ctx)
	case "os_list_apps":
		return execOSListApps(ctx, tc)
	case "os_explore":
		return execOSExplore(ctx, tc)
	case "os_find_files":
		return execOSFindFiles(ctx, tc)
	case "os_active_windows":
		return execOSActiveWindows(ctx)
	case "os_system_info":
		return execOSSystemInfo(ctx)
	case "os_create_dir":
		return execOSCreateDir(ctx, tc)
	case "os_move_item":
		return execOSMoveItem(ctx, tc)
	case "os_copy_item":
		return execOSCopyItem(ctx, tc)
	case "os_delete_item":
		return execOSDeleteItem(ctx, tc)
	case "os_organize_folder":
		return execOSOrganizeFolder(ctx, tc)
	case "os_launch_app":
		return execOSLaunchApp(ctx, tc)
	case "os_hardware_control":
		return execOSHardwareControl(ctx, tc)
	case "os_power_state":
		return execOSPowerState(ctx, tc)
	case "os_focus_window":
		return execOSFocusWindow(ctx, tc)
	case "os_kill_process":
		return execOSKillProcess(ctx, tc)
	case "os_run_command":
		return execOSRunCommand(ctx, tc)
	case "os_draft_email":
		return execOSDraftEmail(ctx, tc)
	case "os_draft_whatsapp":
		return execOSDraftWhatsApp(ctx, tc)
	case "os_draft_telegram":
		return execOSDraftTelegram(ctx, tc)
	case "telegram_send_message":
		return execTelegramSendMessage(ctx, tc)
	case "browser_list_profiles":
		return execBrowserListProfiles(ctx)
	case "os_get_clipboard":
		return execOSGetClipboard(ctx)
	case "os_set_clipboard":
		return execOSSetClipboard(ctx, tc)
	case "os_notify":
		return execOSNotify(ctx, tc)
	case "os_read_document":
		return execOSReadDocument(ctx, tc)
	case "os_schedule_alarm":
		return execOSScheduleAlarm(ctx, tc)
	case "os_list_alarms":
		return execOSListAlarms(ctx)
	case "browser_open_groq":
		return execBrowserOpenGroqConsole(ctx)
	case "os_setup_groq_key":
		return execOSSetupGroqKey(ctx, tc)
	case "os_take_screenshot":
		return execOSTakeScreenshot(ctx)
	case "os_mouse_click":
		return execOSMouseClick(ctx, tc)
	case "os_type_text":
		return execOSTypeText(ctx, tc)
	case "os_create_excel":
		return execOSCreateExcel(ctx, tc)
	case "os_query_db":
		return execOSQueryDB(ctx, tc)
	case "os_analyze_screen":
		return execOSAnalyzeScreen(ctx, tc)
	case "os_watchdog":
		return execOSWatchdog(ctx, tc)
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
	case "write_file", "apply_diff", "os_create_excel":
		return "file_write"
	case "run_command":
		return "command_exec"
	case "list_files", "search_text":
		return "file_read"
	case "os_get_desktop", "os_list_apps", "os_explore", "os_find_files", "os_active_windows", "os_take_screenshot", "browser_list_profiles", "os_get_clipboard", "os_read_document", "os_list_alarms", "os_query_db", "os_analyze_screen":
		return "os_inspect"
	case "os_create_dir", "os_move_item", "os_copy_item", "os_delete_item", "os_organize_folder":
		return "os_mutate"
	case "os_launch_app", "os_focus_window", "os_kill_process", "os_run_command", "os_draft_email", "os_draft_whatsapp", "os_draft_telegram", "telegram_send_message", "os_mouse_click", "os_type_text", "os_set_clipboard", "os_notify", "os_schedule_alarm", "browser_open_groq", "os_setup_groq_key", "os_watchdog":
		return "os_exec"
	case "web_search", "deep_search", "web_fetch", "web_dns_lookup":
		return "web_search"
	case "os_system_info":
		return "system_info"
	default:
		return "os_inspect"
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
		Cwd     string `json:"cwd"`
		Dir     string `json:"dir"`
		Path    string `json:"path"`
		Project string `json:"project"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	if params.Cwd == "" {
		if params.Dir != "" {
			params.Cwd = params.Dir
		} else if params.Path != "" {
			params.Cwd = params.Path
		} else if params.Project != "" {
			params.Cwd = params.Project
		} else if sandbox != nil && sandbox.ProjectRoot != "" && sandbox.ProjectRoot != "." {
			params.Cwd = sandbox.ProjectRoot
		}
	}
	return runOSCommandInternal(ctx, params.Command, params.Cwd)
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
// Si relPath ya es absoluta, la limpia y la retorna directamente.
func resolveSandboxPath(sandbox *Sandbox, relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return filepath.Clean(relPath), nil
	}
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

func execWebSearch(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	results, err := search.DefaultDuckDuckGoSearch(ctx, params.Query)
	if err != nil {
		return fmt.Sprintf("error en búsqueda web: %v", err), false
	}
	if len(results) == 0 {
		return "No se encontraron resultados en la web para la consulta.", true
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Resultados de búsqueda web para \"%s\":\n\n", params.Query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("[%d] %s\n    URL: %s\n    %s\n\n", i+1, r.Title, r.URL, r.Snippet))
	}
	return sb.String(), true
}

func execDeepSearch(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	prov := providers.GetDefaultOrFirstProvider()
	engine := search.NewDeepSearchEngine(prov, search.DefaultDuckDuckGoSearch)
	result, err := engine.Execute(ctx, params.Query)
	if err != nil {
		return fmt.Sprintf("error en deep search: %v", err), false
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== SÍNTESIS DEEP SEARCH: %s ===\n\n", result.Query))
	sb.WriteString(result.Answer)
	sb.WriteString("\n\n=== FUENTES CONSULTADAS ===\n")
	for _, s := range result.Sources {
		sb.WriteString(fmt.Sprintf("[%d] %s (%s)\n", s.ID, s.Title, s.URL))
	}
	if result.ReportFile != "" {
		sb.WriteString(fmt.Sprintf("\n📄 Reporte completo guardado en: %s\n", result.ReportFile))
	}
	return sb.String(), true
}

func execWebFetch(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		URL       string `json:"url"`
		MaxLength int    `json:"max_length"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}

	target := strings.TrimSpace(params.URL)
	if target == "" {
		return "URL vacía", false
	}

	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}

	maxLen := params.MaxLength
	if maxLen <= 0 {
		maxLen = 4000
	}

	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return fmt.Sprintf("Error creando petición para %s: %v", target, err), false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")

	client := &http.Client{
		Timeout: 12 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("demasiadas redirecciones")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error conectando a %s: %v", target, err), false
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return fmt.Sprintf("Error leyendo respuesta de %s: %v", target, err), false
	}

	bodyStr := string(bodyBytes)

	// Extraer metadatos
	titleRegex := regexp.MustCompile(`(?i)<title[^>]*>([\s\S]*?)</title>`)
	descRegex := regexp.MustCompile(`(?i)<meta[^>]+name=["']description["'][^>]+content=["']([\s\S]*?)["']`)
	ogTitleRegex := regexp.MustCompile(`(?i)<meta[^>]+property=["']og:title["'][^>]+content=["']([\s\S]*?)["']`)
	ogDescRegex := regexp.MustCompile(`(?i)<meta[^>]+property=["']og:description["'][^>]+content=["']([\s\S]*?)["']`)
	ogSiteRegex := regexp.MustCompile(`(?i)<meta[^>]+property=["']og:site_name["'][^>]+content=["']([\s\S]*?)["']`)

	title := ""
	if m := titleRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
		title = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	desc := ""
	if m := descRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
		desc = strings.TrimSpace(html.UnescapeString(m[1]))
	} else if m := ogDescRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
		desc = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	ogTitle := ""
	if m := ogTitleRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
		ogTitle = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	ogSite := ""
	if m := ogSiteRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
		ogSite = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	// Limpieza de HTML para texto legible
	textClean := bodyStr
	textClean = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(textClean, "")
	textClean = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(textClean, "")
	textClean = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`).ReplaceAllString(textClean, "")
	textClean = regexp.MustCompile(`(?is)<svg[^>]*>.*?</svg>`).ReplaceAllString(textClean, "")

	blockTagsRegex := regexp.MustCompile(`(?i)</?(p|div|h[1-6]|li|br|section|article|header|footer|tr|td)[^>]*>`)
	textClean = blockTagsRegex.ReplaceAllString(textClean, "\n")

	anyTagRegex := regexp.MustCompile(`<[^>]+>`)
	textClean = anyTagRegex.ReplaceAllString(textClean, " ")

	textClean = html.UnescapeString(textClean)

	multiSpaces := regexp.MustCompile(`[ \t]+`)
	multiNewlines := regexp.MustCompile(`\n{3,}`)
	textClean = multiSpaces.ReplaceAllString(textClean, " ")
	textClean = multiNewlines.ReplaceAllString(textClean, "\n\n")
	textClean = strings.TrimSpace(textClean)

	if len(textClean) > maxLen {
		textClean = textClean[:maxLen] + fmt.Sprintf("\n\n[...contenido truncado a %d caracteres...]", maxLen)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== INSPECCIÓN WEB EN VIVO: %s ===\n", resp.Request.URL.String()))
	sb.WriteString(fmt.Sprintf("- Estado HTTP: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode)))
	if srv := resp.Header.Get("Server"); srv != "" {
		sb.WriteString(fmt.Sprintf("- Servidor / Hosting: %s\n", srv))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		sb.WriteString(fmt.Sprintf("- Content-Type: %s\n", ct))
	}
	if title != "" {
		sb.WriteString(fmt.Sprintf("- Título: %s\n", title))
	}
	if ogSite != "" {
		sb.WriteString(fmt.Sprintf("- Sitio: %s\n", ogSite))
	}
	if ogTitle != "" && ogTitle != title {
		sb.WriteString(fmt.Sprintf("- Título OG: %s\n", ogTitle))
	}
	if desc != "" {
		sb.WriteString(fmt.Sprintf("- Descripción: %s\n", desc))
	}

	sb.WriteString("\n=== CONTENIDO PRINCIPAL EXTRAÍDO ===\n")
	if textClean != "" {
		sb.WriteString(textClean)
	} else {
		sb.WriteString("(No se pudo extraer texto visible en el cuerpo HTML)")
	}

	return sb.String(), resp.StatusCode >= 200 && resp.StatusCode < 400
}

func execWebDNSLookup(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}

	rawDomain := strings.TrimSpace(params.Domain)
	rawDomain = strings.TrimPrefix(rawDomain, "http://")
	rawDomain = strings.TrimPrefix(rawDomain, "https://")
	if slashIdx := strings.Index(rawDomain, "/"); slashIdx != -1 {
		rawDomain = rawDomain[:slashIdx]
	}
	if colonIdx := strings.Index(rawDomain, ":"); colonIdx != -1 {
		rawDomain = rawDomain[:colonIdx]
	}

	if rawDomain == "" {
		return "Dominio vacío", false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== REGISTROS DNS PARA: %s ===\n", rawDomain))

	// 1. Direcciones IP (A / AAAA)
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", rawDomain)
	if err != nil {
		return fmt.Sprintf("Error resolviendo DNS para %s: %v (El dominio no existe o no tiene resolución DNS activa)", rawDomain, err), false
	}

	sb.WriteString(fmt.Sprintf("✓ Direcciones IP encontradas (%d):\n", len(ips)))
	for _, ip := range ips {
		version := "IPv4"
		if ip.To4() == nil {
			version = "IPv6"
		}
		sb.WriteString(fmt.Sprintf("  • [%s] %s\n", version, ip.String()))
	}

	// 2. CNAME
	if cname, err := net.DefaultResolver.LookupCNAME(ctx, rawDomain); err == nil && cname != "" && cname != rawDomain+"." {
		sb.WriteString(fmt.Sprintf("\n- CNAME Alias: %s\n", cname))
	}

	// 3. Registros MX
	if mxs, err := net.DefaultResolver.LookupMX(ctx, rawDomain); err == nil && len(mxs) > 0 {
		sb.WriteString("\n- Servidores de Correo (MX):\n")
		for _, mx := range mxs {
			sb.WriteString(fmt.Sprintf("  • %s (Pref: %d)\n", mx.Host, mx.Pref))
		}
	}

	// 4. Registros TXT (SPF, Verificaciones)
	if txts, err := net.DefaultResolver.LookupTXT(ctx, rawDomain); err == nil && len(txts) > 0 {
		sb.WriteString("\n- Registros TXT:\n")
		for _, txt := range txts {
			if len(txt) > 120 {
				txt = txt[:117] + "..."
			}
			sb.WriteString(fmt.Sprintf("  • %s\n", txt))
		}
	}

	return sb.String(), true
}

func execOSGetDesktop(ctx context.Context) (string, bool) {
	nav := system.NewWindowsNavigator()
	items, err := nav.GetDesktopItems(ctx)
	if err != nil {
		return fmt.Sprintf("Error obteniendo elementos del escritorio: %v", err), false
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== ELEMENTOS DEL ESCRITORIO (%d encontrados) ===\n", len(items)))
	for i, it := range items {
		loc := "Personal"
		if it.IsPublic {
			loc = "Público"
		}
		if it.Kind == system.ItemKindSystemIcon {
			loc = "Sistema"
		}
		sb.WriteString(fmt.Sprintf("%d. [%s | %s] %s\n", i+1, it.Kind, loc, it.Name))
	}
	return sb.String(), true
}

func execOSListApps(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Filter string `json:"filter"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	nav := system.NewWindowsNavigator()
	apps, err := nav.GetInstalledSoftware(ctx, params.Filter)
	if err != nil {
		return fmt.Sprintf("Error consultando registro de aplicaciones: %v", err), false
	}
	if len(apps) == 0 {
		if params.Filter != "" {
			return fmt.Sprintf("No se encontraron aplicaciones instaladas que coincidan con \"%s\".", params.Filter), true
		}
		return "No se encontraron aplicaciones instaladas registradas.", true
	}
	var sb strings.Builder
	title := "APLICACIONES INSTALADAS"
	if params.Filter != "" {
		title += fmt.Sprintf(" (Filtro: \"%s\")", params.Filter)
	}
	sb.WriteString(fmt.Sprintf("=== %s (%d encontradas) ===\n", title, len(apps)))
	for i, app := range apps {
		details := ""
		if app.DisplayVersion != "" {
			details += " v" + app.DisplayVersion
		}
		if app.Publisher != "" {
			details += " (" + app.Publisher + ")"
		}
		if app.InstallLocation != "" {
			details += " en " + app.InstallLocation
		}
		sb.WriteString(fmt.Sprintf("%d. %s%s\n", i+1, app.DisplayName, details))
	}
	return sb.String(), true
}

func execOSExplore(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path  string `json:"path"`
		Depth int    `json:"depth"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	if params.Depth <= 0 {
		params.Depth = 1
	}
	nav := system.NewWindowsNavigator()
	node, err := nav.ExplorePath(ctx, params.Path, params.Depth)
	if err != nil {
		return fmt.Sprintf("Error explorando ruta: %v", err), false
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== EXPLORACIÓN DE: %s (%d elementos) ===\n", node.Path, len(node.Children)))
	for _, c := range node.Children {
		kind := "archivo"
		if c.IsDir {
			kind = "carpeta"
		}
		sb.WriteString(fmt.Sprintf("- [%s] %s (%d bytes, mod: %s)\n", kind, c.Name, c.Size, c.ModifiedAt.Format("2006-01-02 15:04")))
	}
	return sb.String(), true
}

func execOSFindFiles(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Root       string `json:"root"`
		Pattern    string `json:"pattern"`
		MaxResults int    `json:"max_results"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	nav := system.NewWindowsNavigator()
	matches, err := nav.FindFiles(ctx, params.Root, params.Pattern, params.MaxResults)
	if err != nil && len(matches) == 0 {
		return fmt.Sprintf("Error buscando archivos: %v", err), false
	}
	if len(matches) == 0 {
		return fmt.Sprintf("No se encontraron archivos que coincidan con \"%s\" en \"%s\".", params.Pattern, params.Root), true
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== ARCHIVOS ENCONTRADOS (%d coincidencias para \"%s\") ===\n", len(matches), params.Pattern))
	for i, m := range matches {
		sb.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, m.Name, m.Path))
	}
	return sb.String(), true
}

func execOSActiveWindows(ctx context.Context) (string, bool) {
	nav := system.NewWindowsNavigator()
	windows, err := nav.GetActiveWindows(ctx)
	if err != nil {
		return fmt.Sprintf("Error obteniendo ventanas activas: %v", err), false
	}
	if len(windows) == 0 {
		return "No se detectaron ventanas visibles abiertas.", true
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== VENTANAS ABIERTAS EN PANTALLA (%d) ===\n", len(windows)))
	for i, w := range windows {
		sb.WriteString(fmt.Sprintf("%d. %s (PID: %d)\n", i+1, w.Title, w.ProcessID))
	}
	return sb.String(), true
}

func execOSSystemInfo(ctx context.Context) (string, bool) {
	collector := system.NewCollector()
	metrics, err := collector.Collect(ctx)
	if err != nil {
		return fmt.Sprintf("Error obteniendo métricas del sistema: %v", err), false
	}
	return fmt.Sprintf("=== MÉTRICAS DEL SISTEMA ===\n- CPU: %.1f%% (%d núcleos)\n- RAM: %d MB usados / %d MB totales (%.1f%%)\n- Disco: %d GB libres / %d GB totales (%.1f%%)\n- Plataforma: %s",
		metrics.CPUUsagePercent, metrics.CPUCores, metrics.RAMUsedMB, metrics.RAMTotalMB, metrics.RAMUsagePercent,
		metrics.DiskFreeGB, metrics.DiskTotalGB, metrics.DiskUsagePercent, metrics.Platform), true
}

func execOSCreateDir(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	ext := strings.ToLower(filepath.Ext(params.Path))
	if ext == ".xlsx" || ext == ".xls" {
		return "Error: 'os_create_dir' es exclusivamente para carpetas. Para crear un archivo de Excel debes usar la herramienta 'os_create_excel' especificando 'path', 'headers' y 'rows'.", false
	}
	if ext == ".txt" || ext == ".md" || ext == ".csv" || ext == ".json" {
		return "Error: 'os_create_dir' es para carpetas. Para crear archivos de texto usa 'write_file'.", false
	}

	targetPath := system.ResolveUserPath(params.Path)
	nav := system.NewWindowsNavigator()
	if err := nav.CreateDirectory(ctx, targetPath); err != nil {
		return fmt.Sprintf("Error creando carpeta %s: %v", targetPath, err), false
	}
	return fmt.Sprintf("Carpeta creada exitosamente: %s", targetPath), true
}

func execOSMoveItem(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	src := system.ResolveUserPath(params.Src)
	dst := system.ResolveUserPath(params.Dst)
	nav := system.NewWindowsNavigator()
	if err := nav.MoveItem(ctx, src, dst); err != nil {
		return fmt.Sprintf("Error moviendo %s a %s: %v", src, dst, err), false
	}
	return fmt.Sprintf("Elemento movido exitosamente de %s a %s", src, dst), true
}

func execOSCopyItem(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	src := system.ResolveUserPath(params.Src)
	dst := system.ResolveUserPath(params.Dst)
	nav := system.NewWindowsNavigator()
	if err := nav.CopyItem(ctx, src, dst); err != nil {
		return fmt.Sprintf("Error copiando %s a %s: %v", src, dst, err), false
	}
	return fmt.Sprintf("Elemento copiado exitosamente a %s", dst), true
}

func execOSDeleteItem(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path      string `json:"path"`
		Permanent bool   `json:"permanent"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	targetPath := system.ResolveUserPath(params.Path)
	nav := system.NewWindowsNavigator()
	useRecycle := !params.Permanent
	if err := nav.DeleteItem(ctx, targetPath, useRecycle); err != nil {
		return fmt.Sprintf("Error eliminando %s: %v", targetPath, err), false
	}
	targetLoc := "la Papelera de reciclaje"
	if params.Permanent {
		targetLoc = "forma permanente"
	}
	return fmt.Sprintf("Elemento %s eliminado correctamente (%s).", targetPath, targetLoc), true
}

func execOSOrganizeFolder(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	if strings.TrimSpace(params.Path) == "" {
		params.Path = "mis descargas"
	}
	nav := system.NewWindowsNavigator()
	res, err := nav.OrganizeFolder(ctx, params.Path, "extension")
	if err != nil {
		return fmt.Sprintf("Error organizando carpeta %s: %v", params.Path, err), false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== CARPETA ORGANIZADA CON ÉXITO (%s) ===\n", res.SourcePath))
	sb.WriteString(fmt.Sprintf("- Total archivos procesados: %d\n", res.TotalFiles))
	sb.WriteString(fmt.Sprintf("- Total archivos clasificados y movidos: %d\n", res.MovedFiles))
	sb.WriteString("- Desglose por categorías:\n")
	for cat, count := range res.CategorizedMap {
		sb.WriteString(fmt.Sprintf("  • %s: %d archivos\n", cat, count))
	}
	if len(res.Errors) > 0 {
		sb.WriteString("\nAdvertencias:\n")
		for _, e := range res.Errors {
			sb.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}
	return sb.String(), true
}

func execOSLaunchApp(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Target   string   `json:"target"`
		AppName  string   `json:"appName"`
		App_Name string   `json:"app_name"`
		Path     string   `json:"path"`
		Args     []string `json:"args"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	target := strings.TrimSpace(params.Target)
	if target == "" {
		target = strings.TrimSpace(params.AppName)
	}
	if target == "" {
		target = strings.TrimSpace(params.App_Name)
	}

	args := params.Args
	if strings.TrimSpace(params.Path) != "" {
		args = append(args, strings.TrimSpace(params.Path))
	}

	cleanLower := strings.ToLower(target)
	// Limpiar prefijos conversacionales comunes
	for _, prefix := range []string{"abrir con ", "abre con ", "abrir ", "abre ", "lanza ", "ejecuta ", "el ", "la ", "los ", "las ", "un ", "una "} {
		if strings.HasPrefix(cleanLower, prefix) {
			cleanLower = strings.TrimSpace(cleanLower[len(prefix):])
			target = strings.TrimSpace(target[len(prefix):])
		}
	}

	// Si no se pasaron args pero el target contiene una app seguida de un proyecto o ruta (ej: "antigravity crmgeofal", "code mi_proyecto")
	if len(args) == 0 {
		appPrefixes := []string{"antigravity ide ", "antigravity ", "code ", "vscode ", "notepad ", "calc ", "chrome ", "brave ", "explorer "}
		for _, ap := range appPrefixes {
			if strings.HasPrefix(cleanLower, ap) {
				target = strings.TrimSpace(target[:len(ap)])
				remPath := strings.TrimSpace(cleanLower[len(ap):])
				if remPath != "" {
					args = append(args, remPath)
				}
				break
			}
		}
	}

	nav := system.NewWindowsNavigator()
	if err := nav.LaunchApplication(ctx, target, args); err != nil {
		return fmt.Sprintf("Error ejecutando aplicación %s: %v", target, err), false
	}
	if len(args) > 0 {
		return fmt.Sprintf("Aplicación %s lanzada exitosamente con: %s", target, strings.Join(args, " ")), true
	}
	return fmt.Sprintf("Aplicación lanzada exitosamente: %s", target), true
}

func execOSFocusWindow(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		HWND uintptr `json:"hwnd"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	nav := system.NewWindowsNavigator()
	if err := nav.FocusWindow(ctx, params.HWND); err != nil {
		return fmt.Sprintf("Error enfocando ventana (HWND: %d): %v", params.HWND, err), false
	}
	return fmt.Sprintf("Ventana (HWND: %d) enfocada y traída al frente.", params.HWND), true
}

func execOSKillProcess(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		PID     uint32 `json:"pid"`
		Name    string `json:"name"`
		AppName string `json:"appName"`
		Force   bool   `json:"force"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	if !params.Force {
		params.Force = true
	}

	nav := system.NewWindowsNavigator()

	if params.PID > 0 {
		_ = nav.KillProcess(ctx, params.PID, params.Force)
		psCmd := fmt.Sprintf(`$p = Get-Process -Id %d -ErrorAction SilentlyContinue; if ($p) { Stop-Process -Name $p.ProcessName -Force -ErrorAction SilentlyContinue }`, params.PID)
		_ = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Run()
		return fmt.Sprintf("Proceso PID %d y sus instancias asociadas terminados exitosamente.", params.PID), true
	}

	target := strings.TrimSpace(params.Name)
	if target == "" {
		target = strings.TrimSpace(params.AppName)
	}

	cleanLower := strings.ToLower(target)
	for _, prefix := range []string{"cerrar ", "cierra ", "matar ", "mata ", "el ", "la ", "los ", "las "} {
		if strings.HasPrefix(cleanLower, prefix) {
			cleanLower = strings.TrimSpace(cleanLower[len(prefix):])
			target = cleanLower
		}
	}

	targetsToKill := []string{}
	switch cleanLower {
	case "bloc de notas", "bloc", "notas", "notepad", "notepad.exe":
		targetsToKill = []string{"notepad.exe", "Notepad.exe"}
	case "calculadora", "calc", "calc.exe", "calculator", "calculatorapp":
		targetsToKill = []string{"CalculatorApp.exe", "calc.exe", "Calculator.exe"}
	case "explorador", "explorador de archivos", "explorer", "explorer.exe":
		targetsToKill = []string{"explorer.exe"}
	case "chrome", "google chrome":
		targetsToKill = []string{"chrome.exe"}
	case "edge", "microsoft edge":
		targetsToKill = []string{"msedge.exe"}
	case "spotify":
		targetsToKill = []string{"spotify.exe"}
	case "paint", "mspaint":
		targetsToKill = []string{"mspaint.exe"}
	case "excel":
		targetsToKill = []string{"excel.exe", "EXCEL.EXE"}
	default:
		if cleanLower != "" {
			if !strings.HasSuffix(cleanLower, ".exe") {
				targetsToKill = []string{cleanLower + ".exe", cleanLower}
			} else {
				targetsToKill = []string{cleanLower}
			}
		}
	}

	killedAny := false
	for _, img := range targetsToKill {
		args := []string{"/IM", img}
		if params.Force {
			args = append(args, "/F", "/T")
		}
		cmd := exec.CommandContext(ctx, "taskkill", args...)
		if err := cmd.Run(); err == nil {
			killedAny = true
		}
	}

	// Fallback adicional con PowerShell Stop-Process para procesos UWP
	if !killedAny {
		for _, img := range targetsToKill {
			procBase := strings.TrimSuffix(img, ".exe")
			psCmd := fmt.Sprintf("Stop-Process -Name '%s' -Force -ErrorAction SilentlyContinue", procBase)
			_ = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Run()
		}
	}

	if killedAny {
		return fmt.Sprintf("Aplicación %s cerrada exitosamente.", target), true
	}

	// Fallback inteligente: buscar en las ventanas activas por coincidencia de título
	if wins, err := nav.GetActiveWindows(ctx); err == nil && cleanLower != "" {
		for _, w := range wins {
			if strings.Contains(strings.ToLower(w.Title), cleanLower) {
				if err := nav.KillProcess(ctx, w.ProcessID, true); err == nil {
					return fmt.Sprintf("Ventana '%s' (PID %d) cerrada exitosamente.", w.Title, w.ProcessID), true
				}
			}
		}
	}

	return fmt.Sprintf("No se encontró ningún proceso activo para '%s'.", target), false
}

func execOSHardwareControl(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Setting string `json:"setting"`
		Value   int    `json:"value"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	out, err := OSHardwareControl(ctx, params.Setting, params.Value)
	if err != nil {
		return err.Error(), false
	}
	return out, true
}

func execOSPowerState(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		State string `json:"state"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	out, err := OSPowerState(ctx, params.State)
	if err != nil {
		return err.Error(), false
	}
	return out, true
}

func execOSRunCommand(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Command string `json:"command"`
		Cwd     string `json:"cwd"`
		Dir     string `json:"dir"`
		Path    string `json:"path"`
		Project string `json:"project"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_run_command: %v", err), false
	}
	if params.Cwd == "" {
		if params.Dir != "" {
			params.Cwd = params.Dir
		} else if params.Path != "" {
			params.Cwd = params.Path
		} else if params.Project != "" {
			params.Cwd = params.Project
		}
	}
	return runOSCommandInternal(ctx, params.Command, params.Cwd)
}

func runOSCommandInternal(ctx context.Context, command, rawCwd string) (string, bool) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "comando vacío", false
	}

	// Verificación de seguridad básica contra comandos destructivos no deseados
	if IsDestructiveCommand(command) {
		return fmt.Sprintf("⚠️ COMANDO BLOQUEADO POR SEGURIDAD: '%s' contiene operaciones destructivas que requieren autorización explícita.", command), false
	}

	cwd := strings.TrimSpace(rawCwd)
	if cwd != "" {
		cwd = system.ResolveUserPath(cwd)
	}

	// Si cwd no fue especificado, buscar si el mensaje del usuario menciona un proyecto conocido en Documents
	if cwd == "" {
		userMsg := strings.ToLower(UserMessageFromContext(ctx))
		if userMsg != "" {
			userProfile := os.Getenv("USERPROFILE")
			if userProfile != "" {
				docsDir := filepath.Join(userProfile, "Documents")
				if entries, err := os.ReadDir(docsDir); err == nil {
					for _, entry := range entries {
						if entry.IsDir() {
							folderLower := strings.ToLower(entry.Name())
							if len(folderLower) >= 4 && strings.Contains(userMsg, folderLower) {
								cwd = filepath.Join(docsDir, entry.Name())
								break
							}
						}
					}
				}
			}
		}
	}

	// Comprobar si el comando es de Git
	isGit := strings.HasPrefix(strings.ToLower(command), "git ") || strings.HasPrefix(strings.ToLower(command), "git.exe ")
	if isGit && cwd != "" {
		if fi, err := os.Stat(cwd); err == nil && fi.IsDir() {
			gitDir := filepath.Join(cwd, ".git")
			if _, err := os.Stat(gitDir); os.IsNotExist(err) {
				// La carpeta no tiene .git en la raíz. Verificar si contiene sub-repositorios Git independientes
				if multiRepoOut, hasRepos := inspectGitMultiRepo(ctx, cwd, command); hasRepos {
					return multiRepoOut, true
				}
			}
		}
	}

	res, healLogs, err := executeWithSelfHealing(ctx, command, cwd)
	if err != nil {
		return fmt.Sprintf("error ejecutando comando '%s': %v", command, err), false
	}

	// Si el comando git falló con "not a git repository", intentar inspección multi-repo
	if isGit && (strings.Contains(res.Stderr, "not a git repository") || strings.Contains(res.Stdout, "not a git repository")) && cwd != "" {
		if multiRepoOut, hasRepos := inspectGitMultiRepo(ctx, cwd, command); hasRepos {
			return multiRepoOut, true
		}
	}

	var sb strings.Builder
	if healLogs != "" {
		sb.WriteString(healLogs)
	}
	effectiveDir := cwd
	if effectiveDir == "" {
		effectiveDir, _ = os.Getwd()
	}
	sb.WriteString(fmt.Sprintf("💻 [DIR: %s] [COMANDO: %s] (Salida: código %d, duración %s)\n", effectiveDir, command, res.ExitCode, res.Duration))

	stdout := strings.TrimSpace(res.Stdout)
	stderr := strings.TrimSpace(res.Stderr)

	// Truncar para no desbordar tokens si el comando produce salida kilométrica
	if len([]rune(stdout)) > 4000 {
		runes := []rune(stdout)
		stdout = string(runes[:4000]) + "\n[...salida truncada a 4000 caracteres...]"
	}
	if len([]rune(stderr)) > 2000 {
		runes := []rune(stderr)
		stderr = string(runes[:2000]) + "\n[...errores truncados...]"
	}

	if stdout != "" {
		sb.WriteString("--- SALIDA (STDOUT) ---\n")
		sb.WriteString(stdout)
		sb.WriteString("\n")
	}
	if stderr != "" {
		sb.WriteString("--- ERRORES (STDERR) ---\n")
		sb.WriteString(stderr)
		sb.WriteString("\n")
	}
	if stdout == "" && stderr == "" {
		sb.WriteString("(Comando ejecutado exitosamente sin salida en consola)\n")
	}

	return sb.String(), res.ExitCode == 0
}

func inspectGitMultiRepo(ctx context.Context, dir, gitCommand string) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}

	var subRepos []string
	for _, entry := range entries {
		if entry.IsDir() {
			subGit := filepath.Join(dir, entry.Name(), ".git")
			if _, err := os.Stat(subGit); err == nil {
				subRepos = append(subRepos, entry.Name())
			}
		}
	}

	if len(subRepos) == 0 {
		return "", false
	}

	// Extraer argumentos de git (ej: "log -n 3 --oneline", "status")
	lower := strings.ToLower(gitCommand)
	gitArgs := gitCommand
	if strings.HasPrefix(lower, "git ") {
		gitArgs = strings.TrimSpace(gitCommand[4:])
	} else if strings.HasPrefix(lower, "git.exe ") {
		gitArgs = strings.TrimSpace(gitCommand[8:])
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📁 [ESPACIO DE TRABAJO MULTI-REPO: %s]\n", dir))
	sb.WriteString(fmt.Sprintf("Nota: Esta carpeta no es un único repositorio .git, sino un proyecto que contiene %d repositorios Git independientes.\n", len(subRepos)))
	sb.WriteString(fmt.Sprintf("Resultados reales de 'git %s' en cada repositorio:\n\n", gitArgs))

	hasSuccess := false
	for _, repoName := range subRepos {
		repoPath := filepath.Join(dir, repoName)
		res, err := ExecuteShellInDir(ctx, "powershell", "git "+gitArgs, repoPath)
		if err == nil && res.ExitCode == 0 {
			hasSuccess = true
			out := strings.TrimSpace(res.Stdout)
			if out == "" {
				out = "(Sin cambios o sin salida)"
			}
			sb.WriteString(fmt.Sprintf("🔹 Repositorio [%s]:\n%s\n\n", repoName, out))
		} else {
			errText := strings.TrimSpace(res.Stderr)
			if errText == "" && err != nil {
				errText = err.Error()
			}
			sb.WriteString(fmt.Sprintf("🔸 Repositorio [%s] (error): %s\n\n", repoName, errText))
		}
	}

	return sb.String(), hasSuccess
}
