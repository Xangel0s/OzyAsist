package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ozyassist/backend/internal/browser"
	"github.com/ozyassist/backend/internal/providers"
)

type WhatsAppDraftParams struct {
	Phone    string `json:"phone,omitempty"`     // Número telefónico (ej: "987654321", "+51987654321")
	Message  string `json:"message,omitempty"`   // Texto del mensaje a enviar
	Text     string `json:"text,omitempty"`      // Alias de message
	AutoOpen *bool  `json:"auto_open,omitempty"` // Si debe abrir automáticamente WhatsApp Web (por defecto true)
}

// DraftWhatsAppParams es un alias de WhatsAppDraftParams para retrocompatibilidad
type DraftWhatsAppParams = WhatsAppDraftParams

type DraftWhatsAppResult struct {
	Success    bool   `json:"success"`
	Opened     bool   `json:"opened"`
	Phone      string `json:"phone"`
	CleanPhone string `json:"clean_phone"`
	Message    string `json:"message"`
	URL        string `json:"url"`
	Status     string `json:"status"`
}

type TelegramDraftParams struct {
	Recipient string `json:"recipient,omitempty"` // Usuario de Telegram (ej: "@zorro", "zorro") o teléfono
	Message   string `json:"message,omitempty"`   // Texto del mensaje a enviar
	Text      string `json:"text,omitempty"`      // Alias de message
	Client    string `json:"client,omitempty"`    // "web" (por defecto) o "app" (aplicación nativa)
	AutoOpen  *bool  `json:"auto_open,omitempty"` // Si debe abrir automáticamente Telegram (por defecto true)
}

// DraftTelegramParams es un alias de TelegramDraftParams para retrocompatibilidad
type DraftTelegramParams = TelegramDraftParams

type DraftTelegramResult struct {
	Success   bool   `json:"success"`
	Opened    bool   `json:"opened"`
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
	URL       string `json:"url"`
	Status    string `json:"status"`
}

type TelegramSendParams struct {
	BotToken string `json:"bot_token,omitempty"`
	ChatID   string `json:"chat_id,omitempty"`
	Text     string `json:"text"`
}

type TelegramSendResult struct {
	Success   bool   `json:"success"`
	MessageID int    `json:"message_id,omitempty"`
	ChatID    string `json:"chat_id"`
	Status    string `json:"status"`
}

// CleanPhoneNumber limpia y normaliza números telefónicos
func CleanPhoneNumber(phone string) string {
	cleaned := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")
	cleaned = strings.TrimPrefix(cleaned, "+")

	// Si son 9 dígitos y empieza con 9 (número celular en Perú), anteponer código país 51
	if len(cleaned) == 9 && strings.HasPrefix(cleaned, "9") {
		return "51" + cleaned
	}
	return cleaned
}

// NormalizePhoneNumber es alias para CleanPhoneNumber
func NormalizePhoneNumber(phone string) string {
	return CleanPhoneNumber(phone)
}

// LaunchURLInOS abre una URL usando el perfil predeterminado o navegador de sistema
func LaunchURLInOS(ctx context.Context, target string) error {
	return browser.LaunchWithProfile(ctx, nil, target)
}

// OpenWhatsAppDraft genera el enlace para WhatsApp Web y lo abre en Chrome
func OpenWhatsAppDraft(ctx context.Context, params WhatsAppDraftParams) (*DraftWhatsAppResult, error) {
	msg := strings.TrimSpace(params.Message)
	if msg == "" {
		msg = strings.TrimSpace(params.Text)
	}

	phone := strings.TrimSpace(params.Phone)
	if msg == "" && phone == "" {
		return nil, fmt.Errorf("debe proporcionar al menos un número de teléfono ('phone') o un mensaje ('message')")
	}

	normPhone := CleanPhoneNumber(phone)
	escapedText := url.QueryEscape(msg)

	var targetURL string
	if normPhone != "" {
		targetURL = fmt.Sprintf("https://web.whatsapp.com/send?phone=%s&text=%s", normPhone, escapedText)
	} else {
		targetURL = fmt.Sprintf("https://web.whatsapp.com/send?text=%s", escapedText)
	}

	autoOpen := true
	if params.AutoOpen != nil {
		autoOpen = *params.AutoOpen
	}

	result := &DraftWhatsAppResult{
		Success:    true,
		Phone:      normPhone,
		CleanPhone: normPhone,
		Message:    msg,
		URL:        targetURL,
	}

	if !autoOpen {
		result.Opened = false
		result.Status = "Borrador de WhatsApp generado correctamente (sin abrir ventana)."
		return result, nil
	}

	profile := browser.FindMatchingProfile("chrome", "")
	err := browser.LaunchWithProfile(ctx, profile, targetURL)
	if err != nil {
		result.Opened = false
		result.Status = fmt.Sprintf("Enlace generado, pero ocurrió un error al abrir el navegador: %v", err)
		return result, nil
	}

	result.Opened = true
	result.Status = "Ventana de WhatsApp Web abierta en Chrome con el chat y mensaje listos para enviar."
	return result, nil
}

