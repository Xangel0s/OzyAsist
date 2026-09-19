package system

import (
	"context"
	"time"
)

// DesktopItemKind clasifica el tipo de elemento en el escritorio
type DesktopItemKind string

const (
	ItemKindShortcut   DesktopItemKind = "shortcut"   // Acceso directo (.lnk / .url)
	ItemKindFile       DesktopItemKind = "file"       // Archivo de datos / documento / código
	ItemKindFolder     DesktopItemKind = "folder"     // Directorio
	ItemKindSystemIcon DesktopItemKind = "system_icon" // Elemento virtual de Windows (Papelera, Red, etc.)
)

// DesktopItem representa un icono o elemento visible en el escritorio
type DesktopItem struct {
	Name         string          `json:"name"`
	Path         string          `json:"path"`
	Kind         DesktopItemKind `json:"kind"`
	Target       string          `json:"target,omitempty"`        // Destino real si es acceso directo (.lnk)
	Size         int64           `json:"size_bytes,omitempty"`
	ModifiedAt   time.Time       `json:"modified_at,omitempty"`
	IsPublic     bool            `json:"is_public"`              // true si proviene de C:\Users\Public\Desktop
}

// InstalledApp representa una aplicación o software instalado en el sistema operativo
type InstalledApp struct {
	DisplayName     string `json:"display_name"`
	Publisher       string `json:"publisher,omitempty"`
	DisplayVersion  string `json:"display_version,omitempty"`
	InstallLocation string `json:"install_location,omitempty"`
	UninstallString string `json:"uninstall_string,omitempty"`
	InstallDate     string `json:"install_date,omitempty"`
	RegistrySource  string `json:"registry_source,omitempty"` // HKLM64, HKLM32, HKCU
}

// WindowInfo representa una ventana visible abierta en la pantalla
type WindowInfo struct {
	Handle    uintptr `json:"handle"`
	Title     string  `json:"title"`
	ProcessID uint32  `json:"process_id"`
	IsVisible bool    `json:"is_visible"`
}

// PathNode es un nodo del árbol VFS para navegación rápida de carpetas
type PathNode struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	IsDir      bool        `json:"is_dir"`
	Size       int64       `json:"size_bytes"`
	ModifiedAt time.Time   `json:"modified_at"`
	Extension  string      `json:"extension,omitempty"`
	Children   []*PathNode `json:"children,omitempty"`
	ChildCount int         `json:"child_count,omitempty"`
}

// KnownFolders contiene las rutas de carpetas estándar del usuario
type KnownFolders struct {
	UserDesktop   string `json:"user_desktop"`
	PublicDesktop string `json:"public_desktop"`
	Documents     string `json:"documents"`
	Downloads     string `json:"downloads"`
	Pictures      string `json:"pictures"`
	Videos        string `json:"videos"`
	AppDataLocal  string `json:"app_data_local"`
	AppDataRoam   string `json:"app_data_roaming"`
	ProgramFiles  string `json:"program_files"`
	ProgramFilesX86 string `json:"program_files_x86"`
}

// OrganizeResult contiene el resumen de una operación de auto-organización de carpetas
type OrganizeResult struct {
	SourcePath      string            `json:"source_path"`
	TotalFiles      int               `json:"total_files"`
	MovedFiles      int               `json:"moved_files"`
	CreatedFolders  []string          `json:"created_folders"`
	CategorizedMap  map[string]int    `json:"categorized_map"` // Ej: {"Documentos": 5, "Imágenes": 12}
	Errors          []string          `json:"errors,omitempty"`
}

// OSNavigator define las capacidades de navegación, inspección y manipulación nativa del sistema
type OSNavigator interface {
	// Inspección y Consulta
	GetDesktopItems(ctx context.Context) ([]DesktopItem, error)
	GetInstalledSoftware(ctx context.Context, filter string) ([]InstalledApp, error)
	GetKnownFolders(ctx context.Context) (KnownFolders, error)
	GetActiveWindows(ctx context.Context) ([]WindowInfo, error)
	ExplorePath(ctx context.Context, targetPath string, maxDepth int) (*PathNode, error)
	FindFiles(ctx context.Context, rootDir string, pattern string, maxResults int) ([]PathNode, error)

	// Manipulación del Sistema de Archivos
	CreateDirectory(ctx context.Context, path string) error
	MoveItem(ctx context.Context, srcPath string, dstPath string) error
	CopyItem(ctx context.Context, srcPath string, dstPath string) error
	DeleteItem(ctx context.Context, path string, useRecycleBin bool) error
	OrganizeFolder(ctx context.Context, folderPath string, strategy string) (*OrganizeResult, error)

	// Control de Aplicaciones, Ventanas y Procesos
	LaunchApplication(ctx context.Context, target string, args []string) error
	FocusWindow(ctx context.Context, hwnd uintptr) error
	CloseWindow(ctx context.Context, hwnd uintptr) error
	KillProcess(ctx context.Context, pid uint32, force bool) error
	DetectDialogs(ctx context.Context, appFilter string) ([]DialogInfo, error)
}

// DialogInfo describe un cuadro de diálogo emergente o modal detectado en el sistema
type DialogInfo struct {
	Handle      uintptr  `json:"handle"`
	Title       string   `json:"title"`
	ClassName   string   `json:"class_name"`
	ProcessID   uint32   `json:"process_id"`
	ProcessName string   `json:"process_name,omitempty"`
	Message     string   `json:"message"`
	Buttons     []string `json:"buttons,omitempty"`
	IsError     bool     `json:"is_error"`
	Severity    string   `json:"severity"` // "ERROR", "WARNING", "INFO"
}
