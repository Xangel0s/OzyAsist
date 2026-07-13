package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
)

func CreateAndExecuteTask(provider providers.Provider, projectID, chatID, userID, goal, permissionLevel string, stepCallback func(StepResult)) (string, []StepResult, error) {
	planner := NewPlanner(provider)

	var contextInfo string
	if projectID != "" {
		project, err := db.GetProject(projectID)
		if err == nil {
			if project.InstructionsMd != "" {
				contextInfo = "Instrucciones del proyecto:\n" + project.InstructionsMd + "\n\n"
			}
		}
	}

	if projectID != "" {
		graphCtx := memory.BuildGraphContext(projectID, goal)
		if graphCtx != "" {
			contextInfo = "[Dependencias relevantes del proyecto]:\n" + graphCtx + "\n\n" + contextInfo
		}
	}
	plan, err := planner.CreatePlan(context.Background(), goal, contextInfo)
	if err != nil {
		return "", nil, fmt.Errorf("plan creation failed: %w", err)
	}

	var sandbox *Sandbox
	if projectID != "" {
		project, err := db.GetProject(projectID)
		if err == nil && project.RootPath != "" {
			sb, err := NewSandbox(project.RootPath)
			if err == nil {
				sandbox = sb
			}
		}
	}

	level := PermissionLevel(permissionLevel)
	if level == "" {
		level = Sandboxed
	}

	taskID := uuid.NewString()
	executor := &Executor{
		Provider:     provider,
		Sandbox:      sandbox,
		Level:        level,
		ChatID:       chatID,
		ProjectID:    projectID,
		UserID:       userID,
		TaskID:       taskID,
		StepCallback: stepCallback,
	}

	results, err := executor.ExecutePlan(context.Background(), plan)
	return taskID, results, err
}
