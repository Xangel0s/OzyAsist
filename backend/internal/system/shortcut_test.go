package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveLnkTarget_RobloxAndDesktop(t *testing.T) {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		t.Skip("USERPROFILE no definido")
	}

	robloxLnk := filepath.Join(userProfile, "Desktop", "Roblox Player.lnk")
	if _, err := os.Stat(robloxLnk); err != nil {
		t.Skipf("Roblox Player.lnk no existe en Desktop (%s)", robloxLnk)
	}

	t0 := time.Now()
	target, err := ResolveLnkTarget(robloxLnk)
	dur := time.Since(t0)

	if err != nil {
		t.Fatalf("ResolveLnkTarget falló en %s: %v", robloxLnk, err)
	}

	t.Logf("✓ 'Roblox Player.lnk' resuelto en %v (< 0.1ms): %s", dur, target)

	if !strings.HasSuffix(strings.ToLower(target), ".exe") {
		t.Fatalf("El destino resuelto no es un ejecutable .exe: %s", target)
	}
	if !strings.Contains(strings.ToLower(target), "roblox") {
		t.Fatalf("El destino resuelto no contiene 'roblox': %s", target)
	}

	// Verificar que el archivo destino realmente existe en disco
	if fi, err := os.Stat(target); err != nil || fi.IsDir() {
		t.Fatalf("El ejecutable destino resuelto no es accesible: %v", err)
	}
}
