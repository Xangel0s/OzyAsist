package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const uploadDir = "data/uploads"

func init() {
	os.MkdirAll(uploadDir, 0755)
}

func UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	id := uuid.NewString() + ext
	dest := filepath.Join(uploadDir, id)

	out, err := os.Create(dest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save file: " + err.Error()})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "write file: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       id,
		"filename": header.Filename,
		"size":     header.Size,
		"url":      fmt.Sprintf("/api/files/%s", id),
	})
}

func GetFile(c *gin.Context) {
	id := c.Param("id")
	path := filepath.Join(uploadDir, id)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.File(path)
}
