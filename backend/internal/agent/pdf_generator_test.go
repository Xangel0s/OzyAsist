package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePDFReport(t *testing.T) {
	tmpDir := t.TempDir()
	outPdf := filepath.Join(tmpDir, "test_report.pdf")

	opts := PDFReportOptions{
		Title:      "Informe de Prueba: Cage The Elephant - Neon Pill",
		Subtitle:   "Análisis Discográfico y Retorno al Garage Rock",
		Author:     "OzyAssist AI",
		ThemeColor: "lime",
		Sections: []PDFSection{
			{
				Title:   "1. Contexto y Lanzamiento",
				Content: "Neon Pill es el sexto álbum de estudio de la banda de rock estadounidense Cage The Elephant, lanzado el 17 de mayo de 2024 a través de RCA Records.",
				Bullets: []string{
					"Producido por John Hill en Sonic Ranch (Texas)",
					"Primer trabajo de estudio desde Social Cues (2019)",
					"Explora temas de redención, salud mental y supervivencia",
				},
			},
			{
				Title:   "2. Análisis Crítico",
				Content: "El álbum equilibra melodías contagiosas con una textura psicodélica más sobria y personal.",
			},
		},
		Table: &PDFTable{
			Headers: []string{"#", "Canción", "Duración", "Comentarios"},
			Rows: [][]string{
				{"1", "HiFi (True Light)", "3:24", "Apertura enérgica y melódica"},
				{"2", "Rainbow", "3:10", "Guitarras brillantes inspiradas en el pop de los 70"},
				{"3", "Neon Pill", "3:19", "Sencillo homónimo, clásico sonido alt-rock"},
			},
		},
	}

	err := GeneratePDFReport(outPdf, opts)
	if err != nil {
		t.Fatalf("GeneratePDFReport falló: %v", err)
	}

	data, err := os.ReadFile(outPdf)
	if err != nil {
		t.Fatalf("No se pudo leer el archivo generado: %v", err)
	}

	if len(data) < 1000 {
		t.Errorf("El archivo PDF es demasiado pequeño (%d bytes)", len(data))
	}

	// Comprobar la cabecera mágica de PDF
	if !strings.HasPrefix(string(data[:10]), "%PDF-") {
		t.Errorf("El archivo generado no tiene cabecera válida de PDF: %q", string(data[:10]))
	}
}

func TestConvertDocumentToPDF_Markdown(t *testing.T) {
	tmpDir := t.TempDir()
	srcMd := filepath.Join(tmpDir, "notas.md")
	dstPdf := filepath.Join(tmpDir, "notas.pdf")

	mdContent := `# Resumen Ejecutivo
Este es un documento de prueba en Markdown.
- Primer punto relevante
- Segundo punto relevante

# Conclusiones
La conversión automática a PDF funciona de forma nativa sin Docker ni librerías CGO.
`
	if err := os.WriteFile(srcMd, []byte(mdContent), 0644); err != nil {
		t.Fatalf("error escribiendo markdown temporal: %v", err)
	}

	err := ConvertDocumentToPDF(srcMd, dstPdf)
	if err != nil {
		t.Fatalf("ConvertDocumentToPDF falló: %v", err)
	}

	data, err := os.ReadFile(dstPdf)
	if err != nil {
		t.Fatalf("no se pudo leer el PDF convertido: %v", err)
	}

	if !strings.HasPrefix(string(data[:10]), "%PDF-") {
		t.Errorf("El archivo convertido no es un PDF válido")
	}
}

