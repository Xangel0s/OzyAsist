package taskengine_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/taskengine"
	_ "modernc.org/sqlite"
)

type mockBroadcaster struct {
	mu           sync.Mutex
	progressList []models.AgentTask
	pinRequests  []string
}

func (m *mockBroadcaster) BroadcastTaskProgress(task models.AgentTask, step *models.TaskStep) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progressList = append(m.progressList, task)
}

func (m *mockBroadcaster) BroadcastPINRequest(taskID, stepID, action, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pinRequests = append(m.pinRequests, fmt.Sprintf("%s:%s:%s", taskID, stepID, action))
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL
	);
	INSERT INTO users (id, name) VALUES ('usr_default', 'Test User');

	CREATE TABLE IF NOT EXISTS agent_tasks (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		project_id TEXT,
		title TEXT NOT NULL,
		prompt TEXT NOT NULL,
		status TEXT NOT NULL,
		total_steps INTEGER DEFAULT 0,
		current_step INTEGER DEFAULT 0,
		error_message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS task_steps (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL,
		step_order INTEGER NOT NULL,
		agent_assigned TEXT NOT NULL DEFAULT 'ozy',
		action_type TEXT NOT NULL,
		payload TEXT NOT NULL,
		requires_pin BOOLEAN DEFAULT 0,
		pin_authorized BOOLEAN DEFAULT 0,
		verification_rule TEXT,
		status TEXT NOT NULL,
		output TEXT,
		retry_count INTEGER DEFAULT 0,
		max_retries INTEGER DEFAULT 3,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return db
}

func TestEngine_SuccessFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	broadcaster := &mockBroadcaster{}
	engine := taskengine.New(db, broadcaster, 10)
	engine.Start(2)
	defer engine.Stop()

	taskID := uuid.NewString()
	_, err := db.Exec(`INSERT INTO agent_tasks (id, user_id, title, prompt, status, total_steps, current_step) VALUES (?, 'usr_default', 'Test Task', 'Do something', 'pending', 2, 0)`, taskID)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	step1ID := uuid.NewString()
	step2ID := uuid.NewString()
	_, err = db.Exec(`INSERT INTO task_steps (id, task_id, step_order, action_type, payload, status) VALUES (?, ?, 1, 'shell_exec', 'ls', 'pending'), (?, ?, 2, 'file_patch', 'main.go', 'pending')`, step1ID, taskID, step2ID, taskID)
	if err != nil {
		t.Fatalf("insert steps: %v", err)
	}

	engine.EnqueueTask(taskID)

	// Wait for processing
	time.Sleep(400 * time.Millisecond)

	var status string
	var currentStep int
	err = db.QueryRow(`SELECT status, current_step FROM agent_tasks WHERE id = ?`, taskID).Scan(&status, &currentStep)
	if err != nil {
		t.Fatalf("query task: %v", err)
	}

	if status != string(models.TaskStatusCompleted) {
		t.Errorf("expected task status completed, got %s", status)
	}
	if currentStep != 2 {
		t.Errorf("expected current_step 2, got %d", currentStep)
	}
}

func TestEngine_PINBlockedFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	broadcaster := &mockBroadcaster{}
	engine := taskengine.New(db, broadcaster, 10)
	engine.Start(2)
	defer engine.Stop()

	taskID := uuid.NewString()
	_, err := db.Exec(`INSERT INTO agent_tasks (id, user_id, title, prompt, status, total_steps, current_step) VALUES (?, 'usr_default', 'Dangerous Task', 'Drop DB', 'pending', 1, 0)`, taskID)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	stepID := uuid.NewString()
	_, err = db.Exec(`INSERT INTO task_steps (id, task_id, step_order, action_type, payload, requires_pin, pin_authorized, status) VALUES (?, ?, 1, 'shell_exec', 'rm -rf /', 1, 0, 'pending')`, stepID, taskID)
	if err != nil {
		t.Fatalf("insert steps: %v", err)
	}

	engine.EnqueueTask(taskID)

	time.Sleep(200 * time.Millisecond)

	var status string
	err = db.QueryRow(`SELECT status FROM agent_tasks WHERE id = ?`, taskID).Scan(&status)
	if err != nil {
		t.Fatalf("query task: %v", err)
	}

	if status != string(models.TaskStatusBlockedApproval) {
		t.Errorf("expected task status blocked_approval, got %s", status)
	}

	// Authorize PIN and resume
	if err := engine.AuthorizeStepPIN(taskID, stepID); err != nil {
		t.Fatalf("authorize pin: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	err = db.QueryRow(`SELECT status FROM agent_tasks WHERE id = ?`, taskID).Scan(&status)
	if err != nil {
		t.Fatalf("query task after auth: %v", err)
	}

	if status != string(models.TaskStatusCompleted) {
		t.Errorf("expected task status completed after authorization, got %s", status)
	}
}

func TestEngine_StepFailure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	broadcaster := &mockBroadcaster{}
	engine := taskengine.New(db, broadcaster, 10)
	engine.SetStepHandler(func(ctx context.Context, task models.AgentTask, step *models.TaskStep) error {
		return errors.New("command failed with exit code 1")
	})
	engine.Start(2)
	defer engine.Stop()

	taskID := uuid.NewString()
	_, err := db.Exec(`INSERT INTO agent_tasks (id, user_id, title, prompt, status, total_steps, current_step) VALUES (?, 'usr_default', 'Failing Task', 'Run error', 'pending', 1, 0)`, taskID)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	stepID := uuid.NewString()
	_, err = db.Exec(`INSERT INTO task_steps (id, task_id, step_order, action_type, payload, status) VALUES (?, ?, 1, 'shell_exec', 'exit 1', 'pending')`, stepID, taskID)
	if err != nil {
		t.Fatalf("insert steps: %v", err)
	}

	engine.EnqueueTask(taskID)

	time.Sleep(200 * time.Millisecond)

	var status string
	var errMsg sql.NullString
	err = db.QueryRow(`SELECT status, error_message FROM agent_tasks WHERE id = ?`, taskID).Scan(&status, &errMsg)
	if err != nil {
		t.Fatalf("query task: %v", err)
	}

	if status != string(models.TaskStatusFailed) {
		t.Errorf("expected status failed, got %s", status)
	}
	if !errMsg.Valid || errMsg.String == "" {
		t.Errorf("expected error message to be recorded")
	}
}