// OpenTelegramDraft genera el enlace para Telegram y lo abre en Chrome o la app de escritorio
func OpenTelegramDraft(ctx context.Context, params TelegramDraftParams) (*DraftTelegramResult, error) {
	msg := strings.TrimSpace(params.Message)
	if msg == "" {
		msg = strings.TrimSpace(params.Text)
	}
	recipient := strings.TrimSpace(params.Recipient)
	recipientClean := strings.TrimPrefix(recipient, "@")

	if msg == "" && recipient == "" {
		return nil, fmt.Errorf("debe proporcionar al menos un destinatario ('recipient') o un mensaje ('message')")
	}

	clientMode := strings.ToLower(strings.TrimSpace(params.Client))
	if clientMode == "" {
		clientMode = "web"
	}

	escapedText := url.QueryEscape(msg)
	var targetURL string

	if clientMode == "app" {
		if recipientClean != "" {
			targetURL = fmt.Sprintf("tg://msg?text=%s&to=%s", escapedText, recipientClean)
		} else {
			targetURL = fmt.Sprintf("tg://msg?text=%s", escapedText)
		}
	} else {
		if recipientClean != "" {
			targetURL = fmt.Sprintf("https://t.me/%s?text=%s", recipientClean, escapedText)
		} else {
			targetURL = fmt.Sprintf("https://web.telegram.org/k/#?text=%s", escapedText)
		}
	}

	autoOpen := true
	if params.AutoOpen != nil {
		autoOpen = *params.AutoOpen
	}

	result := &DraftTelegramResult{
		Success:   true,
		Recipient: recipient,
		Message:   msg,
		URL:       targetURL,
	}

	if !autoOpen {
		result.Opened = false
		result.Status = "Borrador de Telegram generado correctamente (sin abrir ventana)."
		return result, nil
	}

	profile := browser.FindMatchingProfile("chrome", "")
	err := browser.LaunchWithProfile(ctx, profile, targetURL)
	if err != nil {
		result.Opened = false
		result.Status = fmt.Sprintf("Enlace generado, pero ocurrió un error al abrir Telegram: %v", err)
		return result, nil
	}

	result.Opened = true
	result.Status = "Ventana de Telegram abierta con el mensaje listo para enviar."
	return result, nil
}

// SendTelegramBotMessage envía un mensaje directamente vía API de Bot de Telegram
func SendTelegramBotMessage(ctx context.Context, params TelegramSendParams) (*TelegramSendResult, error) {
	token := strings.TrimSpace(params.BotToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	}
	chatID := strings.TrimSpace(params.ChatID)
	if chatID == "" {
		chatID = strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID"))
	}

	if token == "" || chatID == "" {
		return nil, fmt.Errorf("se requiere 'bot_token' y 'chat_id' (o configurar TELEGRAM_BOT_TOKEN y TELEGRAM_CHAT_ID en .env)")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       params.Text,
		"parse_mode": "Markdown",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al conectar con Telegram API: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var tgResp struct {
		Ok     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(body, &tgResp); err != nil {
		return nil, fmt.Errorf("respuesta inválida de Telegram: %s", string(body))
	}

	if !tgResp.Ok {
		return nil, fmt.Errorf("telegram API error: %s", tgResp.Description)
	}

	return &TelegramSendResult{
		Success:   true,
		MessageID: tgResp.Result.MessageID,
		ChatID:    chatID,
		Status:    "Mensaje enviado exitosamente vía Telegram Bot API.",
	}, nil
}

func execOSDraftWhatsApp(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params WhatsAppDraftParams
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_draft_whatsapp: %v", err), false
	}

	res, err := OpenWhatsAppDraft(ctx, params)
	if err != nil {
		return fmt.Sprintf("error al preparar borrador de WhatsApp: %v", err), false
	}

	var sb strings.Builder
	sb.WriteString("💬 === MENSAJE DE WHATSAPP PREPARADO ===\n")
	if res.Phone != "" {
		sb.WriteString(fmt.Sprintf("• Teléfono:  +%s\n", res.Phone))
	}
	sb.WriteString(fmt.Sprintf("• Mensaje:   %s\n", res.Message))
	sb.WriteString(fmt.Sprintf("• Estado:    %s\n", res.Status))
	sb.WriteString(fmt.Sprintf("• Enlace:    %s\n", res.URL))
	return sb.String(), true
}

func execOSDraftTelegram(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params TelegramDraftParams
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_draft_telegram: %v", err), false
	}

	res, err := OpenTelegramDraft(ctx, params)
	if err != nil {
		return fmt.Sprintf("error al preparar mensaje de Telegram: %v", err), false
	}

	var sb strings.Builder
	sb.WriteString("✈️ === MENSAJE DE TELEGRAM PREPARADO ===\n")
	if res.Recipient != "" {
		sb.WriteString(fmt.Sprintf("• Destino:   %s\n", res.Recipient))
	}
	sb.WriteString(fmt.Sprintf("• Mensaje:   %s\n", res.Message))
	sb.WriteString(fmt.Sprintf("• Estado:    %s\n", res.Status))
	sb.WriteString(fmt.Sprintf("• Enlace:    %s\n", res.URL))
	return sb.String(), true
}

func execTelegramSendMessage(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params TelegramSendParams
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para telegram_send_message: %v", err), false
	}

	res, err := SendTelegramBotMessage(ctx, params)
	if err != nil {
		return fmt.Sprintf("error enviando mensaje por bot de Telegram: %v", err), false
	}

	return fmt.Sprintf("✓ Mensaje enviado a Telegram chat %s (Message ID: %d)", res.ChatID, res.MessageID), true
}
