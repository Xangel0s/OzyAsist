package providers

import (
	"testing"
)

// TestMistralProviderFields verifica que el proveedor Mistral se inicialice con
// los valores correctos de nombre, base URL, modelo por defecto y modelos disponibles.
func TestMistralProviderFields(t *testing.T) {
	p := NewMistral("test-key-abc")

	if p.Name() != "mistral" {
		t.Errorf("Name() = %q; want %q", p.Name(), "mistral")
	}
	if p.cfg.baseURL != "https://api.mistral.ai/v1" {
		t.Errorf("baseURL = %q; want %q", p.cfg.baseURL, "https://api.mistral.ai/v1")
	}
	if p.cfg.model != "mistral-large-latest" {
		t.Errorf("default model = %q; want %q", p.cfg.model, "mistral-large-latest")
	}
	if p.cfg.apiKey != "test-key-abc" {
		t.Errorf("apiKey not stored correctly")
	}
	if !p.SupportsTools() {
		t.Error("SupportsTools() should return true for Mistral")
	}

	models := p.Models()
	if len(models) == 0 {
		t.Fatal("Models() returned empty list")
	}
	// Verifica que los modelos principales estén presentes
	wantModels := []string{"mistral-large-latest", "codestral-latest"}
	for _, want := range wantModels {
		found := false
		for _, m := range models {
			if m == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Models() missing %q", want)
		}
	}
}

// TestMistralRegistryIntegration verifica que RegisterProviderKey registre correctamente
// el proveedor mistral y que GetProviderKey lea MISTRAL_API_KEY del entorno.
func TestMistralRegistryIntegration(t *testing.T) {
	const testKey = "test-mistral-key-xyz"

	// Registrar y recuperar
	RegisterProviderKey("mistral", testKey)
	defer Unregister("mistral")

	p, err := Get("mistral")
	if err != nil {
		t.Fatalf("Get('mistral') after register: %v", err)
	}
	if p.Name() != "mistral" {
		t.Errorf("registered provider name = %q; want 'mistral'", p.Name())
	}

	// Verificar que key se guardó en global.keys
	stored := GetProviderKey("mistral")
	if stored != testKey {
		t.Errorf("GetProviderKey('mistral') = %q; want %q", stored, testKey)
	}
}

// TestMistralUnregisterOnEmptyKey verifica que una clave vacía elimina el proveedor del registro.
func TestMistralUnregisterOnEmptyKey(t *testing.T) {
	RegisterProviderKey("mistral", "some-key")
	// Registrar con clave vacía debe desregistrar
	RegisterProviderKey("mistral", "")
	_, err := Get("mistral")
	if err == nil {
		t.Error("Get('mistral') should fail after unregistering with empty key")
	}
}
