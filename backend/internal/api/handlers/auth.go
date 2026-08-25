package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ozyassist/backend/internal/db"
)

type CreateProfileReq struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	AvatarColor string `json:"avatarColor"`
	Pin         string `json:"pin"`
	Role        string `json:"role"`
}

type LoginReq struct {
	UserID string `json:"userId"`
	Pin    string `json:"pin"`
}

type UpdatePinReq struct {
	UserID string `json:"userId"`
	OldPin string `json:"oldPin"`
	NewPin string `json:"newPin"`
}

// ListProfiles returns all registered profiles in SQLite.
func ListProfiles(c *gin.Context) {
	users, err := db.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

// CreateProfile creates a new local user profile with optional 6-digit PIN.
func CreateProfile(c *gin.Context) {
	var req CreateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nombre del perfil requerido"})
		return
	}

	pin := strings.TrimSpace(req.Pin)
	if pin != "" && len(pin) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el PIN de seguridad debe contener exactamente 6 dígitos"})
		return
	}

	user, err := db.CreateUserProfile(req.Name, req.Email, req.AvatarColor, pin, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	db.SetActiveUserID(user.ID)
	c.JSON(http.StatusCreated, user)
}

// Login verifies 6-digit PIN and activates user profile session.
func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.UserID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId es requerido"})
		return
	}

	valid, user, err := db.VerifyUserPin(req.UserID, req.Pin)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "perfil no encontrado"})
		return
	}
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "PIN de seguridad incorrecto"})
		return
	}

	db.SetActiveUserID(user.ID)
	c.JSON(http.StatusOK, gin.H{
		"status": "authenticated",
		"user":   user,
	})
}

// VerifyPin validates PIN for current or specified profile.
func VerifyPin(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.UserID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId es requerido"})
		return
	}

	valid, user, err := db.VerifyUserPin(req.UserID, req.Pin)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "perfil no encontrado"})
		return
	}
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "PIN incorrecto", "valid": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "user": user})
}

// UpdatePin sets, changes or removes the 6-digit security PIN.
func UpdatePin(c *gin.Context) {
	var req UpdatePinReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.UserID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId es requerido"})
		return
	}

	newPin := strings.TrimSpace(req.NewPin)
	if newPin != "" && len(newPin) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el nuevo PIN debe contener exactamente 6 dígitos"})
		return
	}

	if err := db.UpdateUserPin(req.UserID, req.OldPin, newPin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "pin_updated", "hasPin": newPin != ""})
}

// DeleteProfile deletes a profile from SQLite.
func DeleteProfile(c *gin.Context) {
	id := c.Param("id")
	if err := db.DeleteUser(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "profile_deleted"})
}

// Register handler
func Register(c *gin.Context) {
	CreateProfile(c)
}

// UpdateProfile updates markdown notes / instructions for the user profile.
func UpdateProfile(c *gin.Context) {
	var req struct {
		ProfileMd string `json:"profileMd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
		return
	}
	if err := db.UpdateUserProfile(db.DefaultUserID(), req.ProfileMd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}
