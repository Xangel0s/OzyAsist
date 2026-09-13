package handlers

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/security"
	"github.com/ozyassist/backend/internal/system"
)

var (
	globalAuditLogger *security.AuditLogger
	globalUndoEngine  *system.UndoEngine
	globalWatchdog    *security.Watchdog
	securityMu        sync.RWMutex
)

func SetSecurityComponents(logger *security.AuditLogger, undo *system.UndoEngine, watchdog *security.Watchdog) {
	securityMu.Lock()
	defer securityMu.Unlock()
	globalAuditLogger = logger
	globalUndoEngine = undo
	globalWatchdog = watchdog
}

// GetAuditTrail retorna la cadena inmutable y su verificación de integridad
func GetAuditTrail(c *gin.Context) {
	securityMu.RLock()
	logger := globalAuditLogger
	securityMu.RUnlock()

	if logger == nil {
		c.JSON(http.StatusOK, gin.H{
			"is_valid":       true,
			"total_verified": 0,
			"logs":           []security.AuditEntry{},
		})
		return
	}

	valid, count, err := logger.VerifyIntegrity(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":          err.Error(),
			"is_valid":       false,
			"total_verified": count,
		})
		return
	}

	logs, err := logger.GetRecentLogs(c.Request.Context(), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"is_valid":       valid,
		"total_verified": count,
		"logs":           logs,
	})
}

// GetTaskSnapshots lista los snapshots de archivos creados para una tarea
func GetTaskSnapshots(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "taskId requerido"})
		return
	}

	securityMu.RLock()
	undo := globalUndoEngine
	securityMu.RUnlock()

	if undo == nil {
		c.JSON(http.StatusOK, gin.H{"snapshots": []system.ShadowSnapshot{}})
		return
	}

	snapshots, err := undo.ListSnapshots(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id":   taskID,
		"snapshots": snapshots,
	})
}

// UndoTask revierte todos los cambios de archivos realizados por una tarea
func UndoTask(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "taskId requerido"})
		return
	}

	securityMu.RLock()
	undo := globalUndoEngine
	logger := globalAuditLogger
	securityMu.RUnlock()

	if undo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Undo Engine no disponible"})
		return
	}

	restored, err := undo.RollbackTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error en rollback: %v", err)})
		return
	}

	if logger != nil {
		_, _ = logger.Record(c.Request.Context(), "user", "time_travel_undo", fmt.Sprintf("task=%s restauró %d archivos", taskID, len(restored)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"task_id":        taskID,
		"restored_files": restored,
		"message":        fmt.Sprintf("Se restauraron %d archivos con éxito.", len(restored)),
	})
}

// KillSuspectProcess termina un proceso malicioso detectado por PID
func KillSuspectProcess(c *gin.Context) {
	var req struct {
		PID int32 `json:"pid" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PID requerido"})
		return
	}

	securityMu.RLock()
	wd := globalWatchdog
	securityMu.RUnlock()

	if wd == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Watchdog no disponible"})
		return
	}

	if err := wd.KillProcess(req.PID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"pid":     req.PID,
		"message": fmt.Sprintf("Proceso PID %d terminado con éxito", req.PID),
	})
}
