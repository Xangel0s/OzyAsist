package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestVoiceConsoleModel_InitAndUpdate(t *testing.T) {
	m := NewVoiceConsoleModel(nil, nil)

	if m.state != VoiceStateIdle {
		t.Fatalf("Estado inicial esperado VoiceStateIdle, obtenido: %v", m.state)
	}

	// 1. Probar tick de animación
	updatedM, cmd := m.Update(voiceTickMsg{})
	if cmd == nil {
		t.Errorf("Se esperaba comando de siguiente tick")
	}
	model := updatedM.(VoiceConsoleModel)
	if model.waveFrame != 1 {
		t.Errorf("Se esperaba waveFrame=1 tras tick, obtenido: %d", model.waveFrame)
	}

	// 2. Probar cambio de tamaño de ventana
	updatedM, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model = updatedM.(VoiceConsoleModel)
	if model.width != 100 || model.height != 30 {
		t.Errorf("Tamaño de ventana no actualizado correctamente: %dx%d", model.width, model.height)
	}

	// 3. Probar renderizado de vista
	viewStr := model.View()
	if viewStr == "" {
		t.Errorf("La vista no debería estar vacía")
	}

	// 4. Probar tecla Esc (Barge-in / Cancel)
	updatedM, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updatedM.(VoiceConsoleModel)
	if model.state != VoiceStateIdle {
		t.Errorf("Estado tras Esc debería ser VoiceStateIdle")
	}
}

func TestVoiceConsoleModel_StateTransitions(t *testing.T) {
	m := NewVoiceConsoleModel(nil, nil)

	// Simular evento de oración hablada
	updatedM, _ := m.Update(voiceAgentEventMsg{
		Type:    "voice:sentence",
		Content: "Hola, soy Ozy.",
	})
	model := updatedM.(VoiceConsoleModel)
	if model.state != VoiceStateSpeaking {
		t.Errorf("Se esperaba VoiceStateSpeaking, obtenido: %v", model.state)
	}
	if model.activeUtterance != "Hola, soy Ozy." {
		t.Errorf("Oración activa incorrecta: %s", model.activeUtterance)
	}

	// Simular cancelación con Esc
	updatedM, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updatedM.(VoiceConsoleModel)
	if model.state != VoiceStateIdle {
		t.Errorf("Se esperaba VoiceStateIdle tras cancelar, obtenido: %v", model.state)
	}
	if len(model.history) == 0 || model.history[len(model.history)-1].Speaker != "system" {
		t.Errorf("Se esperaba entrada en historial de cancelación por el sistema")
	}
}
