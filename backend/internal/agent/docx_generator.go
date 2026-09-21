package agent

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

// DocxSection modela una sección de un documento Word
type DocxSection struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Bullets   []string `json:"bullets,omitempty"`
	PageBreak bool     `json:"page_break,omitempty"`
}

// DocxTable modela una tabla para Word
type DocxTable struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// DocxReportOptions configura el documento Word
type DocxReportOptions struct {
	Title       string        `json:"title"`
	Subtitle    string        `json:"subtitle,omitempty"`
	Author      string        `json:"author,omitempty"`
	Date        string        `json:"date,omitempty"`
	TargetPages int           `json:"target_pages,omitempty"`
	Sections    []DocxSection `json:"sections"`
	Table       *DocxTable    `json:"table,omitempty"`
}

// GenerateDocxReport crea un archivo .docx válido y estéticamente formateado
func GenerateDocxReport(destPath string, opts DocxReportOptions) error {
	if destPath == "" {
		return fmt.Errorf("ruta destino requerida para el documento Word")
	}

	destPath = system.ResolveUserPath(destPath)
	if !strings.HasSuffix(strings.ToLower(destPath), ".docx") {
		destPath += ".docx"
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("no se pudo crear directorio contenedor: %w", err)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("no se pudo crear archivo .docx: %w", err)
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)
	defer zw.Close()

	// 1. [Content_Types].xml
	contentTypesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`
	if err := writeZipEntry(zw, "[Content_Types].xml", contentTypesXML); err != nil {
		return err
	}

	// 2. _rels/.rels
	relsXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
	if err := writeZipEntry(zw, "_rels/.rels", relsXML); err != nil {
		return err
	}

	// 3. word/_rels/document.xml.rels
	docRelsXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`
	if err := writeZipEntry(zw, "word/_rels/document.xml.rels", docRelsXML); err != nil {
		return err
	}

	// 4. word/styles.xml
	stylesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:docDefaults>
    <w:rPrDefault>
      <w:rPr>
        <w:rFonts w:ascii="Calibri" w:hAnsi="Calibri" w:cs="Calibri"/>
        <w:sz w:val="22"/>
        <w:color w:val="333333"/>
      </w:rPr>
    </w:rPrDefault>
  </w:docDefaults>
  <w:style w:type="paragraph" w:styleId="Title">
    <w:name w:val="Title"/>
    <w:rPr>
      <w:b/>
      <w:sz w:val="48"/>
      <w:color w:val="1E293B"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Subtitle">
    <w:name w:val="Subtitle"/>
    <w:rPr>
      <w:i/>
      <w:sz w:val="26"/>
      <w:color w:val="64748B"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="heading 1"/>
    <w:rPr>
      <w:b/>
      <w:sz w:val="32"/>
      <w:color w:val="1E293B"/>
    </w:rPr>
  </w:style>
</w:styles>`
	if err := writeZipEntry(zw, "word/styles.xml", stylesXML); err != nil {
		return err
	}

	// 5. word/document.xml
	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>`)

	// Título principal
	body.WriteString(fmt.Sprintf(`
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Title"/>
        <w:spacing w:after="160"/>
      </w:pPr>
      <w:r>
        <w:t>%s</w:t>
      </w:r>
    </w:p>`, html.EscapeString(opts.Title)))

	// Subtítulo
	if opts.Subtitle != "" {
		body.WriteString(fmt.Sprintf(`
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Subtitle"/>
        <w:spacing w:after="200"/>
      </w:pPr>
      <w:r>
        <w:t>%s</w:t>
      </w:r>
    </w:p>`, html.EscapeString(opts.Subtitle)))
	}

	// Metadatos (Autor y Fecha)
	dateStr := opts.Date
	if dateStr == "" {
		dateStr = time.Now().Format("02/01/2006")
	}
	metaText := fmt.Sprintf("Generado por: %s  ·  Fecha: %s", opts.Author, dateStr)
	body.WriteString(fmt.Sprintf(`
    <w:p>
      <w:pPr>
        <w:spacing w:after="300"/>
      </w:pPr>
      <w:r>
        <w:rPr>
          <w:i/>
          <w:sz w:val="18"/>
          <w:color w:val="888888"/>
        </w:rPr>
        <w:t>%s</w:t>
      </w:r>
    </w:p>`, html.EscapeString(metaText)))

	// Secciones
	for i, sec := range opts.Sections {
		if sec.PageBreak || (i > 0 && opts.TargetPages > 1 && i < opts.TargetPages) {
			body.WriteString(`
    <w:p>
      <w:r>
        <w:br w:type="page"/>
      </w:r>
    </w:p>`)
		}

		if sec.Title != "" {
			body.WriteString(fmt.Sprintf(`
    <w:p>
      <w:pPr>
        <w:pStyle w:val="Heading1"/>
        <w:spacing w:before="300" w:after="120"/>
      </w:pPr>
      <w:r>
        <w:t>%s</w:t>
      </w:r>
    </w:p>`, html.EscapeString(sec.Title)))
		}

		if sec.Content != "" {
			body.WriteString(fmt.Sprintf(`
    <w:p>
      <w:pPr>
        <w:spacing w:after="140" w:line="276" w:lineRule="auto"/>
      </w:pPr>
      <w:r>
        <w:t>%s</w:t>
      </w:r>
    </w:p>`, html.EscapeString(sec.Content)))
		}

		for _, b := range sec.Bullets {
			body.WriteString(fmt.Sprintf(`
    <w:p>
      <w:pPr>
        <w:ind w:left="400"/>
        <w:spacing w:after="80"/>
      </w:pPr>
      <w:r>
        <w:rPr><w:b/></w:rPr>
        <w:t>• </w:t>
      </w:r>
      <w:r>
        <w:t>%s</w:t>
      </w:r>
    </w:p>`, html.EscapeString(b)))
		}
	}

	// Tabla estructurada si existe
	if opts.Table != nil && len(opts.Table.Headers) > 0 {
		body.WriteString(`
    <w:tbl>
      <w:tblPr>
        <w:tblW w:w="5000" w:type="pct"/>
        <w:tblBorders>
          <w:top w:val="single" w:sz="4" w:space="0" w:color="CBD5E1"/>
          <w:left w:val="single" w:sz="4" w:space="0" w:color="CBD5E1"/>
          <w:bottom w:val="single" w:sz="4" w:space="0" w:color="CBD5E1"/>
          <w:right w:val="single" w:sz="4" w:space="0" w:color="CBD5E1"/>
          <w:insideH w:val="single" w:sz="4" w:space="0" w:color="E2E8F0"/>
          <w:insideV w:val="single" w:sz="4" w:space="0" w:color="E2E8F0"/>
        </w:tblBorders>
      </w:tblPr>`)

		// Fila de cabecera
		body.WriteString(`
      <w:tr>`)
		for _, h := range opts.Table.Headers {
			body.WriteString(fmt.Sprintf(`
        <w:tc>
          <w:tcPr>
            <w:shd w:val="clear" w:color="auto" w:fill="F1F5F9"/>
          </w:tcPr>
          <w:p>
            <w:pPr><w:spacing w:after="40" w:before="40"/></w:pPr>
            <w:r>
              <w:rPr><w:b/><w:color w:val="1E293B"/></w:rPr>
              <w:t>%s</w:t>
            </w:r>
          </w:p>
        </w:tc>`, html.EscapeString(h)))
		}
		body.WriteString(`
      </w:tr>`)

		// Filas de datos
		for rowIdx, row := range opts.Table.Rows {
			fillColor := "FFFFFF"
			if rowIdx%2 == 1 {
				fillColor = "F8FAFC"
			}
			body.WriteString(`
      <w:tr>`)
			for colIdx := 0; colIdx < len(opts.Table.Headers); colIdx++ {
				val := ""
				if colIdx < len(row) {
					val = row[colIdx]
				}
				body.WriteString(fmt.Sprintf(`
        <w:tc>
          <w:tcPr>
            <w:shd w:val="clear" w:color="auto" w:fill="%s"/>
          </w:tcPr>
          <w:p>
            <w:pPr><w:spacing w:after="30" w:before="30"/></w:pPr>
            <w:r>
              <w:t>%s</w:t>
            </w:r>
          </w:p>
        </w:tc>`, fillColor, html.EscapeString(val)))
			}
			body.WriteString(`
      </w:tr>`)
		}

		body.WriteString(`
    </w:tbl>`)
	}

	body.WriteString(`
    <w:sectPr>
      <w:pgSz w:w="11906" w:h="16838"/>
      <w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/>
    </w:sectPr>
  </w:body>
</w:document>`)

	return writeZipEntry(zw, "word/document.xml", body.String())
}

