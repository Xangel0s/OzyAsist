package agent

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateDocxReport(t *testing.T) {
	tempDir := t.TempDir()
	outDocx := filepath.Join(tempDir, "propuesta_comercial.docx")

	opts := DocxReportOptions{
		Title:    "Propuesta Comercial — Sistema de Gestión Geofal",
		Subtitle: "Desarrollo de Software y Automatización con OzyAssist",
		Author:   "OzyAssist AI Team",
		Sections: []DocxSection{
			{
				Title:   "1. Objetivos del Proyecto",
				Content: "El objetivo es diseñar e implementar una solución integral de cowork y gestión de expedientes.",
				Bullets: []string{
					"Reducción de tiempos operativos en un 40%",
					"Integración directa con bases de datos locales y APIs",
					"Seguridad de datos con cifrado de punto a punto",
				},
			},
			{
				Title:   "2. Cronograma y Entregables",
				Content: "A continuación se presentan los módulos contratados y sus plazos estimados.",
			},
		},
		Table: &DocxTable{
			Headers: []string{"Fase", "Entregable", "Semanas", "Estado"},
			Rows: [][]string{
				{"Fase 1", "Arquitectura y Wireframes", "2", "Completado"},
				{"Fase 2", "Backend en Go y Base de Datos", "3", "En Progreso"},
				{"Fase 3", "Integración TUI y Testing", "2", "Planificado"},
			},
		},
	}

	err := GenerateDocxReport(outDocx, opts)
	if err != nil {
		t.Fatalf("GenerateDocxReport failed: %v", err)
	}

	// Verificar que el archivo generado es un ZIP válido con las entradas OpenXML estándar
	zr, err := zip.OpenReader(outDocx)
	if err != nil {
		t.Fatalf("no se pudo abrir el archivo .docx como paquete ZIP: %v", err)
	}
	defer zr.Close()

	hasDocumentXML := false
	hasContentTypes := false

	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			hasDocumentXML = true
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("error abriendo word/document.xml: %v", err)
			}
			content, _ := io.ReadAll(rc)
			rc.Close()

			if !strings.Contains(string(content), "Propuesta Comercial") {
				t.Errorf("document.xml no contiene el título esperado")
			}
		}
		if f.Name == "[Content_Types].xml" {
			hasContentTypes = true
		}
	}

	if !hasDocumentXML {
		t.Errorf("el archivo .docx no contiene word/document.xml")
	}
	if !hasContentTypes {
		t.Errorf("el archivo .docx no contiene [Content_Types].xml")
	}
}

func TestGenerateDocxReport_MultiPage(t *testing.T) {
	tempDir := t.TempDir()
	outDocx := filepath.Join(tempDir, "manual_arquitectura_10p.docx")

	var sections []DocxSection
	for i := 1; i <= 10; i++ {
		sections = append(sections, DocxSection{
			Title:     strings.Repeat("A", 10),
			Content:   "Contenido de prueba para página.",
			PageBreak: i > 1,
		})
	}

	opts := DocxReportOptions{
		Title:       "Manual de Arquitectura de Sistemas Distribuidos",
		Author:      "OzyAssist Testing Suite",
		TargetPages: 10,
		Sections:    sections,
	}

	err := GenerateDocxReport(outDocx, opts)
	if err != nil {
		t.Fatalf("GenerateDocxReport multi-página falló: %v", err)
	}

	zr, err := zip.OpenReader(outDocx)
	if err != nil {
		t.Fatalf("no se pudo abrir el zip generado: %v", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("error abriendo word/document.xml: %v", err)
			}
			content, _ := io.ReadAll(rc)
			rc.Close()

			brCount := strings.Count(string(content), `w:br w:type="page"`)
			if brCount < 9 {
				t.Errorf("Esperaba al menos 9 saltos de página para 10 páginas, obtuve: %d", brCount)
			}
		}
	}
}