type mockDispatcher struct {
	attempts map[string]int
}

func (m *mockDispatcher) Dispatch(ctx context.Context, step *models.TaskStep) (string, int, error) {
	m.attempts[step.ID]++
	if step.Payload == "npm run build" {
		return "vite: not found", 1, errors.New("command not found: vite")
	}
	return "Build OK: bundle generated", 0, nil
}

type mockVerifier struct{}

func (m *mockVerifier) Verify(rawRule string, output string, exitCode int) (bool, string, error) {
	if exitCode != 0 {
		return false, fmt.Sprintf("exit code %d", exitCode), nil
	}
	return true, "ok", nil
}

type mockEngineHealer struct{}

func (m *mockEngineHealer) AttemptRecovery(ctx context.Context, task models.AgentTask, step *models.TaskStep, stepError string, output string) (*taskengine.HealResult, error) {
	step.RetryCount++
	return &taskengine.HealResult{
		Success:      true,
		PatchApplied: "npm install && npm run build",
		NewOutput:    "Parche generado",
	}, nil
}

func TestEngine_SelfHealingFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	broadcaster := &mockBroadcaster{}
	engine := taskengine.New(db, broadcaster, 10)

	dispatcher := &mockDispatcher{attempts: make(map[string]int)}
	verifier := &mockVerifier{}
	healer := &mockEngineHealer{}

	engine.SetDispatcher(dispatcher)
	engine.SetVerifier(verifier)
	engine.SetHealer(healer)

	engine.Start(2)
	defer engine.Stop()

	taskID := uuid.NewString()
	_, err := db.Exec(`INSERT INTO agent_tasks (id, user_id, title, prompt, status, total_steps, current_step) VALUES (?, 'usr_default', 'Self Healing Task', 'Build app', 'pending', 1, 0)`, taskID)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	stepID := uuid.NewString()
	_, err = db.Exec(`INSERT INTO task_steps (id, task_id, step_order, action_type, payload, status, retry_count, max_retries) VALUES (?, ?, 1, 'shell_exec', 'npm run build', 'pending', 0, 3)`, stepID, taskID)
	if err != nil {
		t.Fatalf("insert steps: %v", err)
	}

	engine.EnqueueTask(taskID)

	time.Sleep(300 * time.Millisecond)

	var status string
	err = db.QueryRow(`SELECT status FROM agent_tasks WHERE id = ?`, taskID).Scan(&status)
	if err != nil {
		t.Fatalf("query task: %v", err)
	}

	if status != string(models.TaskStatusCompleted) {
		t.Errorf("expected task status completed after self healing, got %s", status)
	}

	var stepStatus string
	var stepOutput sql.NullString
	err = db.QueryRow(`SELECT status, output FROM task_steps WHERE id = ?`, stepID).Scan(&stepStatus, &stepOutput)
	if err != nil {
		t.Fatalf("query step: %v", err)
	}

	if stepStatus != string(models.StepStatusCompleted) {
		t.Errorf("expected step status completed, got %s", stepStatus)
	}
	if !stepOutput.Valid || stepOutput.String != "Build OK: bundle generated" {
		t.Errorf("unexpected step output: %v", stepOutput)
	}
}

