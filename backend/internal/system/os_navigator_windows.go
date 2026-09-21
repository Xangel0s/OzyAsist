//go:build windows

package system

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

var (
	modUser32               = syscall.NewLazyDLL("user32.dll")
	procEnumWindows         = modUser32.NewProc("EnumWindows")
	procEnumChildWindows    = modUser32.NewProc("EnumChildWindows")
	procGetWindowTextW      = modUser32.NewProc("GetWindowTextW")
	procGetWindowTextLength = modUser32.NewProc("GetWindowTextLengthW")
	procGetClassNameW       = modUser32.NewProc("GetClassNameW")
	procIsWindowVisible     = modUser32.NewProc("IsWindowVisible")
	procGetWindowThreadPID  = modUser32.NewProc("GetWindowThreadProcessId")
	procSetForegroundWindow = modUser32.NewProc("SetForegroundWindow")
	procShowWindow          = modUser32.NewProc("ShowWindow")
	procBringWindowToTop    = modUser32.NewProc("BringWindowToTop")
	procKeybdEvent          = modUser32.NewProc("keybd_event")
	procAttachThreadInput   = modUser32.NewProc("AttachThreadInput")
	procSwitchToThisWindow  = modUser32.NewProc("SwitchToThisWindow")
	procGetForegroundWindow = modUser32.NewProc("GetForegroundWindow")
	procOpenInputDesktop    = modUser32.NewProc("OpenInputDesktop")
	procMoveWindow          = modUser32.NewProc("MoveWindow")
	procSystemParametersInfoW = modUser32.NewProc("SystemParametersInfoW")

	modKernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procGetCurrentThreadId         = modKernel32.NewProc("GetCurrentThreadId")
	procOpenProcess                = modKernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = modKernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                = modKernel32.NewProc("CloseHandle")

	modShell32              = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW       = modShell32.NewProc("ShellExecuteW")
)

type WindowsNavigator struct{}

func NewWindowsNavigator() OSNavigator {
	return &WindowsNavigator{}
}

// GetKnownFolders resuelve las rutas canónicas de Windows
func (w *WindowsNavigator) GetKnownFolders(_ context.Context) (KnownFolders, error) {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		userProfile = "C:\\Users\\" + os.Getenv("USERNAME")
	}

	return KnownFolders{
		UserDesktop:     filepath.Join(userProfile, "Desktop"),
		PublicDesktop:   "C:\\Users\\Public\\Desktop",
		Documents:       filepath.Join(userProfile, "Documents"),
		Downloads:       filepath.Join(userProfile, "Downloads"),
		Pictures:        filepath.Join(userProfile, "Pictures"),
		Videos:          filepath.Join(userProfile, "Videos"),
		AppDataLocal:    os.Getenv("LOCALAPPDATA"),
		AppDataRoam:     os.Getenv("APPDATA"),
		ProgramFiles:    os.Getenv("ProgramFiles"),
		ProgramFilesX86: os.Getenv("ProgramFiles(x86)"),
	}, nil
}

// GetDesktopItems escanea nativamente los directorios de escritorio personal y público
func (w *WindowsNavigator) GetDesktopItems(ctx context.Context) ([]DesktopItem, error) {
	folders, err := w.GetKnownFolders(ctx)
	if err != nil {
		return nil, err
	}

	var items []DesktopItem
	seenNames := make(map[string]bool)

	// 1. Escanear Escritorio de Usuario
	userEntries, _ := os.ReadDir(folders.UserDesktop)
	for _, entry := range userEntries {
		name := entry.Name()
		if strings.EqualFold(name, "desktop.ini") {
			continue
		}
		seenNames[strings.ToLower(name)] = true
		fullPath := filepath.Join(folders.UserDesktop, name)
		info, _ := entry.Info()

		var size int64
		var modTime time.Time
		if info != nil {
			size = info.Size()
			modTime = info.ModTime()
		}

		kind := classifyItem(entry, name)
		items = append(items, DesktopItem{
			Name:       name,
			Path:       fullPath,
			Kind:       kind,
			Size:       size,
			ModifiedAt: modTime,
			IsPublic:   false,
		})
	}

	// 2. Escanear Escritorio Público (C:\Users\Public\Desktop)
	publicEntries, _ := os.ReadDir(folders.PublicDesktop)
	for _, entry := range publicEntries {
		name := entry.Name()
		if strings.EqualFold(name, "desktop.ini") || seenNames[strings.ToLower(name)] {
			continue
		}
		seenNames[strings.ToLower(name)] = true
		fullPath := filepath.Join(folders.PublicDesktop, name)
		info, _ := entry.Info()

		var size int64
		var modTime time.Time
		if info != nil {
			size = info.Size()
			modTime = info.ModTime()
		}

		kind := classifyItem(entry, name)
		items = append(items, DesktopItem{
			Name:       name,
			Path:       fullPath,
			Kind:       kind,
			Size:       size,
			ModifiedAt: modTime,
			IsPublic:   true,
		})
	}

	// 3. Iconos virtuales estándar del Shell de Windows
	items = append(items,
		DesktopItem{Name: "Papelera de reciclaje", Kind: ItemKindSystemIcon},
		DesktopItem{Name: "Este equipo", Kind: ItemKindSystemIcon},
		DesktopItem{Name: "Red", Kind: ItemKindSystemIcon},
		DesktopItem{Name: "Panel de control", Kind: ItemKindSystemIcon},
	)

	return items, nil
}

