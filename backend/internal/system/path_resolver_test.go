package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveUserPath(t *testing.T) {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		t.Skip("USERPROFILE not set")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"mis descargas", filepath.Join(userProfile, "Downloads")},
		{"descargas", filepath.Join(userProfile, "Downloads")},
		{"mi escritorio", filepath.Join(userProfile, "Desktop")},
		{"desktop", filepath.Join(userProfile, "Desktop")},
		{"mis documentos", filepath.Join(userProfile, "Documents")},
		{"mis imágenes", filepath.Join(userProfile, "Pictures")},
		{"~", userProfile},
		{"~/test", filepath.Join(userProfile, "test")},
		{`~\test`, filepath.Join(userProfile, "test")},
		{"crmgeofal", filepath.Join(userProfile, "Documents", "crmgeofal")},
		{"el proyecto crmgeofal", filepath.Join(userProfile, "Documents", "crmgeofal")},
		{"crmgeofal de documentos", filepath.Join(userProfile, "Documents", "crmgeofal")},
		{"crmgeofal en documentos", filepath.Join(userProfile, "Documents", "crmgeofal")},
	}

	for _, tt := range tests {
		got := ResolveUserPath(tt.input)
		if !strings.EqualFold(got, tt.expected) {
			t.Errorf("ResolveUserPath(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}
