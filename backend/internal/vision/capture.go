package vision

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"os/exec"
	"runtime"
	"time"

	"github.com/kbinani/screenshot"
)

type ScreenCapture struct{}

func NewScreenCapture() *ScreenCapture {
	return &ScreenCapture{}
}

type CaptureResult struct {
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	DisplayID  int       `json:"display_id"`
	ImagePNG   []byte    `json:"-"`
	CapturedAt time.Time `json:"captured_at"`
}

// CapturePrimaryScreen obtiene una fotografía del monitor principal
func (sc *ScreenCapture) CapturePrimaryScreen(ctx context.Context) (*CaptureResult, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return nil, fmt.Errorf("no se detectaron pantallas activas")
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("error capturando pantalla: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("error codificando captura en PNG: %w", err)
	}

	return &CaptureResult{
		Width:      bounds.Dx(),
		Height:     bounds.Dy(),
		DisplayID:  0,
		ImagePNG:   buf.Bytes(),
		CapturedAt: time.Now(),
	}, nil
}

// RunLocalOCR procesa la imagen usando un binario OCR local (ej. tesseract o CLI ONNX)
func RunLocalOCR(ctx context.Context, imgBytes []byte) (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "tesseract", "stdin", "stdout", "-l", "spa+eng")
	default:
		cmd = exec.CommandContext(ctx, "tesseract", "stdin", "stdout", "-l", "spa+eng")
	}

	cmd.Stdin = bytes.NewReader(imgBytes)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error ejecutando OCR local: %w", err)
	}
	return string(out), nil
}
