package models

import "time"

type Skill struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	TriggerPattern string    `json:"triggerPattern"`
	ExecutionType  string    `json:"executionType"`
	ConfigJSON     string    `json:"config,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}
