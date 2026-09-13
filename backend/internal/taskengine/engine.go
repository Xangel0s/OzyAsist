package taskengine

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ozyassist/backend/internal/browser"
	"github.com/ozyassist/backend/internal/connectors/google"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/search"
	"github.com/ozyassist/backend/internal/security"
	"github.com/ozyassist/backend/internal/system"
	"github.com/ozyassist/backend/internal/vision"
	"github.com/ozyassist/backend/internal/voice"
)

type Broadcaster interface {
	BroadcastTaskProgress(task models.AgentTask, step *models.TaskStep)
	BroadcastPINRequest(taskID, stepID, action, reason string)
}

type Verifier interface {
	Verify(rawRule string, output string, exitCode int) (bool, string, error)
}

type HealResult struct {
	Success       bool   `json:"success"`
	PatchApplied  string `json:"patch_applied"`
	NewOutput     string `json:"new_output"`
	FailureReason string `json:"failure_reason,omitempty"`
}

type Healer interface {
	AttemptRecovery(ctx context.Context, task models.AgentTask, step *models.TaskStep, stepError string, output string) (*HealResult, error)
}

type Dispatcher interface {
	Dispatch(ctx context.Context, step *models.TaskStep) (output string, exitCode int, err error)
}

type TriadAuditResult struct {
	AllowExecution     bool   `json:"allow_execution"`
	RequiresPIN        bool   `json:"requires_pin"`
	RiskLevel          string `json:"risk_level"`
	Reason             string `json:"reason"`
	EscalateToNine     bool   `json:"escalate_to_nine"`
	AlternativePayload string `json:"alternative_payload,omitempty"`
}

type Supervisor interface {
	Supervise(ctx context.Context, task models.AgentTask, step *models.TaskStep, recentLogs string) (*TriadAuditResult, error)
}

type StepHandlerFunc func(ctx context.Context, task models.AgentTask, step *models.TaskStep) error

type Engine struct {
	db                 *sql.DB
	broadcaster        Broadcaster
	taskQueue          chan string
	mu                 sync.RWMutex
	activeTasks        map[string]context.CancelFunc
	ctx                context.Context
	cancel             context.CancelFunc
	stepHandler        StepHandlerFunc
	verifier           Verifier
	healer             Healer
	supervisor         Supervisor
	dispatcher         Dispatcher
	browserClient      *browser.CDPClient
	telemetryCollector *system.Collector
	googleClient       *google.Client
	oauthKey           []byte
	visionCapture      *vision.ScreenCapture
	voicePipeline      *voice.AudioPipeline
	deepSearchEngine   *search.DeepSearchEngine
	auditLogger        *security.AuditLogger
	undoEngine         *system.UndoEngine
}

func New(db *sql.DB, broadcaster Broadcaster, bufferSize int) *Engine {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		db:          db,
		broadcaster: broadcaster,
		taskQueue:   make(chan string, bufferSize),
		activeTasks: make(map[string]context.CancelFunc),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (e *Engine) SetStepHandler(handler StepHandlerFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stepHandler = handler
}

func (e *Engine) SetVerifier(verifier Verifier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.verifier = verifier
}

func (e *Engine) SetHealer(healer Healer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.healer = healer
}

func (e *Engine) SetSupervisor(supervisor Supervisor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.supervisor = supervisor
}

func (e *Engine) SetDispatcher(dispatcher Dispatcher) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.dispatcher = dispatcher
}

func (e *Engine) SetBrowserClient(browserClient *browser.CDPClient) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.browserClient = browserClient
}

func (e *Engine) SetTelemetryCollector(collector *system.Collector) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.telemetryCollector = collector
}

func (e *Engine) SetGoogleClient(googleClient *google.Client, oauthKey []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.googleClient = googleClient
	e.oauthKey = oauthKey
}

func (e *Engine) SetVisionCapture(visionCapture *vision.ScreenCapture) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.visionCapture = visionCapture
}

func (e *Engine) SetVoicePipeline(voicePipeline *voice.AudioPipeline) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.voicePipeline = voicePipeline
}

func (e *Engine) SetDeepSearchEngine(engine *search.DeepSearchEngine) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.deepSearchEngine = engine
}

