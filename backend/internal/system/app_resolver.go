//go:build windows

package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AppLaunchInfo contiene la información de ejecución para una aplicación.
type AppLaunchInfo struct {
	Command string
	IsAUMID bool
}

// ResolveAppExecutable busca dinámicamente y resuelve el ejecutable, acceso directo o AUMID
// para cualquier aplicación solicitada en Windows.
func ResolveAppExecutable(name string) (*AppLaunchInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("el nombre de la aplicación no puede estar vacío")
	}

	clean := strings.ToLower(name)
	for _, prefix := range []string{"abrir con ", "abre con ", "abrir ", "abre ", "lanza ", "ejecuta ", "el ", "la ", "los ", "las ", "un ", "una "} {
		if strings.HasPrefix(clean, prefix) {
			clean = strings.TrimSpace(clean[len(prefix):])
		}
	}
	clean = strings.TrimSuffix(clean, " ide")
	clean = strings.TrimSuffix(clean, ".exe")

	// Si el objetivo es una URL directa (http/https/www)
	if strings.HasPrefix(clean, "http://") || strings.HasPrefix(clean, "https://") {
		return &AppLaunchInfo{Command: name}, nil
	}
	if strings.HasPrefix(clean, "www.") {
		return &AppLaunchInfo{Command: "https://" + name}, nil
	}

	userProfile := os.Getenv("USERPROFILE")
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" && userProfile != "" {
		localAppData = filepath.Join(userProfile, "AppData", "Local")
	}
	appData := os.Getenv("APPDATA")
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")

	// 1. Mapeo a ejecutables estándar de Windows y AUMID
	standardMap := map[string]string{
		"notepad":       "notepad.exe",
		"bloc de notas": "notepad.exe",
		"bloc":          "notepad.exe",
		"calc":          "calc.exe",
		"calculadora":   "calc.exe",
		"calculator":    "calc.exe",
		"paint":         "mspaint.exe",
		"mspaint":       "mspaint.exe",
	}
	if exe, ok := standardMap[clean]; ok {
		return &AppLaunchInfo{
			Command: exe,
		}, nil
	}

	// 2. Mapeo de aplicaciones de desarrollo y herramientas populares del sistema
	switch clean {
	case "antigravity", "antigravityide", "agy":
		candidates := []string{
			filepath.Join(localAppData, "Programs", "Antigravity", "Antigravity.exe"),
			filepath.Join(localAppData, "Programs", "Antigravity IDE", "Antigravity IDE.exe"),
			filepath.Join(programFiles, "Antigravity", "Antigravity.exe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return &AppLaunchInfo{Command: c}, nil
			}
		}

	case "code", "vscode", "visual studio code", "vs code":
		candidates := []string{
			filepath.Join(localAppData, "Programs", "Microsoft VS Code", "Code.exe"),
			filepath.Join(programFiles, "Microsoft VS Code", "Code.exe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return &AppLaunchInfo{Command: c}, nil
			}
		}

	case "cursor":
		candidate := filepath.Join(localAppData, "Programs", "cursor", "Cursor.exe")
		if _, err := os.Stat(candidate); err == nil {
			return &AppLaunchInfo{Command: candidate}, nil
		}

	case "explorer", "explorador", "explorador de archivos", "archivos", "mis documentos":
		return &AppLaunchInfo{Command: "explorer.exe"}, nil

	case "chrome", "google chrome", "navegador":
		candidates := []string{
			filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(programFilesX86, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return &AppLaunchInfo{Command: c}, nil
			}
		}
		return &AppLaunchInfo{Command: "chrome.exe"}, nil

	case "brave", "brave browser":
		candidates := []string{
			filepath.Join(programFiles, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
			filepath.Join(localAppData, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return &AppLaunchInfo{Command: c}, nil
			}
		}

	case "edge", "microsoft edge":
		return &AppLaunchInfo{Command: "msedge.exe"}, nil

	case "docker", "docker desktop":
		candidate := filepath.Join(programFiles, "Docker", "Docker", "Docker Desktop.exe")
		if _, err := os.Stat(candidate); err == nil {
			return &AppLaunchInfo{Command: candidate}, nil
		}

	case "terminal", "powershell", "consola", "cmd":
		if lp, err := exec.LookPath("wt.exe"); err == nil {
			return &AppLaunchInfo{Command: lp}, nil
		}
		return &AppLaunchInfo{Command: "powershell.exe"}, nil

	case "configuracion", "configuración", "settings", "ajustes":
		return &AppLaunchInfo{Command: "ms-settings:"}, nil

	case "taskmgr", "administrador de tareas", "task manager":
		return &AppLaunchInfo{Command: "taskmgr.exe"}, nil
	}

	// 3. Comprobar si ya es ejecutable o existe en PATH
	if lp, err := exec.LookPath(name); err == nil {
		return &AppLaunchInfo{Command: lp}, nil
	}
	if lp, err := exec.LookPath(clean + ".exe"); err == nil {
		return &AppLaunchInfo{Command: lp}, nil
	}

	// 4. Búsqueda dinámica en carpetas de Programas (%LOCALAPPDATA%\Programs)
	programsDir := filepath.Join(localAppData, "Programs")
	if entries, err := os.ReadDir(programsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				folderLower := strings.ToLower(e.Name())
				if strings.Contains(folderLower, clean) || strings.Contains(clean, folderLower) {
					subDir := filepath.Join(programsDir, e.Name())
					if subEntries, err := os.ReadDir(subDir); err == nil {
						for _, se := range subEntries {
							if !se.IsDir() && strings.HasSuffix(strings.ToLower(se.Name()), ".exe") {
								return &AppLaunchInfo{Command: filepath.Join(subDir, se.Name())}, nil
							}
						}
					}
				}
			}
		}
	}

	// 5. Búsqueda en accesos directos del Menú Inicio (.lnk)
	startMenuDirs := []string{
		filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs"),
		filepath.Join(os.Getenv("ProgramData"), "Microsoft", "Windows", "Start Menu", "Programs"),
	}
	for _, smDir := range startMenuDirs {
		var foundShortcut string
		_ = filepath.WalkDir(smDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(d.Name()), ".lnk") {
				base := strings.ToLower(strings.TrimSuffix(d.Name(), ".lnk"))
				if strings.Contains(base, clean) || strings.Contains(clean, base) {
					foundShortcut = path
					return fmt.Errorf("found")
				}
			}
			return nil
		})
		if foundShortcut != "" {
			return &AppLaunchInfo{Command: foundShortcut}, nil
		}
	}

	// 6. Por defecto, devolver el nombre con extensión .exe si no contiene ya extensión
	if !strings.HasSuffix(strings.ToLower(name), ".exe") && !strings.Contains(name, ":") {
		return &AppLaunchInfo{Command: name + ".exe"}, nil
	}

	return &AppLaunchInfo{Command: name}, nil
}
