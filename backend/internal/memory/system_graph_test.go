package memory

import (
	"testing"
)

func TestSystemGraph_DefaultEngrams(t *testing.T) {
	g := GetSystemGraph()
	if g == nil {
		t.Fatal("SystemGraph instance is nil")
	}

	// 1. Probar resolución exacta de mute
	match, ok := g.ResolveIntent("apaga la bulla")
	if !ok || match == nil {
		t.Fatalf("Esperaba resolución para 'apaga la bulla'")
	}
	if match.Engram.ToolName != "os_audio_device" {
		t.Errorf("Herramienta incorrecta: esperaba os_audio_device, obtuvo %s", match.Engram.ToolName)
	}
	if !match.IsFastTrack {
		t.Errorf("Esperaba IsFastTrack = true para 'apaga la bulla'")
	}
	if match.Engram.Destructive {
		t.Errorf("Audio mute no debe ser destructivo")
	}

	// 2. Probar resolución con muletillas y sinónimos
	match2, ok2 := g.ResolveIntent("Oye ozy porfa ponme en silencio que tengo reunion")
	if !ok2 || match2 == nil {
		t.Fatalf("Esperaba resolución para frase coloquial con muletillas")
	}
	if match2.Engram.ID != "audio_mute" {
		t.Errorf("ID esperado: audio_mute, obtenido: %s", match2.Engram.ID)
	}

	// 3. Probar extracción dinámica de número (volumen)
	matchVol, okVol := g.ResolveIntent("baja el volumen al 30")
	if !okVol || matchVol == nil {
		t.Fatalf("Esperaba resolución para 'baja el volumen al 30'")
	}
	if matchVol.ExtractedArgs["action"] != "set_volume" {
		t.Errorf("Acción esperada: set_volume, obtuvo: %v", matchVol.ExtractedArgs["action"])
	}
	if level, ok := matchVol.ExtractedArgs["level"].(float64); !ok || level != 30.0 {
		t.Errorf("Nivel esperado: 30.0, obtuvo: %v", matchVol.ExtractedArgs["level"])
	}

	// 4. Probar comando de ventana (mostrar escritorio)
	matchDesk, okDesk := g.ResolveIntent("minimiza todo y muestra el escritorio")
	if !okDesk || matchDesk == nil {
		t.Fatalf("Esperaba resolución para 'minimiza todo'")
	}
	if matchDesk.Engram.ToolName != "os_tile_windows" {
		t.Errorf("Herramienta esperada: os_tile_windows, obtuvo %s", matchDesk.Engram.ToolName)
	}
	if matchDesk.ExtractedArgs["layout"] != "show_desktop" {
		t.Errorf("Layout esperado: show_desktop, obtuvo %v", matchDesk.ExtractedArgs["layout"])
	}

	// 5. Blast Radius Guard: Acciones destructivas NUNCA deben tener FastTrack
	matchKill, okKill := g.ResolveIntent("mata el proceso notepad")
	if !okKill || matchKill == nil {
		t.Fatalf("Esperaba resolución para 'mata el proceso notepad'")
	}
	if matchKill.IsFastTrack {
		t.Errorf("VIOLACIÓN DE BLAST RADIUS: 'mata el proceso' no debe tener FastTrack!")
	}
	if !matchKill.Engram.Destructive {
		t.Errorf("El engrama de kill debe estar marcado como Destructive = true")
	}
}

func TestSystemGraph_Rollback(t *testing.T) {
	g := GetSystemGraph()

	// Probar detección de intención de rollback
	if !g.IsRollbackQuery("oye deshazlo por favor") {
		t.Errorf("Esperaba que 'oye deshazlo por favor' fuera detectado como Rollback")
	}
	if !g.IsRollbackQuery("espera no, vuelve a ponerlo") {
		t.Errorf("Esperaba que 'espera no, vuelve a ponerlo' fuera detectado como Rollback")
	}
	if g.IsRollbackQuery("abre la calculadora") {
		t.Errorf("'abre la calculadora' no debe ser Rollback")
	}

	// Registrar un estado previo
	g.RecordRollback(RollbackState{
		EngramID:       "audio_mute",
		ToolName:       "os_audio_device",
		ActionTaken:    "mute",
		RevertToolName: "os_audio_device",
		RevertArgs:     map[string]any{"action": "mute", "mute": false},
		Description:    "Desilenciar audio",
	})

	popped, ok := g.PopRollback()
	if !ok || popped == nil {
		t.Fatalf("Esperaba recuperar el estado de rollback")
	}
	if popped.RevertToolName != "os_audio_device" {
		t.Errorf("Herramienta de reversión incorrecta: %s", popped.RevertToolName)
	}
	if popped.RevertArgs["mute"] != false {
		t.Errorf("Argumento de reversión incorrecto: %v", popped.RevertArgs["mute"])
	}

	// Segundo pop debe dar false
	_, okEmpty := g.PopRollback()
	if okEmpty {
		t.Errorf("Búfer de rollback debería estar vacío")
	}
}

