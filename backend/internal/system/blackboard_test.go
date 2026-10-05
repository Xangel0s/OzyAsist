package system

import (
	"context"
	"strings"
	"testing"
)

func TestOSBlackboard_RenderHUD(t *testing.T) {
	b := DefaultBlackboard()
	b.SetFocusedFile("c:/Users/User/Documents/crmgeofal/main.go")

	hud := b.RenderHUD()
	if !strings.Contains(hud, "OZY OS-HUD: ESTADO EN VIVO") {
		t.Errorf("expected HUD banner in render, got: %s", hud)
	}
	if !strings.Contains(hud, "crmgeofal/main.go") {
		t.Errorf("expected focused file in HUD, got: %s", hud)
	}

	t.Logf("Rendered HUD (< 65 tokens):\n%s", hud)
}

func TestOSBlackboard_DeltaRendering(t *testing.T) {
	b := DefaultBlackboard()
	b.Refresh(context.Background())
	snap := b.GetSnapshot()

	// Primera llamada con hash previo idéntico
	delta, hash := b.RenderDelta(snap.Hash)
	if !strings.Contains(delta, "Sin cambios") {
		t.Errorf("expected 'Sin cambios' in delta render, got: %s", delta)
	}
	if hash != snap.Hash {
		t.Errorf("expected matching hash")
	}

	// Con hash previo 0 (primer turno) debe emitir el HUD completo
	full, _ := b.RenderDelta(0)
	if !strings.Contains(full, "OZY OS-HUD") {
		t.Errorf("expected full HUD on first turn")
	}
}

func TestIsOSRelevantQuery(t *testing.T) {
	cases := []struct {
		query    string
		expected bool
	}{
		{"explícame qué es una goroutine en Go", false},
		{"hola cómo estás", false},
		{"redacta una carta de renuncia", false},
		{"abre la calculadora", true},
		{"cierra esta ventana", true},
		{"qué tengo abierto en pantalla", true},
		{"mira el error copiado en el portapapeles", true},
		{"cuánta memoria RAM está libre", true},
	}

	for _, c := range cases {
		got := IsOSRelevantQuery(c.query)
		if got != c.expected {
			t.Errorf("query %q: expected %v, got %v", c.query, c.expected, got)
		}
	}
}
