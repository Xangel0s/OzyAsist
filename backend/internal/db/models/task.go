package models

import "time"

type TaskStatus string

const (
	TaskStatusPending         TaskStatus = "pending"
	TaskStatusPlanning        TaskStatus = "planning"
	TaskStatusRunning         TaskStatus = "running"
	TaskStatusBlockedApproval TaskStatus = "blocked_approval"
	TaskStatusCompleted       TaskStatus = "completed"
	TaskStatusFailed          TaskStatus = "failed"
	TaskStatusCancelled       TaskStatus = "cancelled"
)

type StepStatus string

const (
	StepStatusPending    StepStatus = "pending"
	StepStatusRunning    StepStatus = "running"
	StepStatusVerifying  StepStatus = "verifying"
	StepStatusRecovering StepStatus = "recovering"
	StepStatusCompleted  StepStatus = "completed"
	StepStatusFailed     StepStatus = "failed"
)

type AgentTask struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	ProjectID    *string    `json:"project_id,omitempty"`
	Title        string     `json:"title"`
	Prompt       string     `json:"prompt"`
	Status       TaskStatus `json:"status"`
	TotalSteps   int        `json:"total_steps"`
	CurrentStep  int        `json:"current_step"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Steps        []TaskStep `json:"steps,omitempty"`
}

type TaskStep struct {
	ID               string     `json:"id"`
	TaskID           string     `json:"task_id"`
	StepOrder        int        `json:"step_order"`
	AgentAssigned    string     `json:"agent_assigned"`
	ActionType       string     `json:"action_type"`
	Payload          string     `json:"payload"`
	RequiresPIN      bool       `json:"requires_pin"`
	PINAuthorized    bool       `json:"pin_authorized"`
	VerificationRule *string    `json:"verification_rule,omitempty"`
	Status           StepStatus `json:"status"`
	Output           *string    `json:"output,omitempty"`
	RetryCount       int        `json:"retry_count"`
	MaxRetries       int        `json:"max_retries"`
	CreatedAt        time.Time  `json:"created_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

type AgentAction struct {
	ID                   string    `json:"id"`
	TaskID               string    `json:"taskId"`
	ActionType           string    `json:"actionType"`
	Target               string    `json:"target"`
	DetailsJSON          string    `json:"details,omitempty"`
	RequiresConfirmation bool      `json:"requiresConfirmation"`
	ConfirmedByUser      bool      `json:"confirmedByUser"`
	Result               string    `json:"result,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
}

