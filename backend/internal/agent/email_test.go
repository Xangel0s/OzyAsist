package agent

import (
	"context"
	"testing"
)

func TestEncodeMailParam(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"Hola mundo", "Hola%20mundo"},
		{"Cotización & Servicios", "Cotizaci%C3%B3n%20%26%20Servicios"},
		{"test@example.com", "test%40example.com"},
	}

	for _, c := range cases {
		actual := encodeMailParam(c.input)
		if actual != c.expected {
			t.Errorf("encodeMailParam(%q) = %q, want %q", c.input, actual, c.expected)
		}
	}
}

func TestOpenEmailDraft_MemoryOnly(t *testing.T) {
	autoOpen := false
	params := DraftEmailParams{
		To:       "cliente@empresa.com",
		Subject:  "Propuesta de Servicio",
		Body:     "Estimado cliente,\nAdjunto la propuesta solicitada.",
		AutoOpen: &autoOpen,
	}

	res, err := OpenEmailDraft(context.Background(), params)
	if err != nil {
		t.Fatalf("OpenEmailDraft falló: %v", err)
	}

	if !res.Success {
		t.Errorf("Esperaba Success=true")
	}
	if res.Opened {
		t.Errorf("Esperaba Opened=false cuando auto_open=false")
	}
	if res.To != "cliente@empresa.com" {
		t.Errorf("Destinatario incorrecto: %s", res.To)
	}
}

func TestOpenEmailDraft_Validation(t *testing.T) {
	_, err := OpenEmailDraft(context.Background(), DraftEmailParams{})
	if err == nil {
		t.Errorf("Esperaba error al pasar parámetros vacíos")
	}
}
