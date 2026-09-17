package memory

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func BuildSystemIndex() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("SystemIndex panic recuperado: %v", r)
			}
		}()
		log.Println("Iniciando escaneo de aplicaciones en segundo plano para ChromaDB...")

		// Escanear Start Menu (All Users)
		startMenuPath := `C:\ProgramData\Microsoft\Windows\Start Menu\Programs`
		scanAndIndex(startMenuPath, "StartMenu")

		log.Println("Escaneo de aplicaciones finalizado.")
	}()
}

func scanAndIndex(root, source string) {
	c := GetCaps2()

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Ignorar errores de acceso
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".lnk" || ext == ".exe" {
			name := strings.TrimSuffix(info.Name(), ext)
			
			content := fmt.Sprintf("Aplicación instalada: %s. Ruta: %s", name, path)
			
			entry := Caps2Entry{
				ID:        "sys-" + fmt.Sprintf("%d", info.Size()) + "-" + name,
				Content:   content,
				Source:    "system_index",
				SourceID:  path,
				UserID:    "system",
				ProjectID: "system",
			}

			// Solo indexamos si el contenido tiene un tamaño prudente
			if len(content) < 500 {
				_ = c.Store(entry)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Error escaneando %s: %v", root, err)
	}
}
