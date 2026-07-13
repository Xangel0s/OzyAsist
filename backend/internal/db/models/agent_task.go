package models

import "time"

type AgentTask struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"projectId,omitempty"`
	ChatID          string     `json:"chatId,omitempty"`
	Goal            string     `json:"goal"`
	PlanJSON        string     `json:"plan,omitempty"`
	Status          string     `json:"status"`
	PermissionLevel string     `json:"permissionLevel"`
	Summary         string     `json:"summary,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
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
