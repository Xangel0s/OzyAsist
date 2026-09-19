package system

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CompressZip empaqueta uno o más archivos y carpetas en un archivo .zip estándar
func CompressZip(srcPaths []string, destZip string) error {
	if len(srcPaths) == 0 {
		return fmt.Errorf("se requiere al menos una ruta de origen para comprimir")
	}
	if destZip == "" {
		return fmt.Errorf("ruta de archivo destino .zip requerida")
	}

	destZip = ResolveUserPath(destZip)
	if !strings.HasSuffix(strings.ToLower(destZip), ".zip") {
		destZip += ".zip"
	}

	if err := os.MkdirAll(filepath.Dir(destZip), 0755); err != nil {
		return fmt.Errorf("no se pudo crear directorio contenedor: %w", err)
	}

	zipFile, err := os.Create(destZip)
	if err != nil {
		return fmt.Errorf("no se pudo crear el archivo zip: %w", err)
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for _, src := range srcPaths {
		resolvedSrc := ResolveUserPath(src)
		info, err := os.Stat(resolvedSrc)
		if err != nil {
			return fmt.Errorf("no se pudo acceder a '%s': %w", src, err)
		}

		baseDir := filepath.Dir(resolvedSrc)
		if info.IsDir() {
			err = filepath.Walk(resolvedSrc, func(path string, fInfo os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}

				// Obtener ruta relativa dentro del zip
				relPath, err := filepath.Rel(baseDir, path)
				if err != nil {
					return err
				}
				// Normalizar separadores a '/' para compatibilidad universal con ZIP
				zipRelPath := filepath.ToSlash(relPath)

				header, err := zip.FileInfoHeader(fInfo)
				if err != nil {
					return err
				}
				header.Name = zipRelPath
				if fInfo.IsDir() {
					header.Name += "/"
				} else {
					header.Method = zip.Deflate
				}

				writer, err := archive.CreateHeader(header)
				if err != nil {
					return err
				}

				if !fInfo.IsDir() {
					file, err := os.Open(path)
					if err != nil {
						return err
					}
					defer file.Close()
					if _, err := io.Copy(writer, file); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("error al comprimir directorio '%s': %w", src, err)
			}
		} else {
			// Archivo individual
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name = filepath.Base(resolvedSrc)
			header.Method = zip.Deflate

			writer, err := archive.CreateHeader(header)
			if err != nil {
				return err
			}

			file, err := os.Open(resolvedSrc)
			if err != nil {
				return err
			}
			defer file.Close()
			if _, err := io.Copy(writer, file); err != nil {
				return err
			}
		}
	}

	return nil
}

// ExtractZip descomprime de forma segura un archivo .zip en el directorio destino
// Incluye protección estricta contra ataques Zip Slip (path traversal)
func ExtractZip(zipPath string, destDir string) ([]string, error) {
	zipPath = ResolveUserPath(zipPath)
	destDir = ResolveUserPath(destDir)

	if destDir == "" {
		destDir = strings.TrimSuffix(zipPath, filepath.Ext(zipPath))
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir el archivo zip '%s': %w", zipPath, err)
	}
	defer reader.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("no se pudo crear directorio destino '%s': %w", destDir, err)
	}

	cleanDest := filepath.Clean(destDir)
	var extractedFiles []string

	for _, file := range reader.File {
		// Protección contra Zip Slip
		targetPath := filepath.Join(cleanDest, file.Name)
		cleanTarget := filepath.Clean(targetPath)

		if !strings.HasPrefix(cleanTarget, cleanDest+string(filepath.Separator)) && cleanTarget != cleanDest {
			return nil, fmt.Errorf("archivo inseguro detectado en zip (Zip Slip vulnerable): %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0755); err != nil {
				return nil, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return nil, err
		}

		outFile, err := os.OpenFile(cleanTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return nil, fmt.Errorf("no se pudo crear archivo '%s': %w", cleanTarget, err)
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return nil, fmt.Errorf("no se pudo leer entrada del zip '%s': %w", file.Name, err)
		}

		_, copyErr := io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if copyErr != nil {
			return nil, fmt.Errorf("error al escribir '%s': %w", cleanTarget, copyErr)
		}

		extractedFiles = append(extractedFiles, cleanTarget)
	}

	return extractedFiles, nil
}
