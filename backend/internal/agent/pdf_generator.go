package agent

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

// PDFSection modela una sección estructurada con título, párrafos y viñetas
type PDFSection struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Bullets []string `json:"bullets,omitempty"`
}

// PDFTable modela una tabla de datos con cabeceras y filas
type PDFTable struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// PDFReportOptions define la configuración completa para la generación del documento
type PDFReportOptions struct {
	Title      string       `json:"title"`
	Subtitle   string       `json:"subtitle,omitempty"`
	Author     string       `json:"author,omitempty"`
	Date       string       `json:"date,omitempty"`
	ThemeColor string       `json:"theme_color,omitempty"` // "lime", "dark", "slate"
	Sections   []PDFSection `json:"sections"`
	Table      *PDFTable    `json:"table,omitempty"`
}

// GeneratePDFReport crea un archivo PDF profesional y estéticamente refinado
func GeneratePDFReport(destPath string, opts PDFReportOptions) error {
	if destPath == "" {
		return fmt.Errorf("ruta de destino de PDF requerida")
	}

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("no se pudo crear directorio destino: %w", err)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 22)
	pdf.AliasNbPages("{nb}")

	// Determinar paleta de colores del tema
	primaryR, primaryG, primaryB := 30, 41, 59       // Slate dark por defecto
	accentR, accentG, accentB := 209, 241, 7         // Ozy Electric Neon Lime (#d1f107)
	if strings.ToLower(opts.ThemeColor) == "dark" {
		primaryR, primaryG, primaryB = 18, 18, 18
	}

	// Pie de página elegante con paginación
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(128, 128, 128)
		footerText := fmt.Sprintf("OzyAssist · %s · Página %d de {nb}", opts.Title, pdf.PageNo())
		pdf.CellFormat(0, 10, tr(footerText), "", 0, "C", false, 0, "")
	})

	pdf.AddPage()

	// 1. Barra de acento superior
	pdf.SetFillColor(accentR, accentG, accentB)
	pdf.Rect(18, 18, 174, 3, "F")
	pdf.Ln(6)

	// 2. Título principal
	pdf.SetFont("Arial", "B", 20)
	pdf.SetTextColor(primaryR, primaryG, primaryB)
	pdf.MultiCell(0, 9, tr(opts.Title), "", "L", false)
	pdf.Ln(2)

	// 3. Subtítulo si existe
	if opts.Subtitle != "" {
		pdf.SetFont("Arial", "I", 12)
		pdf.SetTextColor(100, 116, 139)
		pdf.MultiCell(0, 6, tr(opts.Subtitle), "", "L", false)
		pdf.Ln(2)
	}

	// 4. Metadatos (Autor, Fecha)
	metaParts := []string{}
	if opts.Author != "" {
		metaParts = append(metaParts, "Autor: "+opts.Author)
	}
	dateStr := opts.Date
	if dateStr == "" {
		dateStr = time.Now().Format("02/01/2006")
	}
	metaParts = append(metaParts, "Fecha: "+dateStr)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(140, 140, 140)
	pdf.Cell(0, 5, tr(strings.Join(metaParts, "  ·  ")))
	pdf.Ln(6)

	// Línea divisoria sutil
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetLineWidth(0.3)
	pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
	pdf.Ln(6)

	// 5. Renderizar Secciones
	for _, sec := range opts.Sections {
		// Título de Sección con barra vertical decorativa
		if sec.Title != "" {
			currentY := pdf.GetY()
			if currentY > 250 {
				pdf.AddPage()
				currentY = pdf.GetY()
			}
			pdf.SetFillColor(accentR, accentG, accentB)
			pdf.Rect(18, currentY+1, 2.5, 6.5, "F")

			pdf.SetX(23)
			pdf.SetFont("Arial", "B", 13)
			pdf.SetTextColor(primaryR, primaryG, primaryB)
			pdf.Cell(0, 8, tr(sec.Title))
			pdf.Ln(8)
		}

		// Contenido del párrafo
		if sec.Content != "" {
			pdf.SetFont("Arial", "", 10)
			pdf.SetTextColor(51, 65, 85)
			pdf.MultiCell(0, 5.5, tr(sec.Content), "", "L", false)
			pdf.Ln(3)
		}

		// Viñetas si existen
		for _, b := range sec.Bullets {
			pdf.SetFont("Arial", "", 10)
			pdf.SetTextColor(51, 65, 85)
			pdf.SetX(22)
			pdf.Cell(5, 5.5, tr("•"))
			pdf.SetX(27)
			pdf.MultiCell(165, 5.5, tr(b), "", "L", false)
			pdf.Ln(1)
		}
		pdf.Ln(4)
	}

	// 6. Renderizar Tabla si existe
	if opts.Table != nil && len(opts.Table.Headers) > 0 {
		currentY := pdf.GetY()
		if currentY > 240 {
			pdf.AddPage()
		}

		numCols := len(opts.Table.Headers)
		usableWidth := 174.0
		colWidth := usableWidth / float64(numCols)

		// Cabecera de la tabla
		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(241, 245, 249)
		pdf.SetTextColor(30, 41, 59)
		pdf.SetDrawColor(203, 213, 225)
		pdf.SetLineWidth(0.2)

		for _, h := range opts.Table.Headers {
			pdf.CellFormat(colWidth, 7, tr(h), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)

		// Filas de datos
		pdf.SetFont("Arial", "", 9)
		for rowIdx, row := range opts.Table.Rows {
			// Zebra striping
			if rowIdx%2 == 1 {
				pdf.SetFillColor(248, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			pdf.SetTextColor(51, 65, 85)

			for colIdx := 0; colIdx < numCols; colIdx++ {
				val := ""
				if colIdx < len(row) {
					val = row[colIdx]
				}
				pdf.CellFormat(colWidth, 6.5, tr(val), "1", 0, "L", true, 0, "")
			}
			pdf.Ln(-1)
		}
		pdf.Ln(6)
	}

	return pdf.OutputFileAndClose(destPath)
}

// ConvertDocumentToPDF convierte un archivo existente (CSV, TXT, MD, JSON) a PDF estructurado
func ConvertDocumentToPDF(srcPath, destPath string) error {
	srcPath = filepath.Clean(srcPath)
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("archivo fuente no encontrado: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	baseName := strings.TrimSuffix(filepath.Base(srcPath), ext)

	if destPath == "" {
		destPath = filepath.Join(filepath.Dir(srcPath), baseName+".pdf")
	}

	switch ext {
	case ".csv":
		file, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer file.Close()

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err != nil {
			return fmt.Errorf("error leyendo archivo CSV: %w", err)
		}
		if len(records) == 0 {
			return fmt.Errorf("el archivo CSV está vacío")
		}

		headers := records[0]
		rows := records[1:]

		return GeneratePDFReport(destPath, PDFReportOptions{
			Title:    "Informe de Datos: " + baseName,
			Subtitle: fmt.Sprintf("Generado automáticamente desde %s", filepath.Base(srcPath)),
			Table: &PDFTable{
				Headers: headers,
				Rows:    rows,
			},
		})

	case ".json":
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}

		var opts PDFReportOptions
		if err := json.Unmarshal(data, &opts); err == nil && (opts.Title != "" || len(opts.Sections) > 0) {
			if opts.Title == "" {
				opts.Title = baseName
			}
			return GeneratePDFReport(destPath, opts)
		}

		// Formato libre
		return GeneratePDFReport(destPath, PDFReportOptions{
			Title: baseName,
			Sections: []PDFSection{
				{
					Title:   "Contenido JSON",
					Content: string(data),
				},
			},
		})

	case ".txt", ".md":
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}

		lines := strings.Split(string(data), "\n")
		var sections []PDFSection
		var currentTitle string = "Resumen"
		var currentContent []string
		var currentBullets []string

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") {
				if len(currentContent) > 0 || len(currentBullets) > 0 {
					sections = append(sections, PDFSection{
						Title:   currentTitle,
						Content: strings.Join(currentContent, "\n"),
						Bullets: currentBullets,
					})
					currentContent = nil
					currentBullets = nil
				}
				currentTitle = strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			} else if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
				currentBullets = append(currentBullets, strings.TrimSpace(trimmed[2:]))
			} else if trimmed != "" {
				currentContent = append(currentContent, trimmed)
			}
		}

		if len(currentContent) > 0 || len(currentBullets) > 0 {
			sections = append(sections, PDFSection{
				Title:   currentTitle,
				Content: strings.Join(currentContent, "\n"),
				Bullets: currentBullets,
			})
		}

		return GeneratePDFReport(destPath, PDFReportOptions{
			Title:    baseName,
			Sections: sections,
		})

	default:
		return fmt.Errorf("formato '%s' no soportado directamente para conversión a PDF; use os_create_pdf para diseñar el documento", ext)
	}
}

