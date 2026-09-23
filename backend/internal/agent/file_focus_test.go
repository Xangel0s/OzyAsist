package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ozyassist/backend/internal/providers"
)

func TestFileInfo_Execution(t *testing.T) {
	// Crear un archivo temporal para la prueba
	tmpDir := t.TempDir()
	testFilePath := filepath.Join(tmpDir, "test_report.pdf")
	if err := os.WriteFile(testFilePath, []byte("OzyAssist Test Document Content"), 0644); err != nil {
		t.Fatalf("error creando archivo de prueba: %v", err)
	}

	inputBytes, _ := json.Marshal(map[string]string{"path": testFilePath})
	tc := providers.ToolCall{
		Name:  "os_file_info",
		Input: inputBytes,
	}

	out, ok := execOSFileInfo(context.Background(), tc)
	if !ok {
		t.Fatalf("execOSFileInfo falló inesperadamente: %s", out)
	}

	if !strings.Contains(out, "test_report.pdf") {
		t.Errorf("esperaba nombre de archivo en output, obtuve: %s", out)
	}
	if !strings.Contains(out, "Fecha de creación") {
		t.Errorf("esperaba fecha de creación en output, obtuve: %s", out)
	}
	if !strings.Contains(out, "Última modificación") {
		t.Errorf("esperaba fecha de última modificación en output, obtuve: %s", out)
	}
}

func TestAnaphora_FocusedFileResolution(t *testing.T) {
	tmpDir := t.TempDir()
	robloxFile := filepath.Join(tmpDir, "RobloxInfo.pdf")
	_ = os.WriteFile(robloxFile, []byte("Roblox Report"), 0644)

	// Registrar como archivo en foco
	SetFocusedFile(robloxFile, "roblox")

	// 1. Caso 'ábrelo'
	callsOpen := rescueDirectUserIntent("ábrelo", "")
	if len(callsOpen) != 1 || callsOpen[0].Name != "os_launch_app" {
		t.Fatalf("esperaba llamada a os_launch_app para 'ábrelo', obtuve: %+v", callsOpen)
	}
	var openArgs map[string]string
	_ = json.Unmarshal(callsOpen[0].Input, &openArgs)
	if openArgs["target"] != robloxFile {
		t.Errorf("esperaba que 'ábrelo' abriera '%s', pero abrió '%s'", robloxFile, openArgs["target"])
	}

	// 2. Caso 'comentame cuando fueron creados'
	callsCreated := rescueDirectUserIntent("comentame cuando fueron creados", "")
	if len(callsCreated) != 1 || callsCreated[0].Name != "os_file_info" {
		t.Fatalf("esperaba llamada a os_file_info para 'cuando fueron creados', obtuve: %+v", callsCreated)
	}
	var createdArgs map[string]string
	_ = json.Unmarshal(callsCreated[0].Input, &createdArgs)
	if createdArgs["path"] != robloxFile {
		t.Errorf("esperaba que 'cuando fueron creados' inspeccionara '%s', pero inspeccionó '%s'", robloxFile, createdArgs["path"])
	}

	// 3. Caso 'cuándo se creó'
	callsCreatedShort := rescueDirectUserIntent("cuándo se creó", "")
	if len(callsCreatedShort) != 1 || callsCreatedShort[0].Name != "os_file_info" {
		t.Fatalf("esperaba llamada a os_file_info para 'cuándo se creó', obtuve: %+v", callsCreatedShort)
	}
}

func TestLeakedToolJSON_DetectionAndRescue(t *testing.T) {
	rawLeaked := `{"name": "os_file_inspector", "arguments": {"path": "C:\\Users\\User\\Documents\\RobloxInfo.pdf"}}`

	if !isLeakedToolJSON(rawLeaked) {
		t.Errorf("isLeakedToolJSON debió detectar el JSON filtrado")
	}

	rescued := tryRescueLeakedJSON(rawLeaked)
	if len(rescued) != 1 {
		t.Fatalf("esperaba 1 llamada rescatada, obtuve %d", len(rescued))
	}
	if rescued[0].Name != "os_file_info" {
		t.Errorf("esperaba que 'os_file_inspector' fuera rescatado como 'os_file_info', obtuve: %s", rescued[0].Name)
	}

	// Verificar también que findRegisteredTool reconozca el alias
	canonical, ok := findRegisteredTool("os_file_inspector")
	if !ok || canonical != "os_file_info" {
		t.Errorf("findRegisteredTool falló al mapear 'os_file_inspector' a 'os_file_info', obtuve: %s, %v", canonical, ok)
	}
}
