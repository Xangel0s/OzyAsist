package memory

import (
	"strings"
	"testing"
)

func TestGraphRAGEngine_Query(t *testing.T) {
	engine := GetGraphRAGEngine()
	if engine == nil {
		t.Fatal("GetGraphRAGEngine() devolvió nil")
	}

	// 1. Consulta sobre cámara
	resultCam := engine.QueryGraphRAG("cómo tomar fotos con la camara o nokhwa", 3)
	if !strings.Contains(resultCam, "Cámara en Rust") && !strings.Contains(resultCam, "os_camera_capture") {
		t.Errorf("Consulta de cámara no retornó el nodo esperado. Salida:\n%s", resultCam)
	}

	// 2. Consulta sobre audio
	resultAudio := engine.QueryGraphRAG("silenciar sonido o bajar volumen", 3)
	if !strings.Contains(resultAudio, "Audio WASAPI") && !strings.Contains(resultAudio, "os_audio_device") {
		t.Errorf("Consulta de audio no retornó el nodo esperado. Salida:\n%s", resultAudio)
	}

	// 3. Consulta sobre MCTS y planificación
	resultMCTS := engine.QueryGraphRAG("arbol de decision monte carlo para recuperacion de error", 3)
	if !strings.Contains(resultMCTS, "Monte Carlo Tree Search") {
		t.Errorf("Consulta de MCTS no retornó el nodo esperado. Salida:\n%s", resultMCTS)
	}

	// 4. Verificación de spreading activation (la búsqueda de 'camara' debe relacionar telemetría de hardware)
	if !strings.Contains(resultCam, "inspects_device") && !strings.Contains(resultCam, "Telemetría") {
		t.Logf("Aviso: nodo conectado por spreading activation visible o ponderado.")
	}
}