func TestGenerateNeonPillReport(t *testing.T) {
	tempDir := t.TempDir()
	destPath := filepath.Join(tempDir, "Informe Neon Pill.pdf")

	opts := PDFReportOptions{
		Title:      "Informe Discográfico: Cage The Elephant — Neon Pill",
		Subtitle:   "Análisis Integral, Ficha Técnica, Producción y Recepción Crítica",
		Author:     "OzyAssist · Asistente Autónomo de Escritorio",
		Date:       "18 de Septiembre de 2026",
		ThemeColor: "lime",
		Sections: []PDFSection{
			{
				Title:   "1. Contexto Histórico y Retorno Musical",
				Content: "Neon Pill es el sexto álbum de estudio de la banda estadounidense de rock alternativo Cage The Elephant. Publicado el 17 de mayo de 2024 bajo el prestigioso sello RCA Records, este trabajo marca el ansiado regreso discográfico del grupo tras cinco años de silencio desde el lanzamiento de 'Social Cues' (2019, ganador del Grammy al Mejor Álbum de Rock).",
				Bullets: []string{
					"Período de gestación atravesado por intensos desafíos personales, salud mental y supervivencia superados por Matthew Shultz.",
					"Grabado en los estudios Sonic Ranch en Tornillo, Texas, bajo la dirección del productor John Hill.",
					"Alineación oficial: Matthew Shultz, Brad Shultz, Nick Bockrath, Matthan Minster, Daniel Tichenor y Jared Champion.",
				},
			},
			{
				Title:   "2. Dirección Sonora y Estilo Musical",
				Content: "A lo largo de sus 12 pistas y 38 minutos de metraje, Neon Pill fusiona la clásica efervescencia garajera de la banda con texturas psicodélicas refinadas, sintetizadores envolventes y melodías pop con reminiscencias setenteras. Los arreglos privilegian la calidez acústica y la nitidez instrumental.",
				Bullets: []string{
					"Guitarras etéreas con efectos de chorus y delays en 'Rainbow' y 'Float Into the Sky'.",
					"Líneas de bajo con pulso funk bailable y dinámica post-punk en el tema homónimo 'Neon Pill'.",
					"Baladas despojadas como 'Out Loud', donde el piano y la voz desnuda rinden homenaje a la memoria paterna.",
				},
			},
			{
				Title:   "3. Temáticas Líricas: Vulnerabilidad, Redención y Resiliencia",
				Content: "A diferencia del tono melancólico y nocturno de Social Cues, Neon Pill abraza un tono de claridad redentora. Las canciones reflexionan sobre el perdón, la fragilidad psicológica, la desconexión digital ('Metaverse') y la belleza de reencontrarse con uno mismo tras situaciones límite.",
			},
			{
				Title:   "4. Recepción Crítica y Posicionamiento",
				Content: "La crítica internacional celebró unánimemente el trabajo por su honestidad sin filtros y su maestría para tejer estribillos inolvidables. Publicaciones como Rolling Stone, NME y Consequence destacaron que la banda suena más cohesionada, vital y madura que nunca.",
			},
		},
		Table: &PDFTable{
			Headers: []string{"#", "Pista", "Duración", "Aspectos Clave"},
			Rows: [][]string{
				{"01", "HiFi (True Light)", "3:24", "Apertura enérgica, sintetizadores vibrantes"},
				{"02", "Rainbow", "3:10", "Pop psicodélico luminoso, guitarras brillantes"},
				{"03", "Neon Pill", "3:19", "Sencillo líder, bajo infeccioso y gancho vocal"},
				{"04", "Float Into the Sky", "3:51", "Atmósfera espacial, guitarras flotantes"},
				{"05", "Metaverse", "2:33", "Tempo veloz con actitud punk y garage"},
				{"06", "Out Loud", "3:22", "Emotiva balada a piano y corazón abierto"},
				{"07", "Ball and Chain", "2:45", "Groove compacto y tensión introspectiva"},
				{"08", "Good Time", "3:08", "Vibra veraniega con un trasfondo nostálgico"},
				{"09", "Shy Eyes", "3:12", "Ritmo bailable, influencias new wave"},
				{"10", "Silent Picture", "3:49", "Construcción cinematográfica y clímax vocal"},
				{"11", "Same", "2:55", "Riff punzante rememorando el rock clásico"},
				{"12", "Over Your Shoulder", "3:17", "Cierre íntimo, acústico y reconfortante"},
			},
		},
	}

	err := GeneratePDFReport(destPath, opts)
	if err != nil {
		t.Fatalf("GeneratePDFReport falló al escribir en Desktop: %v", err)
	}

	fi, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("No se encontró el archivo generado en Desktop: %v", err)
	}
	t.Logf("PDF generado exitosamente en Desktop: %s (tamaño: %d bytes)", destPath, fi.Size())
}

