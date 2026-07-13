package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

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

type StepResult struct {
	StepID  int    `json:"stepId"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

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
			results = append(results, StepResult{
				StepID:  step.ID,
				Success: false,
				Error:   "dependency not met",
			})
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
			// Pausa — no marcar como failed, esperar confirmación humana
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
	log.Printf("Agent executing step %d: %s (%s)", step.ID, step.Description, step.ActionType)

	action := &models.AgentAction{
		ID:         uuid.NewString(),
		TaskID:     e.TaskID,
		ActionType: step.ActionType,
		Target:     step.Target,
		CreatedAt:  time.Now(),
	}

	// Único punto de autorización — ValidateAndAuthorize bloquea antes de cualquier ejecución
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
	case "browser_action":
		return e.execBrowserAction(ctx, auth, action)
	case "llm_reason":
		return e.execLLMReason(ctx, step, completed, action)
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
	action.RequiresConfirmation = false
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

func (e *Executor) execBrowserAction(ctx context.Context, auth *AuthorizedAction, action *models.AgentAction) StepResult {
	action.Result = "browser_action not implemented yet"
	db.CreateAgentAction(action)
	return StepResult{StepID: auth.Step.ID, Success: false, Error: action.Result}
}

func (e *Executor) execLLMReason(ctx context.Context, step PlanStep, completed map[int]string, action *models.AgentAction) StepResult {
	var contextStr string
	for depID, output := range completed {
		contextStr += fmt.Sprintf("Step %d output:\n%s\n\n", depID, output)
	}

	prompt := fmt.Sprintf("Contexto:\n%s\n\nTarea: %s\n\n%s", contextStr, step.Description, step.Input)

	chunkCh, err := e.Provider.StreamCompletion(ctx, []providers.Message{
		{Role: "user", Content: prompt},
	}, providers.CompletionOptions{Stream: false})
	if err != nil {
		action.Result = "LLM reason error: " + err.Error()
		db.CreateAgentAction(action)
		return StepResult{StepID: step.ID, Success: false, Error: action.Result}
	}

	var response string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			response += chunk.Content
		}
	}

	action.Result = response
	db.CreateAgentAction(action)
	return StepResult{StepID: step.ID, Success: true, Output: response}
}