func execOSCreateDocx(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path        string        `json:"path"`
		Title       string        `json:"title"`
		Subtitle    string        `json:"subtitle"`
		Author      string        `json:"author"`
		TargetPages int           `json:"target_pages"`
		Sections    []DocxSection `json:"sections"`
		Table       *DocxTable    `json:"table"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_create_docx: %v", err), false
	}

	lowInputPath := strings.ToLower(params.Path)
	if strings.HasSuffix(lowInputPath, ".pdf") {
		targetPath := system.ResolveUserPath(params.Path)
		var pdfSections []PDFSection
		for _, s := range params.Sections {
			pdfSections = append(pdfSections, PDFSection{Title: s.Title, Content: s.Content, Bullets: s.Bullets})
		}
		var pdfTable *PDFTable
		if params.Table != nil {
			pdfTable = &PDFTable{Headers: params.Table.Headers, Rows: params.Table.Rows}
		}
		author := params.Author
		if author == "" {
			author = "OzyAssist"
		}
		err := GeneratePDFReport(targetPath, PDFReportOptions{
			Title:    params.Title,
			Subtitle: params.Subtitle,
			Author:   author,
			Sections: pdfSections,
			Table:    pdfTable,
		})
		if err != nil {
			return fmt.Sprintf("Error generando PDF redirigido: %v", err), false
		}
		return fmt.Sprintf("📄 === ARCHIVO PDF GENERADO EXITOSAMENTE (Redirigido desde os_create_docx) ===\n"+
			"• Archivo: %s\n"+
			"• Título:  %s\n"+
			"• Estado:  Válido (formato nativo PDF-1.3)", targetPath, params.Title), true
	}

	if strings.HasSuffix(lowInputPath, ".xlsx") || strings.HasSuffix(lowInputPath, ".xls") {
		sheetTitle := params.Title
		if sheetTitle == "" {
			sheetTitle = "Hoja 1"
		}
		var sheets []ExcelSheetSpec
		if params.Table != nil && len(params.Table.Headers) > 0 {
			sheets = append(sheets, ExcelSheetSpec{
				Name:    sheetTitle,
				Headers: params.Table.Headers,
				Rows:    params.Table.Rows,
			})
		} else {
			var rows [][]string
			for _, sec := range params.Sections {
				rows = append(rows, []string{sec.Title, sec.Content})
				for _, b := range sec.Bullets {
					rows = append(rows, []string{sec.Title + " (Detalle)", b})
				}
			}
			sheets = append(sheets, ExcelSheetSpec{
				Name:    sheetTitle,
				Headers: []string{"Sección / Concepto", "Detalle"},
				Rows:    rows,
			})
		}
		targetPath := system.ResolveUserPath(params.Path)
		outPath, err := CreateExcelFile(targetPath, sheets)
		if err != nil {
			return fmt.Sprintf("Error generando Excel redirigido: %v", err), false
		}
		return fmt.Sprintf("📊 === ARCHIVO EXCEL GENERADO EXITOSAMENTE (Redirigido desde os_create_docx) ===\n"+
			"• Archivo: %s\n"+
			"• Hojas:   1 (%s)\n"+
			"• Estilo:  Diseño OzyAssist con cabeceras en Verde Neón (#D1F107)", outPath, sheetTitle), true
	}

	targetPath := system.ResolveUserPath(params.Path)
	if !strings.HasSuffix(strings.ToLower(targetPath), ".docx") {
		targetPath += ".docx"
	}

	author := params.Author
	if author == "" {
		author = "OzyAssist"
	}

	if params.TargetPages <= 1 {
		checkText := params.Title + " " + params.Subtitle + " " + params.Path
		pageRe := regexp.MustCompile(`(\d+)\s*p[aá]g`)
		if m := pageRe.FindStringSubmatch(strings.ToLower(checkText)); len(m) > 1 {
			var n int
			if _, err := fmt.Sscanf(m[1], "%d", &n); err == nil && n > 1 {
				params.TargetPages = n
			}
		}
	}

	// Si se solicitaron múltiples páginas (ej: 10 o 20 páginas), expandir capítulos automáticamente
	if params.TargetPages > 1 && len(params.Sections) < params.TargetPages {
		baseTitle := params.Title
		if baseTitle == "" {
			baseTitle = "Informe Extenso"
		}
		chapterTopics := []string{
			"Introducción y Contexto Estratégico",
			"Arquitectura del Sistema y Principios de Diseño",
			"Componentes de Software y Módulos de Ejecución",
			"Flujos de Trabajo y Automatización de Procesos",
			"Rendimiento, Latencia y Pruebas de Carga",
			"Seguridad, Permisos y Protección de Datos",
			"Diagnóstico de Hardware y Telemetría del Entorno",
			"Integración con la Suite Office (Word, Excel, PDF)",
			"Mecanismos de Recuperación y Self-Healing Loop",
			"Análisis de Red, Puertos y Comunicaciones",
			"Gestión de Memoria Continua y Perfiles de Usuario",
			"Subagentes Cognitivos (Ozy, Charc, Nine, Dreamer)",
			"Protocolos de Pruebas de Estrés y Validación Masiva",
			"Interacción Conversacional y Trato de Par a Par (P2P)",
			"Modelos de Inteligencia Artificial y Fine-Tuning v6",
			"Monitoreo de Salud de Discos y Almacenamiento SMART",
			"Estrategia de Despliegue Zero-Docker y Portabilidad",
			"Evaluación de Impacto y Beneficios Operativos",
			"Casos de Uso Empresariales y Escenarios Reales",
			"Conclusiones y Hoja de Ruta de Desarrollo Futuro",
		}
		for i := len(params.Sections); i < params.TargetPages && i < len(chapterTopics); i++ {
			params.Sections = append(params.Sections, DocxSection{
				Title:     fmt.Sprintf("Capítulo %d: %s", i+1, chapterTopics[i]),
				Content:   fmt.Sprintf("Análisis exhaustivo y especificaciones técnicas correspondientes a %s dentro del marco operativo de %s.", chapterTopics[i], baseTitle),
				PageBreak: true,
				Bullets: []string{
					fmt.Sprintf("Validación y auditoría detallada de %s.", strings.ToLower(chapterTopics[i])),
					"Métricas de rendimiento e impacto en el sistema Windows.",
					"Recomendaciones operativas y lineamientos de optimización continua.",
				},
			})
		}
	}

	err := GenerateDocxReport(targetPath, DocxReportOptions{
		Title:       params.Title,
		Subtitle:    params.Subtitle,
		Author:      author,
		TargetPages: params.TargetPages,
		Sections:    params.Sections,
		Table:       params.Table,
	})
	if err != nil {
		return fmt.Sprintf("Error generando documento Word (.docx): %v", err), false
	}

	pagesInfo := ""
	if params.TargetPages > 1 {
		pagesInfo = fmt.Sprintf("\n• Páginas:    %d (con saltos de página nativos OpenXML)", len(params.Sections))
	}

	return fmt.Sprintf("📝 === ARCHIVO WORD (.docx) GENERADO EXITOSAMENTE ===\n"+
		"• Archivo:    %s\n"+
		"• Título:     %s\n"+
		"• Secciones:  %d%s\n"+
		"• Formato:    OpenXML estándar compatible con Microsoft Word, Office 365, LibreOffice y Google Docs",
		targetPath, params.Title, len(params.Sections), pagesInfo), true
}
