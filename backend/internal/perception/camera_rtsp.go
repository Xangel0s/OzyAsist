package perception

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type RTSPStreamConfig struct {
	URL         string        `json:"url"`
	Timeout     time.Duration `json:"timeout"`
	FrameWidth  int           `json:"frame_width"`
	FrameHeight int           `json:"frame_height"`
}

type RTSPClient struct {
	config RTSPStreamConfig
}

func NewRTSPClient(config RTSPStreamConfig) *RTSPClient {
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Second
	}
	return &RTSPClient{config: config}
}

// CaptureFrame extrae un fotograma instantáneo del stream RTSP en formato JPEG
func (r *RTSPClient) CaptureFrame(ctx context.Context) ([]byte, error) {
	if r.config.URL == "" {
		return nil, fmt.Errorf("URL de stream RTSP no configurada")
	}

	execCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	// Comando ffmpeg optimizado para captura ultra-rápida de 1 solo frame sin transcodificación pesada
	args := []string{
		"-y",
		"-rtsp_transport", "tcp",
		"-i", r.config.URL,
		"-vframes", "1",
		"-f", "image2",
		"-q:v", "2",
		"pipe:1",
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(execCtx, "ffmpeg", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("captura de fotograma RTSP fallida: %v (%s)", err, stderr.String())
	}

	frameData := stdout.Bytes()
	if len(frameData) == 0 {
		return nil, fmt.Errorf("stream RTSP no retornó datos de imagen válidos")
	}

	return frameData, nil
}

// CaptureFrameToFile guarda el fotograma directamente en una ruta de archivo local
func (r *RTSPClient) CaptureFrameToFile(ctx context.Context, outputPath string) error {
	data, err := r.CaptureFrame(ctx)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}
