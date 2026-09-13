package browser

import (
	"runtime"
	"testing"
)

func TestGetBrowserExecutable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Prueba exclusiva de Windows")
	}

	chromeExe := GetBrowserExecutable("chrome")
	if chromeExe == "" {
		t.Log("Aviso: chrome.exe no encontrado en rutas estándar")
	} else {
		t.Logf("chrome.exe encontrado en: %s", chromeExe)
	}
}

func TestDetectProfiles(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Prueba exclusiva de Windows")
	}

	profiles := DetectProfiles()
	if len(profiles) == 0 {
		t.Log("No se encontraron perfiles en rutas de usuario de prueba")
	} else {
		t.Logf("Se detectaron %d perfiles de navegador", len(profiles))
		for _, p := range profiles {
			t.Logf("- [%s] %s (%s) - Email: %s", p.Browser, p.Name, p.Directory, p.Email)
		}
	}
}

func TestFindMatchingProfile(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Prueba exclusiva de Windows")
	}

	p := FindMatchingProfile("chrome", "")
	if p != nil {
		t.Logf("Perfil por defecto encontrado: %s (%s)", p.Name, p.Email)
	}
}
