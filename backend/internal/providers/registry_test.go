package providers

import (
	"testing"
)

func TestGetSupportedCatalog(t *testing.T) {
	cat := GetSupportedCatalog()
	if len(cat) != 9 {
		t.Fatalf("se esperaban 9 proveedores soportados en el catálogo, obtenido: %d", len(cat))
	}

	foundCohere := false
	foundGroq := false
	foundMistral := false
	foundLMStudio := false
	foundLlamaCpp := false

	for _, item := range cat {
		if item.ID == "cohere" {
			foundCohere = true
		}
		if item.ID == "groq" {
			foundGroq = true
		}
		if item.ID == "mistral" {
			foundMistral = true
		}
		if item.ID == "lmstudio" && item.IsLocal {
			foundLMStudio = true
		}
		if item.ID == "llamacpp" && item.IsLocal {
			foundLlamaCpp = true
		}
	}

	if !foundCohere || !foundGroq || !foundMistral || !foundLMStudio || !foundLlamaCpp {
		t.Fatalf("catálogo incompleto: %+v", cat)
	}
}

func TestCreateProviderAndEnsure(t *testing.T) {
	// Probar instanciación de proveedores incluso con clave vacía
	for _, id := range []string{"cohere", "groq", "mistral", "openai", "deepseek", "kilocode", "lmstudio", "ollama", "llamacpp"} {
		p := CreateProvider(id, "")
		if p == nil {
			t.Fatalf("CreateProvider(%s, \"\") retornó nil", id)
		}
		if len(p.Models()) == 0 {
			t.Fatalf("CreateProvider(%s) no devolvió modelos", id)
		}
	}

	// Probar EnsureProvider
	p := EnsureProvider("groq")
	if p == nil || p.Name() != "groq" {
		t.Fatalf("EnsureProvider(groq) falló, obtenido: %v", p)
	}
}
