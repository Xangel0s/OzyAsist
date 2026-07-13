package agent

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
)

// LogAction registra una acción del agente en agent_actions.
func LogAction(taskID, actionType, target, details string, requiresConfirmation bool) *models.AgentAction {
	action := &models.AgentAction{
		ID:                  uuid.NewString(),
		TaskID:             taskID,
		ActionType:         actionType,
		Target:             target,
		RequiresConfirmation: requiresConfirmation,
		CreatedAt:          time.Now(),
	}

	if details != "" {
		detailsMap := map[string]string{"details": details}
		dJSON, _ := json.Marshal(detailsMap)
		action.DetailsJSON = string(dJSON)
	}

	if err := db.CreateAgentAction(action); err != nil {
		log.Printf("audit: CreateAgentAction error: %v", err)
	}
	return action
}

// ConfirmAction marca una acción como confirmada por el usuario.
func ConfirmAction(actionID string) error {
	return db.ConfirmAction(actionID)
}
