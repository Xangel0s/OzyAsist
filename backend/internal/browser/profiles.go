package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type BrowserProfile struct {
	Browser     string `json:"browser"`       // "chrome", "edge", "brave"
	Directory   string `json:"directory"`     // "Default", "Profile 1", "Profile 2"
	Name        string `json:"name"`          // Nombre asignado al perfil (ej: "Tu Chrome", "Personal", "Trabajo")
	Email       string `json:"email"`         // Correo de la cuenta de Google / Microsoft (ej: "zastuto5@gmail.com")
	DisplayName string `json:"display_name"`  // Nombre de la persona (ej: "Zorro Astuto")
	ExecPath    string `json:"exec_path"`     // Ruta al ejecutable (ej: C:\Program Files\Google\Chrome\Application\chrome.exe)
	UserDataDir string `json:"user_data_dir"` // Ruta a User Data
}

type localStateSchema struct {
	Profile struct {
		InfoCache map[string]struct {
			Name     string `json:"name"`
			UserName string `json:"user_name"`
			GaiaName string `json:"gaia_name"`
		} `json:"info_cache"`
	} `json:"profile"`
}

// GetBrowserExecutable busca el ejecutable del navegador en Windows
func GetBrowserExecutable(browserType string) string {
	if runtime.GOOS != "windows" {
		return ""
	}

	progFiles := os.Getenv("ProgramFiles")
	progFiles86 := os.Getenv("ProgramFiles(x86)")
	localApp := os.Getenv("LOCALAPPDATA")

	var candidates []string
	switch strings.ToLower(browserType) {
	case "chrome", "google-chrome":
		candidates = []string{
			filepath.Join(progFiles, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(progFiles86, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(localApp, "Google", "Chrome", "Application", "chrome.exe"),
		}
	case "edge", "msedge":
		candidates = []string{
			filepath.Join(progFiles86, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(progFiles, "Microsoft", "Edge", "Application", "msedge.exe"),
		}
	case "brave":
		candidates = []string{
			filepath.Join(progFiles, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
			filepath.Join(localApp, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		}
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return ""
}

// DetectProfiles escanea los perfiles existentes en Chrome, Edge y Brave
func DetectProfiles() []BrowserProfile {
	var profiles []BrowserProfile
	localApp := os.Getenv("LOCALAPPDATA")
	if localApp == "" {
		return profiles
	}

	targets := []struct {
		browser string
		subpath string
	}{
		{"chrome", filepath.Join("Google", "Chrome", "User Data")},
		{"edge", filepath.Join("Microsoft", "Edge", "User Data")},
		{"brave", filepath.Join("BraveSoftware", "Brave-Browser", "User Data")},
	}

	for _, target := range targets {
		userDataDir := filepath.Join(localApp, target.subpath)
		localStatePath := filepath.Join(userDataDir, "Local State")

		data, err := os.ReadFile(localStatePath)
		if err != nil {
			continue
		}

		var state localStateSchema
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		execPath := GetBrowserExecutable(target.browser)

		for dirName, info := range state.Profile.InfoCache {
			displayName := strings.TrimSpace(info.GaiaName)
			profileName := strings.TrimSpace(info.Name)
			email := strings.TrimSpace(info.UserName)

			if profileName == "" {
				profileName = dirName
			}

			profiles = append(profiles, BrowserProfile{
				Browser:     target.browser,
				Directory:   dirName,
				Name:        profileName,
				Email:       email,
				DisplayName: displayName,
				ExecPath:    execPath,
				UserDataDir: userDataDir,
			})
		}
	}

	return profiles
}

// FindMatchingProfile encuentra el perfil que mejor coincida por email, nombre o directorio
func FindMatchingProfile(browserType string, query string) *BrowserProfile {
	profiles := DetectProfiles()
	if len(profiles) == 0 {
		return nil
	}

	q := strings.ToLower(strings.TrimSpace(query))
	bType := strings.ToLower(strings.TrimSpace(browserType))

	// 1. Coincidencia exacta por email
	for _, p := range profiles {
		if bType != "" && p.Browser != bType {
			continue
		}
		if p.Email != "" && strings.ToLower(p.Email) == q {
			return &p
		}
	}

	// 2. Coincidencia por subcadena de email o nombre
	if q != "" {
		for _, p := range profiles {
			if bType != "" && p.Browser != bType {
				continue
			}
			if strings.Contains(strings.ToLower(p.Email), q) ||
				strings.Contains(strings.ToLower(p.Name), q) ||
				strings.Contains(strings.ToLower(p.DisplayName), q) ||
				strings.EqualFold(p.Directory, q) {
				return &p
			}
		}
	}

	// 3. Si no hay query específica, devolver el perfil predeterminado del navegador solicitado
	for _, p := range profiles {
		if (bType == "" || p.Browser == bType) && p.Directory == "Default" {
			return &p
		}
	}

	// 4. Fallback al primer perfil encontrado
	if len(profiles) > 0 {
		return &profiles[0]
	}

	return nil
}

// LaunchWithProfile abre una URL en el navegador especificado usando el perfil solicitado
func LaunchWithProfile(ctx context.Context, profile *BrowserProfile, targetURL string) error {
	browserName := "chrome"
	if profile != nil && profile.Browser != "" {
		browserName = profile.Browser
	}

	execPath := ""
	if profile != nil && profile.ExecPath != "" {
		execPath = profile.ExecPath
	} else {
		execPath = GetBrowserExecutable(browserName)
	}

	// 1. Lanzamiento directo al ejecutable del navegador en el escritorio interactivo WinSta0\Default
	if execPath != "" {
		var args []string
		if profile != nil && profile.Directory != "" {
			args = append(args, fmt.Sprintf("--profile-directory=%s", profile.Directory))
		}
		args = append(args, targetURL)

		if err := launchOnDefaultDesktop(execPath, args...); err == nil {
			go bringBrowserToFront(browserName)
			return nil
		}
	}

	// 2. Fallback secundario a Shell de Windows / xdg-open
	err := openViaWin32Shell(targetURL)
	go bringBrowserToFront(browserName)
	return err
}

func isBrowserProcessRunning(browserName string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	clean := strings.TrimSuffix(strings.ToLower(browserName), ".exe")
	cmd := exec.Command("tasklist.exe", "/fi", fmt.Sprintf("imagename eq %s.exe", clean), "/fo", "csv", "/nh")
	out, err := cmd.Output()
	return err == nil && strings.Contains(strings.ToLower(string(out)), clean+".exe")
}

// launchOnDefaultDesktop ejecuta un proceso explícitamente vinculado al escritorio interactivo WinSta0\Default.
// Esto es indispensable en Windows para que Chrome detecte la instancia en ejecución del usuario y abra la nueva pestaña
// sin generar conflictos de Lock de ProcessSingleton (código 21).
func launchOnDefaultDesktop(execPath string, args ...string) error {
	if runtime.GOOS != "windows" {
		cmd := exec.Command(execPath, args...)
		return cmd.Start()
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procCreateProcessW := kernel32.NewProc("CreateProcessW")

	var parts []string
	parts = append(parts, fmt.Sprintf(`"%s"`, execPath))
	for _, a := range args {
		parts = append(parts, fmt.Sprintf(`"%s"`, a))
	}
	cmdLine := strings.Join(parts, " ")
	cmdLinePtr, err := syscall.UTF16PtrFromString(cmdLine)
	if err != nil {
		return err
	}

	desktopStr, err := syscall.UTF16PtrFromString("WinSta0\\Default")
	if err != nil {
		return err
	}

	var si syscall.StartupInfo
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Desktop = desktopStr

	var pi syscall.ProcessInformation

	ret, _, callErr := procCreateProcessW.Call(
		0,
		uintptr(unsafe.Pointer(cmdLinePtr)),
		0,
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&si)),
		uintptr(unsafe.Pointer(&pi)),
	)
	if ret == 0 {
		return callErr
	}

	syscall.CloseHandle(pi.Process)
	syscall.CloseHandle(pi.Thread)
	return nil
}

func openViaWin32Shell(targetURL string) error {
	if runtime.GOOS != "windows" {
		cmd := exec.Command("xdg-open", targetURL)
		return cmd.Start()
	}

	shell32 := syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW := shell32.NewProc("ShellExecuteW")
	verbPtr, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	filePtr, err := syscall.UTF16PtrFromString(targetURL)
	if err != nil {
		return err
	}
	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		0,
		0,
		1, // SW_SHOWNORMAL
	)
	if ret <= 32 {
		cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", targetURL)
		return cmd.Start()
	}
	return nil
}

func bringBrowserToFront(browserName string) {
	if runtime.GOOS != "windows" {
		return
	}
	time.Sleep(450 * time.Millisecond)

	// 1. Activar directamente la ventana de Chrome/Edge vinculando el hilo al escritorio Default
	user32 := syscall.NewLazyDLL("user32.dll")
	procOpenDesktopW := user32.NewProc("OpenDesktopW")
	procSetThreadDesktop := user32.NewProc("SetThreadDesktop")
	procEnumWindows := user32.NewProc("EnumWindows")
	procGetClassName := user32.NewProc("GetClassNameW")
	procIsWinVisible := user32.NewProc("IsWindowVisible")
	procShowWindow := user32.NewProc("ShowWindow")
	procBringWindowToTop := user32.NewProc("BringWindowToTop")
	procSetForeground := user32.NewProc("SetForegroundWindow")

	deskName, _ := syscall.UTF16PtrFromString("Default")
	hDesk, _, _ := procOpenDesktopW.Call(uintptr(unsafe.Pointer(deskName)), 0, 0, 0x10000000)
	if hDesk != 0 {
		procSetThreadDesktop.Call(hDesk)

		targetClass := "Chrome_WidgetWin_1"
		if strings.Contains(strings.ToLower(browserName), "edge") {
			targetClass = "MSEdge"
		}

		cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
			vis, _, _ := procIsWinVisible.Call(hwnd)
			if vis == 0 {
				return 1
			}
			var classBuf [256]uint16
			procGetClassName.Call(hwnd, uintptr(unsafe.Pointer(&classBuf[0])), 256)
			className := syscall.UTF16ToString(classBuf[:])

			if strings.EqualFold(className, targetClass) {
				procShowWindow.Call(hwnd, 9) // SW_RESTORE
				procBringWindowToTop.Call(hwnd)
				procSetForeground.Call(hwnd)
			}
			return 1
		})
		procEnumWindows.Call(cb, 0)
	}

	// 2. Fallback WScript AppActivate
	script := fmt.Sprintf(`$w = New-Object -ComObject Wscript.Shell; Get-Process %s -ErrorAction SilentlyContinue | ForEach-Object { if ($w.AppActivate($_.Id)) { exit } }`, browserName)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	_ = cmd.Run()
}
