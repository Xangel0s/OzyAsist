package ocr

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractText runs OCR on an image file using Tesseract CLI.
// Returns the extracted text, or an error if Tesseract is not available.
func ExtractText(imagePath string) (string, error) {
	// First verify Tesseract is installed
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", fmt.Errorf("tesseract not found: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(imagePath))
	supported := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true,
		".bmp": true, ".tiff": true, ".tif": true,
	}
	if !supported[ext] {
		return "", fmt.Errorf("unsupported image format: %s", ext)
	}

	cmd := exec.Command("tesseract", imagePath, "stdout", "-l", "spa+eng")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract error: %w\nstderr: %s", err, stderr.String())
	}

	text := strings.TrimSpace(stdout.String())
	if text == "" {
		return "", fmt.Errorf("no text found in image")
	}
	return text, nil
}

// IsAvailable checks if Tesseract is installed on the system.
func IsAvailable() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}
