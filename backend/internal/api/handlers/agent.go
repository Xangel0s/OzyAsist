package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/agent"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
)

func CreateTask(c *gin.Context) {
	var req struct {
		ChatID          string `json:"chatId"`
		ProjectID       string `json:"projectId"`
		Goal            string `json:"goal"`
		PermissionLevel string `json:"permissionLevel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Goal == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "goal is required"})
		return
	}

	if req.PermissionLevel == "" {
		req.PermissionLevel = "sandboxed"
	}

	prov, err := providers.Get("opencode")
	if err != nil {
		prov, err = providers.Get("openai")
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no provider available"})
			return
		}
	}

	userID := db.DefaultUserID()

	// Fallback: use project's permission_level if not explicitly overridden
	if req.ProjectID != "" {
		if project, err := db.GetProject(req.ProjectID); err == nil && project.PermissionLevel != "" {
			req.PermissionLevel = project.PermissionLevel
		}
	}

	// Build context from project Caps1 + project instructions
	var contextInfo string
	if req.ProjectID != "" {
		project, err := db.GetProject(req.ProjectID)
		if err == nil {
			if project.InstructionsMd != "" {
				contextInfo = "Instrucciones del proyecto:\n" + project.InstructionsMd + "\n\n"
			}
			if project.RootPath != "" {
				memory.GetCaps1().RegisterProject(req.ProjectID, project.RootPath)
				entries, err := memory.GetCaps1().GetContext(req.ProjectID)
				if err == nil {
					for _, e := range entries {
						contextInfo += "--- " + e.Path + " ---\n" + e.Content + "\n\n"
					}
				}
			}
		}
	}

	// Create plan and execute
	taskID, results, execErr := agent.CreateAndExecuteTask(prov, req.ProjectID, req.ChatID, userID, req.Goal, req.PermissionLevel, nil)
	if execErr != nil {
		// Return partial results
	}

	c.JSON(http.StatusOK, gin.H{
		"taskId":  taskID,
		"results": results,
	})
}

func GetTask(c *gin.Context) {
	id := c.Param("id")
	task, err := db.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

func CancelTask(c *gin.Context) {
	id := c.Param("id")
	if err := db.UpdateTaskStatus(id, "cancelled", "cancelled by user"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func ConfirmAction(c *gin.Context) {
	actionID := c.Param("id")
	if actionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action id is required"})
		return
	}
	if err := db.ConfirmAction(actionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "confirmed"})
}