func execOSCreatePDF(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path       string       `json:"path"`
		Title      string       `json:"title"`
		Subtitle   string       `json:"subtitle"`
		Author     string       `json:"author"`
		ThemeColor string       `json:"theme_color"`
		Sections   []PDFSection `json:"sections"`
		Table      *PDFTable    `json:"table"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_create_pdf: %v", err), false
	}

	targetPath := system.ResolveUserPath(params.Path)
	if !strings.HasSuffix(strings.ToLower(targetPath), ".pdf") {
		targetPath += ".pdf"
	}

	author := params.Author
	if author == "" {
		author = "OzyAssist"
	}

	err := GeneratePDFReport(targetPath, PDFReportOptions{
		Title:      params.Title,
		Subtitle:   params.Subtitle,
		Author:     author,
		ThemeColor: params.ThemeColor,
		Sections:   params.Sections,
		Table:      params.Table,
	})
	if err != nil {
		return fmt.Sprintf("Error generando PDF: %v", err), false
	}

	return fmt.Sprintf("📄 === ARCHIVO PDF GENERADO EXITOSAMENTE ===\n"+
		"• Archivo:    %s\n"+
		"• Título:     %s\n"+
		"• Secciones:  %d\n"+
		"• Estado:     Válido (formato nativo PDF-1.3, compatible con Adobe Acrobat, navegadores y visores de Windows)",
		targetPath, params.Title, len(params.Sections)), true
}

func execOSConvertToPDF(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_convert_to_pdf: %v", err), false
	}

	srcPath := system.ResolveUserPath(params.Src)
	dstPath := ""
	if params.Dst != "" {
		dstPath = system.ResolveUserPath(params.Dst)
	}

	err := ConvertDocumentToPDF(srcPath, dstPath)
	if err != nil {
		return fmt.Sprintf("Error convirtiendo a PDF: %v", err), false
	}

	if dstPath == "" {
		ext := filepath.Ext(srcPath)
		dstPath = strings.TrimSuffix(srcPath, ext) + ".pdf"
	}

	return fmt.Sprintf("📄 === ARCHIVO CONVERTIDO A PDF EXITOSAMENTE ===\n"+
		"• Origen:  %s\n"+
		"• Destino: %s\n"+
		"• Estado:  Válido y estructurado nativamente", srcPath, dstPath), true
}

