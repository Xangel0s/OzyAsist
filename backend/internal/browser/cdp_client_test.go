package browser

import (
	"context"
	"testing"
)

func TestCDPClient_InvalidPayload(t *testing.T) {
	client := NewCDPClient("http://localhost:9222")

	_, err := client.ExecuteAction(context.Background(), "invalid json")
	if err == nil {
		t.Fatal("Esperaba error con payload JSON inválido")
	}
}

func TestCDPClient_UnsupportedAction(t *testing.T) {
	client := NewCDPClient("http://localhost:9222")

	_, err := client.ExecuteAction(context.Background(), `{"action":"unsupported_action"}`)
	if err == nil {
		t.Fatal("Esperaba error con acción no soportada")
	}
}

func TestCDPClient_CheckRemotePortClosed(t *testing.T) {
	client := NewCDPClient("http://localhost:59999") // Puerto cerrado arbitrario

	isOpen := client.CheckRemotePort(context.Background())
	if isOpen {
		t.Fatal("Esperaba que el puerto cerrado retornara false")
	}
}
