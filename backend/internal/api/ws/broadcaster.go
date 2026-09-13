package ws

import (
	"encoding/json"

	"github.com/ozyassist/backend/internal/db/models"
)

type WSBroadcaster struct{}

func NewWSBroadcaster() *WSBroadcaster {
	return &WSBroadcaster{}
}

func (b *WSBroadcaster) BroadcastTaskProgress(task models.AgentTask, step *models.TaskStep) {
	stepDesc := ""
	stepOrder := 0
	if step != nil {
		stepDesc = step.ActionType + ": " + step.Payload
		stepOrder = step.StepOrder
	}
	evt := map[string]any{
		"type": "task:progress",
		"data": map[string]any{
			"task_id":          task.ID,
			"title":            task.Title,
			"step_current":     stepOrder,
			"step_total":       task.TotalSteps,
			"step_description": stepDesc,
			"status":           string(task.Status),
		},
	}
	raw, err := json.Marshal(evt)
	if err == nil {
		Broadcast(raw)
	}
}

func (b *WSBroadcaster) BroadcastPINRequest(taskID, stepID, action, reason string) {
	evt := map[string]any{
		"type": "task:require_approval",
		"data": map[string]any{
			"task_id":    taskID,
			"step_id":    stepID,
			"action":     action,
			"risk_level": "high",
			"reason":     reason,
		},
	}
	raw, err := json.Marshal(evt)
	if err == nil {
		Broadcast(raw)
	}
}

func (b *WSBroadcaster) BroadcastJSON(event string, payload interface{}) {
	evt := map[string]any{
		"type": event,
		"data": payload,
	}
	raw, err := json.Marshal(evt)
	if err == nil {
		Broadcast(raw)
	}
}

