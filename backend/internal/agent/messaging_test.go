package agent

import (
	"context"
	"testing"
)

func TestCleanPhoneNumber(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"+51 999 888 777", "51999888777"},
		{"+1 (555) 123-4567", "15551234567"},
		{"51999888777", "51999888777"},
		{"  +34 600-11-22-33  ", "34600112233"},
	}

	for _, c := range cases {
		actual := CleanPhoneNumber(c.input)
		if actual != c.expected {
			t.Errorf("CleanPhoneNumber(%q) = %q, want %q", c.input, actual, c.expected)
		}
	}
}

func TestOpenWhatsAppDraft_MemoryOnly(t *testing.T) {
	autoOpen := false
	params := WhatsAppDraftParams{
		Phone:    "+51999888777",
		Text:     "Hola! Te escribo para confirmar la reunión de mañana.",
		AutoOpen: &autoOpen,
	}

	res, err := OpenWhatsAppDraft(context.Background(), params)
	if err != nil {
		t.Fatalf("OpenWhatsAppDraft falló: %v", err)
	}

	if !res.Success {
		t.Errorf("Esperaba Success=true")
	}
	if res.Opened {
		t.Errorf("Esperaba Opened=false cuando auto_open=false")
	}
	if res.CleanPhone != "51999888777" {
		t.Errorf("CleanPhone incorrecto: %s", res.CleanPhone)
	}
}

func TestOpenWhatsAppDraft_Validation(t *testing.T) {
	_, err := OpenWhatsAppDraft(context.Background(), WhatsAppDraftParams{})
	if err == nil {
		t.Errorf("Esperaba error al pasar parámetros sin texto")
	}
}

func TestOpenTelegramDraft_MemoryOnly(t *testing.T) {
	autoOpen := false
	params := TelegramDraftParams{
		Recipient: "@juanperez",
		Text:      "Hola Juan, ¿pudiste revisar el reporte?",
		AutoOpen:  &autoOpen,
	}

	res, err := OpenTelegramDraft(context.Background(), params)
	if err != nil {
		t.Fatalf("OpenTelegramDraft falló: %v", err)
	}

	if !res.Success {
		t.Errorf("Esperaba Success=true")
	}
	if res.Opened {
		t.Errorf("Esperaba Opened=false cuando auto_open=false")
	}
	if res.Recipient != "@juanperez" {
		t.Errorf("Recipient incorrecto: %s", res.Recipient)
	}
}

func TestSendTelegramBotMessage_MissingCredentials(t *testing.T) {
	params := TelegramSendParams{
		Text: "Prueba sin token",
	}

	_, err := SendTelegramBotMessage(context.Background(), params)
	if err == nil {
		t.Errorf("Esperaba error explicativo cuando faltan credenciales de Telegram")
	}
}
