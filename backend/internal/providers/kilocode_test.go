package providers

import (
	"testing"
)

// TestKiloCodeProviderFields verifica que el proveedor KiloCode se inicialice con
// los valores correctos de nombre, base URL, modelo por defecto y modelos disponibles.
func TestKiloCodeProviderFields(t *testing.T) {
	p := NewKiloCode("test-jwt-token")

	if p.Name() != "kilocode" {
		t.Errorf("Name() = %q; want %q", p.Name(), "kilocode")
	}
	if p.cfg.baseURL != "https://api.kilo.ai/api/gateway" {
		t.Errorf("baseURL = %q; want %q", p.cfg.baseURL, "https://api.kilo.ai/api/gateway")
	}
	if p.cfg.model != "kilo/anthropic/claude-sonnet-4-5" {
		t.Errorf("default model = %q; want %q", p.cfg.model, "kilo/anthropic/claude-sonnet-4-5")
	}
	if !p.SupportsTools() {
		t.Error("SupportsTools() should return true for KiloCode")
	}

	models := p.Models()
	if len(models) == 0 {
		t.Fatal("Models() returned empty list")
	}
	// Verifica que modelos de distintos proveedores estén presentes en el gateway
	wantPrefixes := []string{"kilo/anthropic/", "kilo/openai/"}
	for _, prefix := range wantPrefixes {
		found := false
		for _, m := range models {
			if len(m) > len(prefix) && m[:len(prefix)] == prefix {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Models() missing any model with prefix %q", prefix)
		}
	}
}

// TestKiloCodeRegistryIntegration verifica registro y recuperación del proveedor kilocode.
func TestKiloCodeRegistryIntegration(t *testing.T) {
	const testKey = "test-kilocode-jwt-xyz"

	RegisterProviderKey("kilocode", testKey)
	defer Unregister("kilocode")

	p, err := Get("kilocode")
	if err != nil {
		t.Fatalf("Get('kilocode') after register: %v", err)
	}
	if p.Name() != "kilocode" {
		t.Errorf("registered provider name = %q; want 'kilocode'", p.Name())
	}

	// Acepta alias "kilo"
	RegisterProviderKey("kilo", testKey)
	p2, err := Get("kilocode")
	if err != nil {
		t.Fatalf("Get('kilocode') after 'kilo' alias register: %v", err)
	}
	if p2.Name() != "kilocode" {
		t.Errorf("alias 'kilo' provider name = %q; want 'kilocode'", p2.Name())
	}
}
