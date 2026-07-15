package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
)

func validateRootPath(path string) string {
	if path == "" {
		return ""
	}
	// Allow relative paths (browser mode — user types folder name)
	if !filepath.IsAbs(path) {
		return ""
	}
	if fi, err := os.Stat(path); err != nil {
		return "root_path no accesible: " + err.Error()
	} else if !fi.IsDir() {
		return "root_path no es un directorio"
	}
	return ""
}

func ListProjects(c *gin.Context) {
	projects, err := db.ListProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projects)
}

type projectReq struct {
	Name            string `json:"name"`
	RootPath        string `json:"root_path"`
	InstructionsMd  string `json:"instructions_md"`
	PermissionLevel string `json:"permission_level"`
}

func CreateProject(c *gin.Context) {
	var req projectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "name es requerido"})
		return
	}

	if msg := validateRootPath(req.RootPath); msg != "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": msg})
		return
	}

	project := &models.Project{
		ID:              uuid.NewString(),
		UserID:          db.DefaultUserID(),
		Name:            req.Name,
		RootPath:        req.RootPath,
		InstructionsMd:  req.InstructionsMd,
		PermissionLevel: req.PermissionLevel,
		AgentConsent:    "ask",
		CreatedAt:       time.Now(),
	}
	if project.PermissionLevel == "" {
		project.PermissionLevel = "sandboxed"
	}

	if err := db.CreateProject(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, project)
}

func GetProject(c *gin.Context) {
	project, err := db.GetProject(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "proyecto no encontrado"})
		return
	}
	c.JSON(http.StatusOK, project)
}

func UpdateProject(c *gin.Context) {
	id := c.Param("id")

	existing, err := db.GetProject(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "proyecto no encontrado"})
		return
	}

	var req projectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
		return
	}

	if req.Name == "" {
		req.Name = existing.Name
	}
	if req.RootPath == "" {
		req.RootPath = existing.RootPath
	}
	if req.InstructionsMd == "" {
		req.InstructionsMd = existing.InstructionsMd
	}

	if msg := validateRootPath(req.RootPath); msg != "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": msg})
		return
	}

	updated := &models.Project{
		Name:          req.Name,
		RootPath:      req.RootPath,
		InstructionsMd: req.InstructionsMd,
	}

	if err := db.UpdateProject(id, updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result, _ := db.GetProject(id)
	c.JSON(http.StatusOK, result)
}

func DeleteProject(c *gin.Context) {
	if err := db.DeleteProject(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "eliminado"})
}
