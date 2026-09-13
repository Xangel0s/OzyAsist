package agent

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestOSTakeScreenshot(t *testing.T) {
	out, err := OSTakeScreenshot(context.Background())
	if err != nil {
		t.Logf("OSTakeScreenshot retornó error (esperable en CI/entorno sin display): %v", err)
		return
	}

	if !strings.Contains(out, "CAPTURA DE PANTALLA EXITOSA") {
		t.Errorf("Salida inesperada de OSTakeScreenshot: %s", out)
	}

	// Limpiar screenshots de prueba
	_ = os.RemoveAll("data/screenshots")
}

func TestOSMouseClick_Validation(t *testing.T) {
	// Verificar que no entra en pánico con coordenadas válidas
	_, err := OSMouseClick(context.Background(), 100, 100, "left")
	if err != nil && !strings.Contains(err.Error(), "solo está disponible") {
		t.Errorf("Error inesperado en OSMouseClick: %v", err)
	}
}

func TestOSTypeText_Empty(t *testing.T) {
	out, err := OSTypeText(context.Background(), "", false)
	if err != nil {
		t.Errorf("Error inesperado con texto vacío: %v", err)
	}
	if out != "Ningún texto o tecla especificada" {
		t.Errorf("Esperaba 'Ningún texto o tecla especificada', obtenido: %s", out)
	}
}