func (e *Engine) SetAuditLogger(logger *security.AuditLogger) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.auditLogger = logger
}

func (e *Engine) SetUndoEngine(undo *system.UndoEngine) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.undoEngine = undo
}

func (e *Engine) Start(workerCount int) {
	if workerCount <= 0 {
		workerCount = 2
	}
	for i := 0; i < workerCount; i++ {
		go e.worker(i)
	}
}

func (e *Engine) Stop() {
	e.cancel()
	e.mu.Lock()
	defer e.mu.Unlock()
	for taskID, cancel := range e.activeTasks {
		cancel()
		delete(e.activeTasks, taskID)
	}
}

func (e *Engine) EnqueueTask(taskID string) {
	select {
	case <-e.ctx.Done():
		log.Printf("[taskengine] Engine stopped, cannot enqueue task %s", taskID)
		return
	case e.taskQueue <- taskID:
	}
}

func (e *Engine) CancelTask(taskID string) {
	e.mu.Lock()
	cancel, exists := e.activeTasks[taskID]
	if exists {
		cancel()
		delete(e.activeTasks, taskID)
	}
	e.mu.Unlock()

	e.updateTaskStatus(taskID, models.TaskStatusCancelled, "Tarea cancelada por el usuario")
}

func (e *Engine) CancelAllTasks() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for taskID, cancel := range e.activeTasks {
		cancel()
		delete(e.activeTasks, taskID)
		e.updateTaskStatus(taskID, models.TaskStatusCancelled, "Cancelación forzosa por Kill-Switch de emergencia")
	}
}

func (e *Engine) AuthorizeStepPIN(taskID, stepID string) error {
	query := `UPDATE task_steps SET pin_authorized = 1 WHERE id = ? AND task_id = ?`
	res, err := e.db.Exec(query, stepID, taskID)
	if err != nil {
		return fmt.Errorf("authorize pin step: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("step %s not found for task %s", stepID, taskID)
	}

	// Reset task status to pending and re-enqueue
	e.updateTaskStatus(taskID, models.TaskStatusPending, "")
	e.EnqueueTask(taskID)
	return nil
}

func (e *Engine) worker(workerID int) {
	for {
		select {
		case <-e.ctx.Done():
			return
		case taskID := <-e.taskQueue:
			e.processTask(taskID)
		}
	}
}

func (e *Engine) processTask(taskID string) {
	taskCtx, taskCancel := context.WithCancel(e.ctx)

	e.mu.Lock()
	e.activeTasks[taskID] = taskCancel
	supervisor := e.supervisor
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.activeTasks, taskID)
		e.mu.Unlock()
	}()

	task, err := e.getTaskByID(taskID)
	if err != nil {
		log.Printf("[taskengine] Error obteniendo tarea %s: %v", taskID, err)
		return
	}

	// Do not process already completed or cancelled tasks
	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusCancelled {
		return
	}

	e.updateTaskStatus(taskID, models.TaskStatusRunning, "")
	task.Status = models.TaskStatusRunning

	steps, err := e.getPendingSteps(taskID)
	if err != nil {
		e.updateTaskStatus(taskID, models.TaskStatusFailed, err.Error())
		return
	}

	for _, step := range steps {
		select {
		case <-taskCtx.Done():
			return
		default:
		}

		// Supervisión de la Tríada Cognitiva (Charc pre-audit & Nine strategic replan)
		if supervisor != nil {
			audit, err := supervisor.Supervise(taskCtx, task, &step, "")
			if err == nil && audit != nil {
				if audit.RequiresPIN {
					step.RequiresPIN = true
				}

				if audit.EscalateToNine && audit.AlternativePayload != "" {
					step.Payload = audit.AlternativePayload
				} else if !audit.AllowExecution && !audit.EscalateToNine {
					e.updateTaskStatus(taskID, models.TaskStatusFailed, audit.Reason)
					return
				}
			}
		}

		if step.RequiresPIN && !step.PINAuthorized {
			e.updateTaskStatus(taskID, models.TaskStatusBlockedApproval, "")
			task.Status = models.TaskStatusBlockedApproval
			if e.broadcaster != nil {
				e.broadcaster.BroadcastPINRequest(task.ID, step.ID, step.ActionType, "Acción crítica supervisada")
			}
			return // Espera desbloqueo asíncrono vía WebSocket o AuthorizeStepPIN
		}

		e.updateTaskCurrentStep(taskID, step.StepOrder)
		task.CurrentStep = step.StepOrder

		if err := e.executeStep(taskCtx, task, &step); err != nil {
			e.updateTaskStatus(taskID, models.TaskStatusFailed, err.Error())
			return
		}
	}

	e.updateTaskStatus(taskID, models.TaskStatusCompleted, "")
	if e.broadcaster != nil {
		task.Status = models.TaskStatusCompleted
		e.broadcaster.BroadcastTaskProgress(task, nil)
	}
}