func classifyItem(entry fs.DirEntry, name string) DesktopItemKind {
	if entry.IsDir() {
		return ItemKindFolder
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".lnk") || strings.HasSuffix(lower, ".url") {
		return ItemKindShortcut
	}
	return ItemKindFile
}

// GetInstalledSoftware lee directamente las claves del registro de Windows (HKLM y HKCU)
func (w *WindowsNavigator) GetInstalledSoftware(_ context.Context, filter string) ([]InstalledApp, error) {
	var apps []InstalledApp
	seen := make(map[string]bool)

	type regTarget struct {
		root   registry.Key
		path   string
		source string
		access uint32
	}

	targets := []regTarget{
		// HKLM 64-bit
		{registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Uninstall`, "HKLM64", registry.READ | registry.WOW64_64KEY},
		// HKLM 32-bit (WOW6432Node)
		{registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Uninstall`, "HKLM32", registry.READ | registry.WOW64_32KEY},
		// HKCU Usuario actual
		{registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall`, "HKCU", registry.READ},
	}

	for _, t := range targets {
		k, err := registry.OpenKey(t.root, t.path, t.access)
		if err != nil {
			continue
		}

		subkeys, err := k.ReadSubKeyNames(-1)
		k.Close()
		if err != nil {
			continue
		}

		for _, sub := range subkeys {
			subKey, err := registry.OpenKey(t.root, t.path+`\`+sub, t.access)
			if err != nil {
				continue
			}

			displayName, _, _ := subKey.GetStringValue("DisplayName")
			if displayName == "" {
				subKey.Close()
				continue
			}

			// Evitar duplicados exactos
			keyIdent := strings.ToLower(strings.TrimSpace(displayName))
			if seen[keyIdent] {
				subKey.Close()
				continue
			}

			// Aplicar filtro si existe
			if filter != "" && !strings.Contains(keyIdent, strings.ToLower(filter)) {
				subKey.Close()
				continue
			}

			seen[keyIdent] = true
			publisher, _, _ := subKey.GetStringValue("Publisher")
			version, _, _ := subKey.GetStringValue("DisplayVersion")
			installLoc, _, _ := subKey.GetStringValue("InstallLocation")
			uninstallStr, _, _ := subKey.GetStringValue("UninstallString")
			installDate, _, _ := subKey.GetStringValue("InstallDate")

			subKey.Close()

			apps = append(apps, InstalledApp{
				DisplayName:     displayName,
				Publisher:       publisher,
				DisplayVersion:  version,
				InstallLocation: installLoc,
				UninstallString: uninstallStr,
				InstallDate:     installDate,
				RegistrySource:  t.source,
			})
		}
	}

	return apps, nil
}

// withInteractiveDesktop asegura que la llamada Win32 se ejecute en el hilo
// atado al escritorio interactivo del usuario actual ("Default" o activo).
func withInteractiveDesktop(fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	procOpenDesktopW := modUser32.NewProc("OpenDesktopW")
	procGetThreadDesktop := modUser32.NewProc("GetThreadDesktop")
	procSetThreadDesktop := modUser32.NewProc("SetThreadDesktop")
	procCloseDesktop := modUser32.NewProc("CloseDesktop")

	curTid, _, _ := procGetCurrentThreadId.Call()
	hOrigDesk, _, _ := procGetThreadDesktop.Call(curTid)

	// Intentar obtener el escritorio interactivo donde el usuario está recibiendo input
	hDesk, _, _ := procOpenInputDesktop.Call(0, 0, 0x01FF)
	if hDesk == 0 {
		deskNamePtr := uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Default")))
		hDesk, _, _ = procOpenDesktopW.Call(deskNamePtr, 0, 0, 0x01FF)
	}

	if hDesk != 0 {
		procSetThreadDesktop.Call(hDesk)
		defer func() {
			if hOrigDesk != 0 {
				procSetThreadDesktop.Call(hOrigDesk)
			}
			procCloseDesktop.Call(hDesk)
		}()
	}

	fn()
}

func getProcessName(pid uint32) string {
	if pid == 0 {
		return ""
	}
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	hProc, _, _ := procOpenProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if hProc == 0 {
		return ""
	}
	defer procCloseHandle.Call(hProc)

	var buf [1024]uint16
	size := uint32(len(buf))
	ret, _, _ := procQueryFullProcessImageNameW.Call(hProc, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret != 0 && size > 0 {
		fullPath := syscall.UTF16ToString(buf[:size])
		return filepath.Base(fullPath)
	}
	return ""
}


// GetActiveWindows enumera las ventanas visibles en el escritorio usando user32.dll
func (w *WindowsNavigator) GetActiveWindows(_ context.Context) ([]WindowInfo, error) {
	var list []WindowInfo
	seen := make(map[uintptr]bool)
	seenKey := make(map[string]bool)

	withInteractiveDesktop(func() {
		cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
			if seen[hwnd] {
				return 1
			}
			seen[hwnd] = true

			vis, _, _ := procIsWindowVisible.Call(hwnd)
			if vis == 0 {
				return 1
			}

			length, _, _ := procGetWindowTextLength.Call(hwnd)
			if length == 0 {
				return 1
			}

			buf := make([]uint16, length+1)
			procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), length+1)
			cleanTitle := strings.TrimSpace(syscall.UTF16ToString(buf))

			if cleanTitle == "" || cleanTitle == "Program Manager" || cleanTitle == "Windows Shell Experience Host" {
				return 1
			}

			var pid uint32
			procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

			key := fmt.Sprintf("%d:%s", pid, cleanTitle)
			if seenKey[key] {
				return 1
			}
			seenKey[key] = true

			list = append(list, WindowInfo{
				Handle:    hwnd,
				Title:     cleanTitle,
				ProcessID: pid,
				IsVisible: true,
			})
			return 1
		})

		procEnumWindows.Call(cb, 0)
	})

	return list, nil
}

// ExplorePath navega un directorio del sistema de archivos y construye un árbol en memoria
func (w *WindowsNavigator) ExplorePath(_ context.Context, targetPath string, maxDepth int) (*PathNode, error) {
	targetPath = ResolveUserPath(targetPath)
	if targetPath == "" || targetPath == "." {
		targetPath, _ = os.Getwd()
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo acceder a %s: %w", targetPath, err)
	}

	rootNode := &PathNode{
		Name:       filepath.Base(targetPath),
		Path:       targetPath,
		IsDir:      info.IsDir(),
		Size:       info.Size(),
		ModifiedAt: info.ModTime(),
		Extension:  filepath.Ext(targetPath),
	}

	if !info.IsDir() || maxDepth <= 0 {
		return rootNode, nil
	}

	var buildTree func(node *PathNode, depth int)
	buildTree = func(node *PathNode, depth int) {
		if depth <= 0 || !node.IsDir {
			return
		}

		entries, err := os.ReadDir(node.Path)
		if err != nil {
			return
		}

		node.ChildCount = len(entries)
		for _, e := range entries {
			// Saltar directorios ocultos o muy pesados por defecto
			name := e.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				continue
			}

			subPath := filepath.Join(node.Path, name)
			subInfo, _ := e.Info()
			var subSize int64
			var modTime time.Time
			if subInfo != nil {
				subSize = subInfo.Size()
				modTime = subInfo.ModTime()
			}

			child := &PathNode{
				Name:       name,
				Path:       subPath,
				IsDir:      e.IsDir(),
				Size:       subSize,
				ModifiedAt: modTime,
				Extension:  filepath.Ext(name),
			}

			if e.IsDir() && depth > 1 {
				buildTree(child, depth-1)
			}
			node.Children = append(node.Children, child)
		}
	}

	buildTree(rootNode, maxDepth)
	return rootNode, nil
}

// FindFiles busca archivos de forma nativa e instantánea por nombre o extensión
var fastSkipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "appdata": true,
	"$recycle.bin": true, "system volume information": true, "programdata": true,
	"recovery": true, "windows": true, ".vscode": true, ".cache": true,
	".npm": true, ".cargo": true, ".nuget": true, ".rustup": true, ".gradle": true,
	".m2": true, "venv": true, ".venv": true, "__pycache__": true, "dist": true,
	"build": true, "target": true, ".next": true, ".tauri": true, ".idea": true,
	".gemini": true, "temp": true, "tmp": true,
}

func matchFileName(nameLower, patternLower string) bool {
	if patternLower == "" || patternLower == "*" || patternLower == "*.*" {
		return true
	}
	if strings.HasPrefix(patternLower, "*.") {
		return strings.HasSuffix(nameLower, patternLower[1:])
	}
	if strings.HasPrefix(patternLower, ".") {
		return strings.HasSuffix(nameLower, patternLower)
	}
	if strings.Contains(nameLower, patternLower) || strings.Contains(nameLower, strings.ReplaceAll(patternLower, " ", "_")) {
		return true
	}
	if ok, _ := filepath.Match(patternLower, nameLower); ok {
		return true
	}
	words := strings.Fields(patternLower)
	if len(words) > 1 {
		allFound := true
		for _, w := range words {
			if len(w) > 2 && !strings.Contains(nameLower, w) {
				allFound = false
				break
			}
		}
		if allFound {
			return true
		}
	}
	return false
}

// FindFiles busca archivos de forma nativa e instantánea por nombre o extensión (< 100 ms)
func (w *WindowsNavigator) FindFiles(_ context.Context, rootDir string, pattern string, maxResults int) ([]PathNode, error) {
	if rootDir == "" || rootDir == "." {
		rootDir, _ = os.Getwd()
	}
	if maxResults <= 0 {
		maxResults = 50
	}

	var matches []PathNode
	patternLower := strings.ToLower(pattern)

	type dirItem struct {
		path  string
		depth int
	}

	queue := []dirItem{{path: rootDir, depth: 0}}
	maxDepth := 4

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		entries, err := os.ReadDir(item.path)
		if err != nil {
			continue
		}

		for _, d := range entries {
			name := d.Name()
			nameLower := strings.ToLower(name)

			if d.IsDir() {
				if fastSkipDirs[nameLower] || (strings.HasPrefix(name, ".") && name != "." && name != "..") {
					continue
				}
				if item.depth < maxDepth {
					queue = append(queue, dirItem{
						path:  filepath.Join(item.path, name),
						depth: item.depth + 1,
					})
				}
				continue
			}

			if matchFileName(nameLower, patternLower) {
				info, _ := d.Info()
				var size int64
				var modTime time.Time
				if info != nil {
					size = info.Size()
					modTime = info.ModTime()
				}

				matches = append(matches, PathNode{
					Name:       name,
					Path:       filepath.Join(item.path, name),
					IsDir:      false,
					Size:       size,
					ModifiedAt: modTime,
					Extension:  filepath.Ext(name),
				})

				if len(matches) >= maxResults {
					return matches, nil
				}
			}
		}
	}

	return matches, nil
}

// CreateDirectory crea una carpeta o árbol de carpetas nativamente
func (w *WindowsNavigator) CreateDirectory(_ context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("la ruta del directorio no puede estar vacía")
	}
	return os.MkdirAll(path, 0755)
}

// MoveItem mueve o renombra un archivo o carpeta
func (w *WindowsNavigator) MoveItem(_ context.Context, srcPath, dstPath string) error {
	srcPath = strings.TrimSpace(srcPath)
	dstPath = strings.TrimSpace(dstPath)
	if srcPath == "" || dstPath == "" {
		return fmt.Errorf("las rutas de origen y destino no pueden estar vacías")
	}

	dstInfo, err := os.Stat(dstPath)
	if err == nil && dstInfo.IsDir() {
		dstPath = filepath.Join(dstPath, filepath.Base(srcPath))
	} else {
		if parent := filepath.Dir(dstPath); parent != "" {
			_ = os.MkdirAll(parent, 0755)
		}
	}

	err = os.Rename(srcPath, dstPath)
	if err == nil {
		return nil
	}

	if copyErr := copyFileOrDir(srcPath, dstPath); copyErr != nil {
		return fmt.Errorf("fallo al mover elemento: %w", copyErr)
	}
	_ = os.RemoveAll(srcPath)
	return nil
}

// CopyItem copia un archivo o directorio recursivamente
func (w *WindowsNavigator) CopyItem(_ context.Context, srcPath, dstPath string) error {
	srcPath = strings.TrimSpace(srcPath)
	dstPath = strings.TrimSpace(dstPath)
	if srcPath == "" || dstPath == "" {
		return fmt.Errorf("las rutas de origen y destino no pueden estar vacías")
	}

	dstInfo, err := os.Stat(dstPath)
	if err == nil && dstInfo.IsDir() {
		dstPath = filepath.Join(dstPath, filepath.Base(srcPath))
	} else {
		if parent := filepath.Dir(dstPath); parent != "" {
			_ = os.MkdirAll(parent, 0755)
		}
	}

	return copyFileOrDir(srcPath, dstPath)
}

// DeleteItem elimina un archivo o carpeta (por defecto a la Papelera de reciclaje de Windows)
func (w *WindowsNavigator) DeleteItem(_ context.Context, path string, useRecycleBin bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("la ruta a eliminar no puede estar vacía")
	}

	if !useRecycleBin {
		return os.RemoveAll(path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	psCmd := fmt.Sprintf(`Add-Type -AssemblyName Microsoft.VisualBasic; if (Test-Path -LiteralPath '%s' -PathType Container) { [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteDirectory('%s', 'OnlyErrorDialogs', 'SendToRecycleBin') } else { [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile('%s', 'OnlyErrorDialogs', 'SendToRecycleBin') }`, absPath, absPath, absPath)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	if err := cmd.Run(); err == nil {
		return nil
	}

	return os.RemoveAll(path)
}

// OrganizeFolder clasifica automáticamente los archivos sueltos en carpetas temáticas
func (w *WindowsNavigator) OrganizeFolder(ctx context.Context, folderPath, strategy string) (*OrganizeResult, error) {
	folderPath = ResolveUserPath(folderPath)
	if folderPath == "" {
		return nil, fmt.Errorf("la ruta de la carpeta no puede estar vacía")
	}

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer la carpeta %s: %w", folderPath, err)
	}

	categoryMap := map[string][]string{
		"Documentos":                {".pdf", ".docx", ".doc", ".xlsx", ".xls", ".pptx", ".ppt", ".txt", ".csv", ".md", ".epub", ".rtf"},
		"Imágenes":                  {".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".webp", ".ico", ".psd", ".raw"},
		"Videos":                    {".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v"},
		"Música y Audio":            {".mp3", ".wav", ".flac", ".aac", ".ogg", ".m4a", ".wma"},
		"Instaladores y Ejecutables": {".exe", ".msi", ".iso", ".bat", ".cmd", ".ps1"},
		"Archivos Comprimidos":      {".zip", ".rar", ".7z", ".tar", ".gz", ".xz", ".bz2"},
		"Código y Proyectos":        {".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".json", ".yaml", ".yml", ".html", ".css", ".rs", ".cpp", ".c", ".java", ".sql"},
	}

	extToCategory := make(map[string]string)
	for cat, exts := range categoryMap {
		for _, ext := range exts {
			extToCategory[ext] = cat
		}
	}

	res := &OrganizeResult{
		SourcePath:     folderPath,
		CategorizedMap: make(map[string]int),
	}

	createdDirs := make(map[string]bool)

	for _, e := range entries {
		if e.IsDir() {
			continue // No mover carpetas ya existentes
		}

		res.TotalFiles++
		ext := strings.ToLower(filepath.Ext(e.Name()))
		category, ok := extToCategory[ext]
		if !ok {
			category = "Otros"
		}

		targetDir := filepath.Join(folderPath, category)
		if !createdDirs[targetDir] {
			if err := os.MkdirAll(targetDir, 0755); err == nil {
				createdDirs[targetDir] = true
				res.CreatedFolders = append(res.CreatedFolders, category)
			}
		}

		srcFile := filepath.Join(folderPath, e.Name())
		dstFile := filepath.Join(targetDir, e.Name())

		if err := w.MoveItem(ctx, srcFile, dstFile); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("error moviendo %s: %v", e.Name(), err))
		} else {
			res.MovedFiles++
			res.CategorizedMap[category]++
		}
	}

	return res, nil
}

// ForceForegroundWindow restaura y trae al primer plano una ventana,
// sorteando el bloqueo de foco de Windows 11 (Foreground Lockout).
func ForceForegroundWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	// 1. Simular pulsación de la tecla ALT (VK_MENU = 0x12) para desbloquear el permiso de primer plano
	procKeybdEvent.Call(0x12, 0, 0, 0) // KeyDown
	procKeybdEvent.Call(0x12, 0, 2, 0) // KeyUp (KEYEVENTF_KEYUP = 0x0002)

	// 2. Restaurar y traer arriba en la pila Z-Order
	procShowWindow.Call(hwnd, 9) // SW_RESTORE
	procBringWindowToTop.Call(hwnd)

	// 3. Conectar hilos si es necesario para transferir el foco
	hFore, _, _ := procGetForegroundWindow.Call()
	var forePID uint32
	foreTID, _, _ := procGetWindowThreadPID.Call(hFore, uintptr(unsafe.Pointer(&forePID)))
	curTID, _, _ := procGetCurrentThreadId.Call()

	if foreTID != 0 && curTID != 0 && foreTID != curTID {
		procAttachThreadInput.Call(curTID, foreTID, 1)
		procSetForegroundWindow.Call(hwnd)
		procAttachThreadInput.Call(curTID, foreTID, 0)
	} else {
		procSetForegroundWindow.Call(hwnd)
	}

	// 4. Invocación complementaria a SwitchToThisWindow
	procSwitchToThisWindow.Call(hwnd, 1)

	// 5. Método de activación garantizada mediante el Windows Shell COM (AppActivate)
	var winPID uint32
	procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&winPID)))
	if winPID > 0 {
		psCmd := fmt.Sprintf(`(New-Object -ComObject Wscript.Shell).AppActivate(%d)`, winPID)
		cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psCmd)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
		_ = cmd.Run()
	}
}

// LaunchApplication abre un ejecutable, archivo o URL nativamente usando el shell de Windows
func (w *WindowsNavigator) LaunchApplication(ctx context.Context, target string, args []string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("el objetivo a ejecutar no puede estar vacío")
	}

	// 1. Permitir que el nuevo proceso tome el foco principal
	procAllowSetForegroundWindow := modUser32.NewProc("AllowSetForegroundWindow")
	procAllowSetForegroundWindow.Call(uintptr(^uint32(0))) // ASFW_ANY

	// 2. Resolver dinámicamente el ejecutable o AUMID de la aplicación
	appInfo, err := ResolveAppExecutable(target)
	if err != nil {
		appInfo = &AppLaunchInfo{Command: target}
	}

	// Normalizar y expandir argumentos de ruta si los hay (ej: "crmgeofal" -> C:\Users\User\Documents\crmgeofal)
	var resolvedArgs []string
	for _, arg := range args {
		resolvedArgs = append(resolvedArgs, ResolveUserPath(arg))
	}

	launched := false
	if appInfo.IsAUMID && len(resolvedArgs) == 0 {
		cmd := exec.Command("explorer.exe", fmt.Sprintf("shell:AppsFolder\\%s", appInfo.Command))
		if err := cmd.Start(); err == nil {
			launched = true
		}
	}

	if !launched {
		// 1. Invocar prioritariamente a través de ShellExecuteW de shell32.dll
		// ShellExecute delega la apertura al shell de Windows Explorer, permitiendo que programas
		// que requieren elevación (como Taskmgr.exe, SystemSettings.exe o herramientas de administración)
		// se abran al primer intento sin error 740 (ERROR_ELEVATION_REQUIRED).
		verbPtr := uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("open")))
		filePtr := uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(appInfo.Command)))
		var argPtr uintptr
		if len(resolvedArgs) > 0 {
			argPtr = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(strings.Join(resolvedArgs, " "))))
		}
		const SW_SHOWNORMAL = 1
		ret, _, _ := procShellExecuteW.Call(0, verbPtr, filePtr, argPtr, 0, SW_SHOWNORMAL)
		if ret > 32 {
			launched = true
		} else {
			// 2. Ejecución directa con CreateProcess (fallback para herramientas CLI o scripts sin shell verb)
			directCmd := exec.Command(appInfo.Command, resolvedArgs...)
			if errDirect := directCmd.Start(); errDirect == nil {
				launched = true
			} else {
				// 3. Broker de fallback con cmd.exe si la ejecución directa falló
				brokerCtx, brokerCancel := context.WithTimeout(ctx, 3*time.Second)
				defer brokerCancel()

				cmdArgs := []string{"/c", "start", "", appInfo.Command}
				cmdArgs = append(cmdArgs, resolvedArgs...)
				cmd := exec.CommandContext(brokerCtx, "cmd.exe", cmdArgs...)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("error ejecutando aplicación %s (%s): %v", target, appInfo.Command, err)
				}
				launched = true
			}
		}
	}

	// 3. Localizar la ventana recién abierta y asegurar foco en primer plano de forma asíncrona
	go func(targetName, resolvedCmd string) {
		cleanTarget := strings.ToLower(filepath.Base(targetName))
		cleanTarget = strings.TrimSuffix(cleanTarget, ".exe")
		cleanResolved := strings.ToLower(filepath.Base(resolvedCmd))
		cleanResolved = strings.TrimSuffix(cleanResolved, ".exe")

		for i := 0; i < 15; i++ {
			time.Sleep(150 * time.Millisecond)
			var foundHwnd uintptr

			withInteractiveDesktop(func() {
				cb := syscall.NewCallback(func(h uintptr, _ uintptr) uintptr {
					vis, _, _ := procIsWindowVisible.Call(h)
					if vis == 0 {
						return 1
					}
					l, _, _ := procGetWindowTextLength.Call(h)
					if l > 0 {
						b := make([]uint16, l+1)
						procGetWindowTextW.Call(h, uintptr(unsafe.Pointer(&b[0])), l+1)
						title := syscall.UTF16ToString(b)
						lower := strings.ToLower(title)
						if strings.Contains(lower, cleanTarget) ||
							strings.Contains(lower, cleanResolved) ||
							(cleanTarget == "notepad" && strings.Contains(lower, "bloc de notas")) ||
							(cleanTarget == "calc" && strings.Contains(lower, "calculadora")) {
							foundHwnd = h
							return 0
						}
					}
					return 1
				})
				procEnumWindows.Call(cb, 0)
			})

			if foundHwnd != 0 {
				ForceForegroundWindow(foundHwnd)
				break
			}
		}
	}(target, appInfo.Command)

	return nil
}

// FocusWindow trae al frente y restaura una ventana de Windows
func (w *WindowsNavigator) FocusWindow(_ context.Context, hwnd uintptr) error {
	if hwnd == 0 {
		return fmt.Errorf("identificador de ventana (hwnd) inválido")
	}
	ForceForegroundWindow(hwnd)
	return nil
}

// CloseWindow envía el mensaje Win32 WM_CLOSE (0x0010) y SC_CLOSE a la ventana especificada para cerrarla elegantemente.
func (w *WindowsNavigator) CloseWindow(ctx context.Context, hwnd uintptr) error {
	if hwnd == 0 {
		return fmt.Errorf("handle de ventana inválido (0)")
	}

	procIsWindow := modUser32.NewProc("IsWindow")
	procPostMessageW := modUser32.NewProc("PostMessageW")
	const WM_CLOSE = 0x0010
	const WM_SYSCOMMAND = 0x0112
	const SC_CLOSE = 0xF060

	var pid uint32
	procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

	var posted bool
	withInteractiveDesktop(func() {
		isWin, _, _ := procIsWindow.Call(hwnd)
		if isWin == 0 {
			return
		}
		r1, _, _ := procPostMessageW.Call(hwnd, WM_CLOSE, 0, 0)
		if r1 != 0 {
			posted = true
		}
		procPostMessageW.Call(hwnd, WM_SYSCOMMAND, SC_CLOSE, 0)
	})

	if !posted {
		if pid > 0 {
			return w.KillProcess(ctx, pid, true)
		}
		return fmt.Errorf("no se pudo cerrar la ventana (HWND: %d)", hwnd)
	}

	return nil
}

type winRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// TileWindow acomoda una ventana en mosaico según el layout solicitado ("left", "right", "maximize", "minimize", "restore", "center", "show_desktop")
func (w *WindowsNavigator) TileWindow(ctx context.Context, hwnd uintptr, layout string) error {
	layout = strings.ToLower(strings.TrimSpace(layout))
	if layout == "" {
		layout = "maximize"
	}

	if layout == "show_desktop" || layout == "minimize_all" {
		psCmd := `(New-Object -ComObject Shell.Application).MinimizeAll()`
		_ = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Run()
		return nil
	}

	if hwnd == 0 {
		return fmt.Errorf("identificador de ventana (hwnd) inválido")
	}

	const (
		SW_MAXIMIZE     = 3
		SW_MINIMIZE     = 6
		SW_RESTORE      = 9
		SPI_GETWORKAREA = 0x0030
	)

	var workArea winRect
	withInteractiveDesktop(func() {
		procSystemParametersInfoW.Call(SPI_GETWORKAREA, 0, uintptr(unsafe.Pointer(&workArea)), 0)
	})

	workWidth := workArea.Right - workArea.Left
	workHeight := workArea.Bottom - workArea.Top
	if workWidth <= 0 || workHeight <= 0 {
		workWidth = 1920
		workHeight = 1080
	}

	switch layout {
	case "maximize", "maximizar":
		procShowWindow.Call(hwnd, SW_MAXIMIZE)
		ForceForegroundWindow(hwnd)
		return nil

	case "minimize", "minimizar":
		procShowWindow.Call(hwnd, SW_MINIMIZE)
		return nil

	case "restore", "restaurar":
		procShowWindow.Call(hwnd, SW_RESTORE)
		ForceForegroundWindow(hwnd)
		return nil

	case "left", "izquierda":
		procShowWindow.Call(hwnd, SW_RESTORE)
		x := workArea.Left
		y := workArea.Top
		wHalf := workWidth / 2
		procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(wHalf), uintptr(workHeight), 1)
		ForceForegroundWindow(hwnd)
		return nil

	case "right", "derecha":
		procShowWindow.Call(hwnd, SW_RESTORE)
		x := workArea.Left + (workWidth / 2)
		y := workArea.Top
		wHalf := workWidth / 2
		procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(wHalf), uintptr(workHeight), 1)
		ForceForegroundWindow(hwnd)
		return nil

	case "center", "centro":
		procShowWindow.Call(hwnd, SW_RESTORE)
		wWidth := int32(float64(workWidth) * 0.7)
		wHeight := int32(float64(workHeight) * 0.7)
		x := workArea.Left + (workWidth-wWidth)/2
		y := workArea.Top + (workHeight-wHeight)/2
		procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(wWidth), uintptr(wHeight), 1)
		ForceForegroundWindow(hwnd)
		return nil

	default:
		return fmt.Errorf("layout '%s' no reconocido. Opciones válidas: left, right, maximize, minimize, restore, center, show_desktop", layout)
	}
}

// KillProcess finaliza un proceso por su Process ID con fallback multinivel (taskkill -> Stop-Process -> WMI/CIM Terminate)
func (w *WindowsNavigator) KillProcess(ctx context.Context, pid uint32, force bool) error {
	if pid == 0 {
		return fmt.Errorf("PID inválido")
	}

	// 1. Intento primario con taskkill nativo
	args := []string{"/PID", fmt.Sprintf("%d", pid)}
	if force {
		args = append(args, "/F", "/T")
	}
	cmd := exec.CommandContext(ctx, "taskkill", args...)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// 2. Intento secundario: PowerShell Stop-Process
	psCmd := fmt.Sprintf("Stop-Process -Id %d -Force -ErrorAction SilentlyContinue", pid)
	_ = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Run()

	// 3. Intento terciario de alta resiliencia: WMI / CIM Terminate (ejecutado con permisos del subsistema WMI)
	// Esto permite finalizar procesos con nivel de integridad alto / UAC (como Taskmgr.exe) sin requerir elevación manual.
	wmiCmd := fmt.Sprintf("Get-CimInstance Win32_Process -Filter 'ProcessId = %d' | Invoke-CimMethod -MethodName Terminate", pid)
	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", wmiCmd).CombinedOutput()
	if err == nil && strings.Contains(string(out), "0") {
		return nil
	}

	// Comprobación final si el proceso ya no existe
	checkCmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", fmt.Sprintf("Get-Process -Id %d -ErrorAction SilentlyContinue", pid))
	if outCheck, _ := checkCmd.Output(); len(strings.TrimSpace(string(outCheck))) == 0 {
		return nil
	}

	return fmt.Errorf("no se pudo finalizar el proceso PID %d (incluso con fallback WMI)", pid)
}

func copyFileOrDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			srcSub := filepath.Join(src, entry.Name())
			dstSub := filepath.Join(dst, entry.Name())
			if err := copyFileOrDir(srcSub, dstSub); err != nil {
				return err
			}
		}
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// DetectDialogs inspecciona ventanas emergentes, diálogos modales (#32770) y mensajes de error
func (w *WindowsNavigator) DetectDialogs(_ context.Context, appFilter string) ([]DialogInfo, error) {
	var dialogs []DialogInfo
	filterLower := strings.ToLower(strings.TrimSpace(appFilter))
	seen := make(map[uintptr]bool)

	withInteractiveDesktop(func() {
		cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
			if seen[hwnd] {
				return 1
			}
			seen[hwnd] = true

			vis, _, _ := procIsWindowVisible.Call(hwnd)
			if vis == 0 {
				return 1
			}

			var classBuf [256]uint16
			procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&classBuf[0])), 256)
			class := strings.TrimSpace(syscall.UTF16ToString(classBuf[:]))

			var titleBuf [512]uint16
			procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&titleBuf[0])), 512)
			title := strings.TrimSpace(syscall.UTF16ToString(titleBuf[:]))

			var pid uint32
			procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
			procName := getProcessName(pid)

			isStandardDialog := class == "#32770"
			titleLower := strings.ToLower(title)
			procLower := strings.ToLower(procName)

			isErrorIndicator := strings.Contains(titleLower, "error") ||
				strings.Contains(titleLower, "advertencia") ||
				strings.Contains(titleLower, "warning") ||
				strings.Contains(titleLower, "alerta") ||
				strings.Contains(titleLower, "alert") ||
				strings.Contains(titleLower, "falló") ||
				strings.Contains(titleLower, "failed")

			matchesFilter := filterLower != "" && (strings.Contains(titleLower, filterLower) || strings.Contains(procLower, filterLower))

			// Si no es un diálogo estándar ni indicador de error ni coincide con el filtro, omitir
			if !isStandardDialog && !isErrorIndicator && !matchesFilter {
				return 1
			}

			// Extraer controles hijos (Static, Edit, Button)
			var messageParts []string
			var buttons []string

			childCb := syscall.NewCallback(func(childHwnd uintptr, _ uintptr) uintptr {
				var cClassBuf [128]uint16
				procGetClassNameW.Call(childHwnd, uintptr(unsafe.Pointer(&cClassBuf[0])), 128)
				cClass := strings.TrimSpace(syscall.UTF16ToString(cClassBuf[:]))

				var cTextBuf [2048]uint16
				procGetWindowTextW.Call(childHwnd, uintptr(unsafe.Pointer(&cTextBuf[0])), 2048)
				cText := strings.TrimSpace(syscall.UTF16ToString(cTextBuf[:]))

				if cText == "" {
					return 1
				}

				switch strings.ToLower(cClass) {
				case "button":
					buttons = append(buttons, cText)
				case "static", "edit":
					messageParts = append(messageParts, cText)
				default:
					if isStandardDialog && len(cText) > 1 {
						messageParts = append(messageParts, cText)
					}
				}
				return 1
			})
			procEnumChildWindows.Call(hwnd, childCb, 0)

			fullMessage := strings.Join(messageParts, " ")
			fullMessage = strings.TrimSpace(fullMessage)

			// Si no hay mensaje ni título relevante, saltar
			if fullMessage == "" && title == "" {
				return 1
			}

			combinedText := strings.ToLower(title + " " + fullMessage)
			isErr := false
			severity := "INFO"

			errorKeywords := []string{
				"error", "dañado", "no pudo abrir", "no se pudo", "falló", "fallo", "failed",
				"not supported", "no admitido", "no compatible", "corrupt", "corrupto",
				"cannot open", "could not open", "no responde", "not responding", "denied",
				"denegado", "invalid", "inválido", "exception", "excepción", "fatal",
			}
			warningKeywords := []string{
				"warning", "advertencia", "alerta", "precaución", "caution", "confirmar",
			}

			for _, kw := range errorKeywords {
				if strings.Contains(combinedText, kw) {
					isErr = true
					severity = "ERROR"
					break
				}
			}
			if !isErr {
				for _, kw := range warningKeywords {
					if strings.Contains(combinedText, kw) {
						severity = "WARNING"
						break
					}
				}
			}

			// Aplicar filtro si se especificó
			if filterLower != "" {
				if !strings.Contains(titleLower, filterLower) &&
					!strings.Contains(procLower, filterLower) &&
					!strings.Contains(strings.ToLower(fullMessage), filterLower) {
					return 1
				}
			}

			dialogs = append(dialogs, DialogInfo{
				Handle:      hwnd,
				Title:       title,
				ClassName:   class,
				ProcessID:   pid,
				ProcessName: procName,
				Message:     fullMessage,
				Buttons:     buttons,
				IsError:     isErr,
				Severity:    severity,
			})

			return 1
		})

		procEnumWindows.Call(cb, 0)
	})

	return dialogs, nil
}

