package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/mcp"
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

	tc.Name = strings.ToLower(strings.TrimSpace(tc.Name))
	switch tc.Name {
	case "remember_fact":
		return execRememberFact(ctx, tc)
	case "search_memory":
		return execSearchMemory(ctx, tc)
	case "update_user_profile":
		return execUpdateUserProfile(ctx, tc)
	case "learn_engram":
		return execLearnEngram(ctx, tc)
	case "read_file":
		return execReadFile(ctx, tc, sandbox)
	case "write_file":
		return execWriteFile(ctx, tc, sandbox)
	case "os_search_index":
		return execOSSearchIndex(ctx, tc)
	case "os_save_custom_tool":
		return execOSSaveCustomTool(ctx, tc)
	case "os_prepare_staging":
		return execOSPrepareStaging(ctx, tc)
	case "os_commit_staging":
		return execOSCommitStaging(ctx, tc)
	case "os_backup_file":
		return execOSBackupFile(ctx, tc)
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
	case "os_file_info":
		return execOSFileInfo(ctx, tc)
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
	case "os_close_window":
		return execOSCloseWindow(ctx, tc)
	case "os_tile_windows":
		return execOSTileWindows(ctx, tc)
	case "os_kill_process":
		return execOSKillProcess(ctx, tc)
	case "os_service_manager":
		return execOSServiceManager(ctx, tc)
	case "os_docker_manager":
		return execOSDockerManager(ctx, tc)
	case "os_analyze_logs":
		return execOSAnalyzeLogs(ctx, tc)
	case "os_disk_cleaner":
		return execOSDiskCleaner(ctx, tc)
	case "os_audio_device":
		return execOSAudioDevice(ctx, tc)
	case "os_wifi_manager":
		return execOSWifiManager(ctx, tc)
	case "os_schedule_task":
		return execOSScheduleTask(ctx, tc)
	case "os_hardware_inspector":
		return execOSHardwareInspector(ctx, tc)
	case "os_power_profile":
		return execOSPowerProfile(ctx, tc)
	case "os_toast_notify":
		return execOSToastNotify(ctx, tc)
	case "os_network_diagnostics":
		return execOSNetworkDiagnostics(ctx, tc)
	case "os_smart_organizer":
		return execOSSmartOrganizer(ctx, tc)
	case "os_keyboard_layout":
		return execOSKeyboardLayout(ctx, tc)
	case "os_startup_manager":
		return execOSStartupManager(ctx, tc)
	case "os_notification_focus":
		return execOSNotificationFocus(ctx, tc)
	case "os_media_control":
		return execOSMediaControl(ctx, tc)
	case "os_display_config":
		return execOSDisplayConfig(ctx, tc)
	case "os_process_sentinel":
		return execOSProcessSentinel(ctx, tc)
	case "os_speak_text":
		return execOSSpeakText(ctx, tc)
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
	case "os_detect_dialogs":
		return execOSDetectDialogs(ctx, tc)
	case "os_create_pdf":
		return execOSCreatePDF(ctx, tc)
	case "os_convert_to_pdf":
		return execOSConvertToPDF(ctx, tc)
	case "os_compress_zip":
		return execOSCompressZip(ctx, tc)
	case "os_extract_zip":
		return execOSExtractZip(ctx, tc)
	case "os_search_content":
		return execOSSearchContent(ctx, tc)
	case "os_port_inspector":
		return execOSPortInspector(ctx, tc)
	case "os_download_file":
		return execOSDownloadFile(ctx, tc)
	case "os_create_docx":
		return execOSCreateDocx(ctx, tc)
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
	case "write_file", "apply_diff", "os_create_excel", "os_create_pdf", "os_convert_to_pdf", "os_create_docx":
		return "file_write"
	case "run_command":
		return "command_exec"
	case "list_files", "search_text":
		return "file_read"
	case "os_get_desktop", "os_list_apps", "os_explore", "os_find_files", "os_active_windows", "os_take_screenshot", "browser_list_profiles", "os_get_clipboard", "os_read_document", "os_list_alarms", "os_query_db", "os_analyze_screen", "os_detect_dialogs", "os_search_content", "os_port_inspector", "os_analyze_logs", "os_wifi_manager", "os_hardware_inspector", "os_network_diagnostics", "os_smart_organizer", "os_process_sentinel":
		return "os_inspect"
	case "os_create_dir", "os_move_item", "os_copy_item", "os_delete_item", "os_organize_folder", "os_compress_zip", "os_extract_zip", "os_download_file", "os_disk_cleaner":
		return "os_mutate"

	case "os_launch_app", "os_focus_window", "os_close_window", "os_tile_windows", "os_kill_process", "os_service_manager", "os_docker_manager", "os_audio_device", "os_schedule_task", "os_power_profile", "os_toast_notify", "os_keyboard_layout", "os_startup_manager", "os_notification_focus", "os_media_control", "os_display_config", "os_speak_text", "os_run_command", "os_draft_email", "os_draft_whatsapp", "os_draft_telegram", "telegram_send_message", "os_mouse_click", "os_type_text", "os_set_clipboard", "os_notify", "os_schedule_alarm", "browser_open_groq", "os_setup_groq_key", "os_watchdog":
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