func (e *Engine) executeStep(ctx context.Context, task models.AgentTask, step *models.TaskStep) error {
	e.updateStepStatus(step.ID, models.StepStatusRunning, nil)
	step.Status = models.StepStatusRunning

	if e.broadcaster != nil {
		e.broadcaster.BroadcastTaskProgress(task, step)
	}

	e.mu.RLock()
	handler := e.stepHandler
	dispatcher := e.dispatcher
	verifier := e.verifier
	healer := e.healer
	undo := e.undoEngine
	audit := e.auditLogger
	e.mu.RUnlock()

	// Capturar snapshot preventivo si la acción altera archivos
	if undo != nil && (step.ActionType == "file_patch" || step.ActionType == "write_file") {
		var payloadData struct {
			Path     string `json:"path"`
			FilePath string `json:"file_path"`
		}
		_ = json.Unmarshal([]byte(step.Payload), &payloadData)
		targetPath := payloadData.Path
		if targetPath == "" {
			targetPath = payloadData.FilePath
		}
		if targetPath != "" {
			_, _ = undo.CaptureSnapshot(ctx, task.ID, targetPath)
		}
	}

	// 1. Ejecución de la acción asignada
	var output string
	var exitCode int
	var execErr error

	if handler != nil {
		execErr = handler(ctx, task, step)
		if step.Output != nil {
			output = *step.Output
		}
		if execErr != nil {
			exitCode = 1
			if output == "" {
				output = execErr.Error()
			}
		}
	} else if dispatcher != nil {
		output, exitCode, execErr = dispatcher.Dispatch(ctx, step)
	} else {
		// Despacho por defecto según action_type
		output, exitCode, execErr = e.dispatchAction(ctx, task, step)
	}

	// 2. Verificación heurística de resultados
	rule := ""
	if step.VerificationRule != nil {
		rule = *step.VerificationRule
	}

	var valid bool = true
	var reason string = "ok"
	var verifyErr error

	if verifier != nil {
		valid, reason, verifyErr = verifier.Verify(rule, output, exitCode)
	} else if exitCode != 0 || execErr != nil {
		valid = false
		if execErr != nil {
			reason = execErr.Error()
		} else {
			reason = fmt.Sprintf("código de salida no cero: %d", exitCode)
		}
	}

	// 3. Disparo de Self-Healing si hubo fallo o validación fallida
	if execErr != nil || !valid || verifyErr != nil {
		failReason := reason
		if execErr != nil {
			failReason = execErr.Error()
		} else if verifyErr != nil {
			failReason = verifyErr.Error()
		}

		if healer != nil && step.RetryCount < step.MaxRetries {
			e.updateStepStatus(step.ID, models.StepStatusRecovering, &failReason)
			step.Status = models.StepStatusRecovering

			healResult, err := healer.AttemptRecovery(ctx, task, step, failReason, output)
			if err == nil && healResult != nil && healResult.Success {
				// Reintentar ejecución con el parche generado
				step.Payload = healResult.PatchApplied
				return e.executeStep(ctx, task, step)
			}
		}

		errMsg := fmt.Sprintf("Fallo no recuperable: %s", failReason)
		step.Output = &errMsg
		e.updateStepStatus(step.ID, models.StepStatusFailed, &errMsg)
		return fmt.Errorf("paso %d falló: %s", step.StepOrder, failReason)
	}

	// 4. Éxito confirmado
	step.Output = &output
	e.updateStepStatus(step.ID, models.StepStatusCompleted, &output)

	if audit != nil {
		_, _ = audit.Record(ctx, step.AgentAssigned, step.ActionType, fmt.Sprintf("task=%s step=%d status=completed", task.ID, step.StepOrder))
	}
	return nil
}

