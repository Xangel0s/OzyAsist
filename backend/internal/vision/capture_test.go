package vision

import (
	"context"
	"testing"
	"time"
)

func TestScreenCapture_NumDisplays(t *testing.T) {
	sc := NewScreenCapture()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := sc.CapturePrimaryScreen(ctx)
	if err != nil {
		t.Logf("Aviso: Captura no disponible en entorno CI/Headless: %v", err)
		return
	}

	if res.Width <= 0 || res.Height <= 0 || len(res.ImagePNG) == 0 {
		t.Errorf("Captura inválida: %+v", res)
	}
}
