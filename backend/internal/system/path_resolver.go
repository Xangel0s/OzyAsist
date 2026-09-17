package system

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// ResolveUserPath normaliza y resuelve expresiones en lenguaje natural, alias,
// variables de entorno y tildes a rutas absolutas canónicas de Windows.
func ResolveUserPath(input string) string {
	clean := strings.TrimSpace(input)
	if clean == "" {
		return ""
	}

	// 1. Quitar comillas accidentales
	clean = strings.Trim(clean, `"'` + "`")

	// 2. Obtener la ruta base del usuario actual
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		if usr, err := user.Current(); err == nil && usr.HomeDir != "" {
			userProfile = usr.HomeDir
		} else {
			userProfile = filepath.Join("C:\\Users", os.Getenv("USERNAME"))
		}
	}

	lower := strings.ToLower(clean)

	// Preservar URLs intactas sin convertirlas a rutas de disco
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "mailto:") {
		return clean
	}
	if strings.HasPrefix(lower, "www.") {
		return "https://" + clean
	}

	// 3. Mapeo de alias en lenguaje natural común (español e inglés)
	switch lower {
	case "mis descargas", "carpeta de descargas", "descargas", "downloads", "download":
		return filepath.Join(userProfile, "Downloads")

	case "mi escritorio", "el escritorio", "escritorio", "desktop":
		return filepath.Join(userProfile, "Desktop")

	case "mis documentos", "carpeta de documentos", "documentos", "documents", "docs":
		return filepath.Join(userProfile, "Documents")

	case "mis imágenes", "mis imagenes", "mis fotos", "fotos", "imágenes", "imagenes", "pictures":
		return filepath.Join(userProfile, "Pictures")

	case "mis videos", "carpeta de videos", "videos", "movies":
		return filepath.Join(userProfile, "Videos")

	case "mis proyectos", "carpeta de proyectos", "proyectos", "projects":
		candidate := filepath.Join(userProfile, "Projects")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		candidate2 := filepath.Join(userProfile, "Documents", "Projects")
		if _, err := os.Stat(candidate2); err == nil {
			return candidate2
		}
		return candidate

	case "mis carpetas", "carpetas", "mi carpeta":
		downloads := filepath.Join(userProfile, "Downloads")
		if _, err := os.Stat(downloads); err == nil {
			return downloads
		}
		return filepath.Join(userProfile, "Desktop")

	case "temp", "temporal", "%temp%":
		return os.TempDir()
	}

	// 4. Prefijos comunes de lenguaje natural (ej: "en mis descargas", "documentos/...")
	prefixes := []string{
		"en mis descargas", "en las descargas", "en descargas",
		"en mi escritorio", "en el escritorio",
		"en mis documentos", "en los documentos",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			remainder := strings.TrimPrefix(lower, p)
			remainder = strings.TrimLeft(remainder, "/\\ ")
			base := ResolveUserPath(p[3:])
			if remainder != "" {
				return filepath.Join(base, remainder)
			}
			return base
		}
	}

	if strings.HasPrefix(lower, "documentos/") || strings.HasPrefix(lower, `documentos\`) {
		return filepath.Join(userProfile, "Documents", clean[11:])
	}
	if strings.HasPrefix(lower, "descargas/") || strings.HasPrefix(lower, `descargas\`) {
		return filepath.Join(userProfile, "Downloads", clean[10:])
	}
	if strings.HasPrefix(lower, "escritorio/") || strings.HasPrefix(lower, `escritorio\`) {
		return filepath.Join(userProfile, "Desktop", clean[11:])
	}

	// 5. Prefijos de proyectos o carpetas (ej: "el proyecto crmgeofal", "carpeta documentos")
	for _, projPrefix := range []string{"el proyecto ", "proyecto ", "la carpeta ", "carpeta "} {
		if strings.HasPrefix(lower, projPrefix) {
			clean = strings.TrimSpace(clean[len(projPrefix):])
			lower = strings.ToLower(clean)
			break
		}
	}

	// 5.1. Sufijos de ubicación natural (ej: "crmgeofal de documentos", "proyecto x en documentos")
	locationSuffixes := []struct {
		suffix string
		folder string
	}{
		{" de documentos", "Documents"},
		{" en documentos", "Documents"},
		{" de descargas", "Downloads"},
		{" en descargas", "Downloads"},
		{" del escritorio", "Desktop"},
		{" en escritorio", "Desktop"},
		{" en el escritorio", "Desktop"},
	}
	for _, ls := range locationSuffixes {
		if strings.HasSuffix(lower, ls.suffix) {
			cleanSub := strings.TrimSpace(clean[:len(clean)-len(ls.suffix)])
			cand := filepath.Join(userProfile, ls.folder, cleanSub)
			if _, err := os.Stat(cand); err == nil {
				return cand
			}
			clean = cleanSub
			lower = strings.ToLower(clean)
			break
		}
	}

	// 6. Corregir nombres de usuario alucinados por LLMs (ej: C:\Users\Usuario\... -> C:\Users\User\...)
	normalizedClean := filepath.Clean(clean)
	lowerNorm := strings.ToLower(normalizedClean)
	if strings.HasPrefix(lowerNorm, "c:\\users\\") {
		parts := strings.Split(normalizedClean, "\\")
		if len(parts) >= 3 {
			hallucinatedUser := parts[2]
			currentUser := os.Getenv("USERNAME")
			if currentUser != "" && !strings.EqualFold(hallucinatedUser, currentUser) && !strings.EqualFold(hallucinatedUser, "public") {
				rem := filepath.Join(parts[3:]...)
				return filepath.Join(userProfile, rem)
			}
		}
	}

	// 7. Expansión de tilde ~
	if strings.HasPrefix(clean, "~/") || strings.HasPrefix(clean, `~\`) {
		return filepath.Join(userProfile, clean[2:])
	} else if clean == "~" {
		return userProfile
	}

	// 8. Expansión de variables de entorno de Windows (%USERPROFILE%, %TEMP%, etc.)
	clean = os.ExpandEnv(clean)

	// 9. Si es una ruta relativa o un nombre simple (ej: "crmgeofal", "archivo.xlsx"):
	if !filepath.IsAbs(clean) && !isWindowsDrivePath(clean) {
		// A. Verificar si existe respecto al directorio actual (CWD)
		//    Solo aceptar si el CWD NO es dentro del directorio del servidor (evita
		//    resolver rutas del proceso Go como rutas de usuario).
		cwd, _ := os.Getwd()
		cwdIsServerDir := strings.Contains(strings.ToLower(cwd), strings.ToLower("Ozyasist"))
		if !cwdIsServerDir {
			if abs, err := filepath.Abs(clean); err == nil {
				if _, err := os.Stat(abs); err == nil {
					return abs
				}
			}
		}

		// A.1 Si el CWD ya termina con el primer componente de clean
		cwdBase := strings.ToLower(filepath.Base(cwd))
		lowerClean := strings.ToLower(clean)
		if strings.HasPrefix(lowerClean, cwdBase+"/") || strings.HasPrefix(lowerClean, cwdBase+`\`) {
			trimmed := clean[len(cwdBase)+1:]
			if abs, err := filepath.Abs(trimmed); err == nil {
				if _, err := os.Stat(abs); err == nil {
					return abs
				}
			}
		}

		// B. Búsqueda inteligente en carpetas comunes del usuario.
		//    El orden importa: Desktop primero (archivos de trabajo del usuario),
		//    luego Documents, Projects, raíz del perfil y Downloads.
		commonDirs := []string{
			filepath.Join(userProfile, "Desktop", clean),
			filepath.Join(userProfile, "Documents", clean),
			filepath.Join(userProfile, "Projects", clean),
			filepath.Join(userProfile, clean),
			filepath.Join(userProfile, "Downloads", clean),
		}
		for _, cand := range commonDirs {
			if _, err := os.Stat(cand); err == nil {
				return cand
			}
		}

		// C. Fallback: si no existe en ningún lado, asumir Documents como
		//    ubicación predeterminada (más seguro que CWD del servidor).
		return filepath.Join(userProfile, "Documents", clean)
	}

	return filepath.Clean(clean)
}

func isWindowsDrivePath(p string) bool {
	if len(p) >= 2 && p[1] == ':' {
		return true
	}
	return false
}
