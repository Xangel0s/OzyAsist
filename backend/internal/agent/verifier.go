package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type VerificationType string

const (
	VerifyExitCode     VerificationType = "exit_code"
	VerifyFileExists   VerificationType = "file_exists"
	VerifyFileContains VerificationType = "file_contains"
	VerifyOutputRegex  VerificationType = "output_regex"
	VerifyHTTPStatus   VerificationType = "http_status"
)

type VerificationRule struct {
	Type         VerificationType `json:"type"`
	ExpectedCode int              `json:"expected_code,omitempty"`
	Path         string           `json:"path,omitempty"`
	Pattern      string           `json:"pattern,omitempty"`
	URL          string           `json:"url,omitempty"`
	MinSizeBytes int64            `json:"min_size_bytes,omitempty"`
}

type Verifier struct {
	httpClient *http.Client
}

func NewVerifier() *Verifier {
	return &Verifier{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// Verify evalúa si la salida o estado resultante satisface la regla configurada.
func (v *Verifier) Verify(rawRule string, output string, exitCode int) (bool, string, error) {
	if strings.TrimSpace(rawRule) == "" {
		// Sin regla explícita: éxito si el código de salida es 0
		if exitCode != 0 {
			return false, fmt.Sprintf("comando finalizó con código de salida no cero: %d", exitCode), nil
		}
		return true, "sin regla específica (exit code 0)", nil
	}

	var rule VerificationRule
	if err := json.Unmarshal([]byte(rawRule), &rule); err != nil {
		return false, "", fmt.Errorf("error deserializando verification_rule JSON: %w", err)
	}

	switch rule.Type {
	case VerifyExitCode:
		if exitCode != rule.ExpectedCode {
			return false, fmt.Sprintf("código de salida %d no coincide con el esperado %d", exitCode, rule.ExpectedCode), nil
		}
		return true, "código de salida verificado", nil

	case VerifyFileExists:
		info, err := os.Stat(rule.Path)
		if err != nil {
			if os.IsNotExist(err) {
				return false, fmt.Sprintf("archivo requerido no existe: %s", rule.Path), nil
			}
			return false, "", fmt.Errorf("error al verificar archivo: %w", err)
		}
		if rule.MinSizeBytes > 0 && info.Size() < rule.MinSizeBytes {
			return false, fmt.Sprintf("el archivo %s tiene tamaño %d bytes (mínimo requerido: %d)", rule.Path, info.Size(), rule.MinSizeBytes), nil
		}
		return true, "archivo verificado correctamente", nil

	case VerifyFileContains:
		content, err := os.ReadFile(rule.Path)
		if err != nil {
			return false, fmt.Sprintf("no se pudo leer archivo %s para verificación: %v", rule.Path, err), nil
		}
		if !strings.Contains(string(content), rule.Pattern) {
			return false, fmt.Sprintf("el archivo %s no contiene el patrón esperado: %q", rule.Path, rule.Pattern), nil
		}
		return true, "patrón encontrado en archivo", nil

	case VerifyOutputRegex:
		matched, err := regexp.MatchString(rule.Pattern, output)
		if err != nil {
			return false, "", fmt.Errorf("expresión regular inválida %q: %w", rule.Pattern, err)
		}
		if !matched {
			return false, fmt.Sprintf("la salida no coincide con la expresión regular: %s", rule.Pattern), nil
		}
		return true, "expresión regular validada en salida", nil

	case VerifyHTTPStatus:
		resp, err := v.httpClient.Get(rule.URL)
		if err != nil {
			return false, fmt.Sprintf("error consultando endpoint %s: %v", rule.URL, err), nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != rule.ExpectedCode {
			return false, fmt.Sprintf("endpoint %s devolvió status %d (esperado %d)", rule.URL, resp.StatusCode, rule.ExpectedCode), nil
		}
		return true, "status HTTP verificado", nil

	default:
		return false, "", fmt.Errorf("tipo de verificación desconocido: %s", rule.Type)
	}
}
