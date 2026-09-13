package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ozyassist/backend/internal/browser"
	"github.com/ozyassist/backend/internal/providers"
)

type DraftEmailParams struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	Cc       string `json:"cc,omitempty"`
	Bcc      string `json:"bcc,omitempty"`
	Client      string `json:"client,omitempty"`       // "auto", "desktop_app", "gmail", "outlook_web"
	FromAccount string `json:"from_account,omitempty"` // Perfil o correo emisor en Chrome (ej: "zastuto5@gmail.com", "Tu Chrome")
	AutoOpen    *bool  `json:"auto_open,omitempty"`
}

type DraftEmailResult struct {
	Success    bool   `json:"success"`
	ClientUsed string `json:"client_used"`
	Opened     bool   `json:"opened"`
	To         string `json:"to"`
	Subject    string `json:"subject"`
	Message    string `json:"message"`
}

// encodeMailParam codifica parámetros para URIs mailto reemplazando + con %20 para máxima compatibilidad
func encodeMailParam(s string) string {
	escaped := url.QueryEscape(s)
	return strings.ReplaceAll(escaped, "+", "%20")
}

// OpenEmailDraft genera el borrador y opcionalmente abre la ventana de composición en Windows
func OpenEmailDraft(ctx context.Context, params DraftEmailParams) (*DraftEmailResult, error) {
	if strings.TrimSpace(params.To) == "" && strings.TrimSpace(params.Subject) == "" {
		return nil, fmt.Errorf("debe especificar al menos un destinatario ('to') o un asunto ('subject')")
	}

	autoOpen := true
	if params.AutoOpen != nil {
		autoOpen = *params.AutoOpen
	}

	clientMode := strings.ToLower(strings.TrimSpace(params.Client))
	if clientMode == "" {
		clientMode = "auto"
	}

	result := &DraftEmailResult{
		Success: true,
		To:      params.To,
		Subject: params.Subject,
	}

	if !autoOpen {
		result.Opened = false
		result.ClientUsed = "none (solo borrador en memoria)"
		result.Message = "Borrador de correo generado correctamente sin abrir ventana."
		return result, nil
	}

	var targetURL string
	var clientName string

	switch clientMode {
	case "outlook_web":
		clientName = "Outlook Web"
		targetURL = fmt.Sprintf("https://outlook.live.com/mail/0/deeplink/compose?to=%s&subject=%s&body=%s",
			encodeMailParam(params.To),
			encodeMailParam(params.Subject),
			encodeMailParam(params.Body),
		)
		if params.Cc != "" {
			targetURL += "&cc=" + encodeMailParam(params.Cc)
		}
		if params.Bcc != "" {
			targetURL += "&bcc=" + encodeMailParam(params.Bcc)
		}

	case "desktop_app":
		clientName = "Cliente de Correo de Escritorio"
		var queryParts []string
		if params.Subject != "" {
			queryParts = append(queryParts, "subject="+encodeMailParam(params.Subject))
		}
		if params.Body != "" {
			queryParts = append(queryParts, "body="+encodeMailParam(params.Body))
		}
		if params.Cc != "" {
			queryParts = append(queryParts, "cc="+encodeMailParam(params.Cc))
		}
		if params.Bcc != "" {
			queryParts = append(queryParts, "bcc="+encodeMailParam(params.Bcc))
		}

		targetURL = "mailto:" + params.To
		if len(queryParts) > 0 {
			targetURL += "?" + strings.Join(queryParts, "&")
		}

	default: // "auto", "gmail", "" (Preferencia máxima por el navegador y perfil autenticado del usuario)
		profile := browser.FindMatchingProfile("chrome", params.FromAccount)
		authuser := ""
		if profile != nil && profile.Email != "" {
			authuser = profile.Email
		} else if strings.Contains(params.FromAccount, "@") {
			authuser = strings.TrimSpace(params.FromAccount)
		}

		clientName = "Gmail Web"
		if profile != nil {
			clientName = fmt.Sprintf("Gmail Web (Perfil Chrome: %s | %s)", profile.Name, profile.Email)
		}

		if authuser != "" {
			targetURL = fmt.Sprintf("https://mail.google.com/mail/u/?authuser=%s&view=cm&fs=1&to=%s&su=%s&body=%s",
				encodeMailParam(authuser),
				encodeMailParam(params.To),
				encodeMailParam(params.Subject),
				encodeMailParam(params.Body),
			)
		} else {
			targetURL = fmt.Sprintf("https://mail.google.com/mail/?view=cm&fs=1&to=%s&su=%s&body=%s",
				encodeMailParam(params.To),
				encodeMailParam(params.Subject),
				encodeMailParam(params.Body),
			)
		}
		if params.Cc != "" {
			targetURL += "&cc=" + encodeMailParam(params.Cc)
		}
		if params.Bcc != "" {
			targetURL += "&bcc=" + encodeMailParam(params.Bcc)
		}

		// Si encontramos el perfil y su ejecutable en Windows, abrir directamente con --profile-directory
		if profile != nil && profile.ExecPath != "" {
			if err := browser.LaunchWithProfile(ctx, profile, targetURL); err == nil {
				result.ClientUsed = clientName
				result.Opened = true
				result.Message = fmt.Sprintf("Ventana de composición abierta exitosamente en Chrome con la cuenta autenticada de %s.", profile.Email)
				return result, nil
			}
		}
	}

	result.ClientUsed = clientName

	// Lanzar URL en el sistema operativo
	if err := launchURLInOS(ctx, targetURL); err != nil {
		result.Opened = false
		result.Message = fmt.Sprintf("Borrador generado, pero ocurrió un error al intentar abrir la ventana de correo: %v", err)
		return result, nil
	}

	result.Opened = true
	result.Message = fmt.Sprintf("Ventana de composición abierta exitosamente con %s.", clientName)
	return result, nil
}

func launchURLInOS(ctx context.Context, target string) error {
	return LaunchURLInOS(ctx, target)
}

func execOSDraftEmail(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params DraftEmailParams
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_draft_email: %v", err), false
	}

	res, err := OpenEmailDraft(ctx, params)
	if err != nil {
		return fmt.Sprintf("Error generando borrador de correo: %v", err), false
	}

	var sb strings.Builder
	sb.WriteString("✉️ === BORRADOR DE CORREO GENERADO CON ÉXITO ===\n")
	sb.WriteString(fmt.Sprintf("• Destinatario: %s\n", res.To))
	if params.Cc != "" {
		sb.WriteString(fmt.Sprintf("• CC:           %s\n", params.Cc))
	}
	sb.WriteString(fmt.Sprintf("• Asunto:       %s\n", res.Subject))
	sb.WriteString(fmt.Sprintf("• Cliente:      %s\n", res.ClientUsed))
	if res.Opened {
		sb.WriteString("• Estado:       ✓ Ventana de composición abierta en pantalla para revisión y envío.\n")
	} else {
		sb.WriteString("• Estado:       Borrador generado en memoria.\n")
	}
	sb.WriteString("\n--- Contenido del Correo ---\n")
	sb.WriteString(params.Body)
	sb.WriteString("\n----------------------------\n")
	sb.WriteString(fmt.Sprintf("💡 Nota: %s", res.Message))

	return sb.String(), true
}
