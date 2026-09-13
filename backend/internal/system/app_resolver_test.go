package system

import (
	"runtime"
	"strings"
	"testing"
)

func TestResolveAppExecutable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping Windows app resolver test on non-windows")
	}

	tests := []struct {
		input       string
		mustContain string
		isAUMID     bool
	}{
		{"antigravity", "Antigravity", false},
		{"antigravity ide", "Antigravity", false},
		{"abre con antigravity ide", "Antigravity", false},
		{"notepad", "notepad", false},
		{"calc", "calc", false},
		{"calculadora", "calc", false},
		{"explorer", "explorer.exe", false},
	}

	for _, tt := range tests {
		info, err := ResolveAppExecutable(tt.input)
		if err != nil {
			t.Errorf("ResolveAppExecutable(%q) failed: %v", tt.input, err)
			continue
		}
		if info.IsAUMID != tt.isAUMID {
			t.Errorf("ResolveAppExecutable(%q).IsAUMID = %v, expected %v", tt.input, info.IsAUMID, tt.isAUMID)
		}
		if !strings.Contains(strings.ToLower(info.Command), strings.ToLower(tt.mustContain)) {
			t.Errorf("ResolveAppExecutable(%q).Command = %q, expected to contain %q", tt.input, info.Command, tt.mustContain)
		}
		t.Logf("Resolved %q -> %s (IsAUMID: %v)", tt.input, info.Command, info.IsAUMID)
	}
}