func TestSystemGraph_CoOccurrence(t *testing.T) {
	g := GetSystemGraph()

	// Probar que os_take_screenshot arrastra os_analyze_screen
	coTools := g.GetCoOccurringTools([]string{"os_take_screenshot"})
	foundAnalyze := false
	for _, tName := range coTools {
		if tName == "os_analyze_screen" {
			foundAnalyze = true
			break
		}
	}
	if !foundAnalyze {
		t.Errorf("Esperaba que os_take_screenshot co-ocurriera con os_analyze_screen, obtuvo: %v", coTools)
	}
}

func TestSystemGraph_LearnAndPromote(t *testing.T) {
	g := GetSystemGraph()

	engram, err := g.LearnEngram("modo cine", "os_power_profile", map[string]any{"action": "set_brightness", "brightness": 20}, "custom_cinema_mode")
	if err != nil {
		t.Fatalf("Error aprendiendo engrama: %v", err)
	}
	if engram.Maturity != "candidate" {
		t.Errorf("Maturidad esperada: candidate, obtuvo: %s", engram.Maturity)
	}
	if engram.FastTrack {
		t.Errorf("Nuevo engrama no debe ser FastTrack inicialmente")
	}

	// Promoverlo tras 2 usos exitosos
	g.PromoteEngram(engram.ID)
	g.PromoteEngram(engram.ID)

	match, ok := g.ResolveIntent("activa el modo cine")
	if !ok || match == nil {
		t.Fatalf("Esperaba resolución para 'modo cine'")
	}
	if match.Engram.Maturity != "reflex" {
		t.Errorf("Maturidad tras promoción esperada: reflex, obtuvo: %s", match.Engram.Maturity)
	}
	if !match.IsFastTrack {
		t.Errorf("Esperaba IsFastTrack = true tras promoción")
	}
}

func TestSystemGraph_AntonymPolarityAndMetaGuard(t *testing.T) {
	g := GetSystemGraph()

	// 1. "cierra la calculadora" DEBE resolver a close_calc (os_close_window) y NUNCA a launch_calc
	matchClose, okClose := g.ResolveIntent("cierra la calculadora")
	if !okClose || matchClose == nil {
		t.Fatalf("Esperaba resolución para 'cierra la calculadora'")
	}
	if matchClose.Engram.ToolName != "os_close_window" {
		t.Errorf("Herramienta errónea para cerrar: esperaba os_close_window, obtuvo %s", matchClose.Engram.ToolName)
	}
	if matchClose.Engram.ID != "close_calc" {
		t.Errorf("Engrama erróneo: esperaba close_calc, obtuvo %s", matchClose.Engram.ID)
	}
	if !matchClose.IsFastTrack {
		t.Errorf("Esperaba FastTrack=true para 'cierra la calculadora'")
	}

	// 2. "abre la calculadora" DEBE resolver a launch_calc (os_launch_app) y NUNCA a close_calc
	matchOpen, okOpen := g.ResolveIntent("abre la calculadora")
	if !okOpen || matchOpen == nil {
		t.Fatalf("Esperaba resolución para 'abre la calculadora'")
	}
	if matchOpen.Engram.ToolName != "os_launch_app" {
		t.Errorf("Herramienta errónea para abrir: esperaba os_launch_app, obtuvo %s", matchOpen.Engram.ToolName)
	}
	if matchOpen.Engram.ID != "launch_calc" {
		t.Errorf("Engrama erróneo: esperaba launch_calc, obtuvo %s", matchOpen.Engram.ID)
	}

	// 3. Consulta meta-conversacional con la palabra "calculadora" NO DEBE activar Fast-Track
	metaQuery := "okay revisa el grafo al poner 'cerrar calculadora' el sistema automanda abrir pero ya esta abierta la app de calculadora puedes verificar ello y actualizarlo para evitar ese fast track erroneo?"
	_, okMeta := g.ResolveIntent(metaQuery)
	if okMeta {
		t.Errorf("Una consulta meta-conversacional de depuración NO debe activar Fast-Track!")
	}

	// 4. "revisa bien mi ultimo mensaje" NO DEBE activar Fast-Track
	_, okMeta2 := g.ResolveIntent("revisa bien mi ultimo mensaje")
	if okMeta2 {
		t.Errorf("'revisa bien mi ultimo mensaje' NO debe activar Fast-Track!")
	}

	// 5. "cerrar" debe resolver a close_window_active (os_close_window)
	matchCerrar, okCerrar := g.ResolveIntent("cerrar")
	if !okCerrar || matchCerrar == nil {
		t.Fatalf("Esperaba resolución para 'cerrar'")
	}
	if matchCerrar.Engram.ToolName != "os_close_window" {
		t.Errorf("Herramienta errónea para 'cerrar': esperaba os_close_window, obtuvo %s", matchCerrar.Engram.ToolName)
	}
}

