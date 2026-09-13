package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ozyassist/backend/internal/providers"
)

func TestOSRunCommand_SafeExecution(t *testing.T) {
	input, _ := json.Marshal(map[string]interface{}{
		"command": "Write-Output 'OzyAssist Terminal Active'",
	})
	tc := providers.ToolCall{
		ID:    "test_call_1",
		Name:  "os_run_command",
		Input: input,
	}

	out, ok := execOSRunCommand(context.Background(), tc)
	if !ok {
		t.Fatalf("esperaba ejecucion exitosa, error en salida: %s", out)
	}
	if !strings.Contains(out, "OzyAssist Terminal Active") {
		t.Errorf("la salida no contiene el texto esperado: %s", out)
	}
}

func TestOSRunCommand_BlocksDestructive(t *testing.T) {
	destructiveCommands := []string{
		"rm -rf /",
		"format c:",
		"git push --force origin main",
		"del /f /q /s *",
	}

	for _, cmd := range destructiveCommands {
		input, _ := json.Marshal(map[string]interface{}{
			"command": cmd,
		})
		tc := providers.ToolCall{
			ID:    "test_call_destruct",
			Name:  "os_run_command",
			Input: input,
		}

		out, ok := execOSRunCommand(context.Background(), tc)
		if ok {
			t.Errorf("comando destructivo '%s' no debio ejecutarse exitosamente", cmd)
		}
		if !strings.Contains(out, "COMANDO BLOQUEADO POR SEGURIDAD") {
			t.Errorf("se esperaba mensaje de bloqueo de seguridad para '%s', obtuvo: %s", cmd, out)
		}
	}
}