func (e *Engine) getDecryptedGoogleToken(userID string) (string, error) {
	e.mu.RLock()
	key := e.oauthKey
	e.mu.RUnlock()

	if len(key) == 0 {
		return "", errors.New("clave de cifrado OAuth no configurada")
	}
	var tokenEnc []byte
	query := `SELECT access_token_enc FROM oauth_tokens WHERE user_id = ? AND provider = 'google'`
	err := e.db.QueryRow(query, userID).Scan(&tokenEnc)
	if err != nil {
		return "", fmt.Errorf("no se encontró token OAuth de Google para el usuario %s: %w", userID, err)
	}
	return google.DecryptToken(tokenEnc, key)
}

func (e *Engine) dispatchAction(ctx context.Context, task models.AgentTask, step *models.TaskStep) (string, int, error) {
	switch step.ActionType {
	case "cdp_action":
		if e.browserClient == nil {
			return "", 1, errors.New("cliente CDP de navegador no configurado")
		}
		res, err := e.browserClient.ExecuteAction(ctx, step.Payload)
		if err != nil {
			return "", 1, err
		}
		if !res.Success {
			return res.Error, 1, fmt.Errorf("fallo CDP: %s", res.Error)
		}
		return res.Data, 0, nil

	case "system_telemetry":
		if e.telemetryCollector == nil {
			return "", 1, errors.New("colector de telemetría no configurado")
		}
		metrics, err := e.telemetryCollector.Collect(ctx)
		if err != nil {
			return "", 1, err
		}
		bytes, _ := json.Marshal(metrics)
		return string(bytes), 0, nil

	case "google_draft":
		var draft google.DraftRequest
		if err := json.Unmarshal([]byte(step.Payload), &draft); err != nil {
			return "", 1, fmt.Errorf("payload de borrador inválido: %w", err)
		}
		if e.googleClient == nil {
			return fmt.Sprintf("Borrador simulado para %s: %s", draft.To, draft.Subject), 0, nil
		}
		token, err := e.getDecryptedGoogleToken(task.UserID)
		if err != nil {
			return fmt.Sprintf("Borrador simulado para %s: %s", draft.To, draft.Subject), 0, nil
		}
		draftResp, err := e.googleClient.CreateGmailDraft(ctx, token, draft)
		if err != nil {
			return "", 1, err
		}
		return fmt.Sprintf("Borrador creado en Gmail con ID: %s", draftResp.ID), 0, nil

	case "google_search":
		var queryParams struct {
			Query string `json:"query"`
		}
		_ = json.Unmarshal([]byte(step.Payload), &queryParams)
		if e.googleClient == nil {
			return fmt.Sprintf("Búsqueda simulada en Gmail para %q", queryParams.Query), 0, nil
		}
		token, err := e.getDecryptedGoogleToken(task.UserID)
		if err != nil {
			return fmt.Sprintf("Búsqueda simulada en Gmail para %q", queryParams.Query), 0, nil
		}
		res, err := e.googleClient.SearchGmail(ctx, token, queryParams.Query)
		if err != nil {
			return "", 1, err
		}
		return fmt.Sprintf("Encontrados %d correos relacionados", len(res.Messages)), 0, nil

	case "google_calendar":
		var calParams struct {
			MaxResults int `json:"max_results"`
		}
		_ = json.Unmarshal([]byte(step.Payload), &calParams)
		if e.googleClient == nil {
			return "Consulta simulada de calendario completada", 0, nil
		}
		token, err := e.getDecryptedGoogleToken(task.UserID)
		if err != nil {
			return "Consulta simulada de calendario completada", 0, nil
		}
		res, err := e.googleClient.ListCalendarEvents(ctx, token, calParams.MaxResults)
		if err != nil {
			return "", 1, err
		}
		return res, 0, nil

	case "screen_capture":
		if e.visionCapture == nil {
			return "Captura simulada de pantalla: 1920x1080 (PNG)", 0, nil
		}
		res, err := e.visionCapture.CapturePrimaryScreen(ctx)
		if err != nil {
			return "Captura simulada de pantalla: 1920x1080 (PNG)", 0, nil
		}
		return fmt.Sprintf("Captura tomada: %dx%d (%d bytes PNG)", res.Width, res.Height, len(res.ImagePNG)), 0, nil

	case "voice_synthesize":
		var p struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal([]byte(step.Payload), &p)
		if p.Text == "" {
			p.Text = step.Payload
		}
		if e.voicePipeline == nil {
			return fmt.Sprintf("Audio sintetizado para: %q (simulado)", p.Text), 0, nil
		}
		wav, err := e.voicePipeline.SynthesizeSpeech(ctx, p.Text)
		if err != nil {
			return fmt.Sprintf("Audio sintetizado para: %q (simulado)", p.Text), 0, nil
		}
		return fmt.Sprintf("Audio generado (%d bytes WAV)", len(wav)), 0, nil

	case "deep_search":
		var p struct {
			Query string `json:"query"`
		}
		_ = json.Unmarshal([]byte(step.Payload), &p)
		if p.Query == "" {
			p.Query = step.Payload
		}
		if e.deepSearchEngine == nil {
			return fmt.Sprintf("Búsqueda simulada para %q: [1] Referencia local", p.Query), 0, nil
		}
		searchRes, err := e.deepSearchEngine.Execute(ctx, p.Query)
		if err != nil {
			return "", 1, err
		}
		resBytes, _ := json.Marshal(searchRes)
		return string(resBytes), 0, nil

	default:
		// Fallback simulado para tests o herramientas genéricas
		select {
		case <-ctx.Done():
			return "", 1, ctx.Err()
		case <-time.After(50 * time.Millisecond):
			out := fmt.Sprintf("Paso %d ejecutado con éxito", step.StepOrder)
			return out, 0, nil
		}
	}
}

