package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/skills"
)

func ListSkills(c *gin.Context) {
	userID := db.DefaultUserID()
	skillList, err := db.ListSkills(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if skillList == nil {
		skillList = []models.Skill{}
	}
	c.JSON(http.StatusOK, skillList)
}

func CreateSkill(c *gin.Context) {
	var req struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		TriggerPattern string `json:"triggerPattern"`
		ExecutionType  string `json:"executionType"`
		Config         string `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	skill := &models.Skill{
		ID:             uuid.NewString(),
		UserID:         db.DefaultUserID(),
		Name:           req.Name,
		Description:    req.Description,
		TriggerPattern: req.TriggerPattern,
		ExecutionType:  req.ExecutionType,
		ConfigJSON:     req.Config,
		CreatedAt:      time.Now(),
	}

	// Default execution type
	if skill.ExecutionType == "" {
		skill.ExecutionType = "prompt_template"
	}

	if err := db.CreateSkill(skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, skill)
}

func DeleteSkill(c *gin.Context) {
	id := c.Param("id")
	if err := db.DeleteSkill(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func ExecuteSkill(c *gin.Context) {
	var req struct {
		SkillID string `json:"skillId"`
		Input   string `json:"input"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SkillID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "skillId is required"})
		return
	}

	skill, err := db.GetSkill(req.SkillID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
		return
	}

	executor := skills.NewExecutor()
	result, err := executor.Execute(skill, req.Input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
