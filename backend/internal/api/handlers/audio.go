package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/audio"
)

type WakeWordTestRequest struct {
	Text string `json:"text" binding:"required"`
}

// GetAudioDevicesHandler devuelve la lista de dispositivos de hardware de entrada y salida de audio
func GetAudioDevicesHandler(c *gin.Context) {
	devices, err := audio.GetAudioDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error enumerando dispositivos de sonido: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"total":   len(devices),
	})
}

// TestWakeWordHandler evalúa una frase transcrita y calcula la coincidencia de activación
func TestWakeWordHandler(c *gin.Context) {
	var req WakeWordTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'text' requerido"})
		return
	}

	match := audio.DetectWakeWord(req.Text)
	c.JSON(http.StatusOK, gin.H{
		"match": match,
	})
}

// GetAudioStatusHandler devuelve el estado general del subsistema de audio nativo
func GetAudioStatusHandler(c *gin.Context) {
	devices, _ := audio.GetAudioDevices()
	c.JSON(http.StatusOK, gin.H{
		"status":          "ready",
		"hardware_access": true,
		"engine":          "Windows WASAPI / WinMM Audio Engine",
		"input_devices":   len(devices),
		"vad_enabled":     true,
	})
}