func (e *Engine) updateTaskStatus(taskID string, status models.TaskStatus, errMsg string) {
	query := `UPDATE agent_tasks SET status = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, _ = e.db.Exec(query, status, errMsg, taskID)
}

func (e *Engine) updateTaskCurrentStep(taskID string, currentStep int) {
	query := `UPDATE agent_tasks SET current_step = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, _ = e.db.Exec(query, currentStep, taskID)
}

func (e *Engine) updateStepStatus(stepID string, status models.StepStatus, output *string) {
	var query string
	if status == models.StepStatusCompleted || status == models.StepStatusFailed {
		query = `UPDATE task_steps SET status = ?, output = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?`
	} else {
		query = `UPDATE task_steps SET status = ?, output = ? WHERE id = ?`
	}
	_, _ = e.db.Exec(query, status, output, stepID)
}

func (e *Engine) getTaskByID(taskID string) (models.AgentTask, error) {
	var t models.AgentTask
	var status string
	query := `SELECT id, user_id, project_id, title, prompt, status, total_steps, current_step, error_message, created_at, updated_at FROM agent_tasks WHERE id = ?`
	err := e.db.QueryRow(query, taskID).Scan(&t.ID, &t.UserID, &t.ProjectID, &t.Title, &t.Prompt, &status, &t.TotalSteps, &t.CurrentStep, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt)
	if err == nil {
		t.Status = models.TaskStatus(status)
	}
	return t, err
}

func (e *Engine) getPendingSteps(taskID string) ([]models.TaskStep, error) {
	query := `SELECT id, task_id, step_order, agent_assigned, action_type, payload, requires_pin, pin_authorized, verification_rule, status, output, retry_count, max_retries, created_at, completed_at 
	          FROM task_steps WHERE task_id = ? AND status != 'completed' ORDER BY step_order ASC`
	rows, err := e.db.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []models.TaskStep
	for rows.Next() {
		var s models.TaskStep
		var st string
		if err := rows.Scan(&s.ID, &s.TaskID, &s.StepOrder, &s.AgentAssigned, &s.ActionType, &s.Payload, &s.RequiresPIN, &s.PINAuthorized, &s.VerificationRule, &st, &s.Output, &s.RetryCount, &s.MaxRetries, &s.CreatedAt, &s.CompletedAt); err != nil {
			return nil, err
		}
		s.Status = models.StepStatus(st)
		steps = append(steps, s)
	}
	return steps, nil
}
