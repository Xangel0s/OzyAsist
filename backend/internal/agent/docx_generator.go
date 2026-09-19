package agent

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

// DocxSection modela una sección de un documento Word
type DocxSection struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Bullets []string `json:"bullets,omitempty"`
}

// DocxTable modela una tabla para Word
type DocxTable struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// DocxReportOptions configura el documento Word
type DocxReportOptions struct {
	Title    string        `json:"title"`
	Subtitle string        `json:"subtitle,omitempty"`
	Author   string        `json:"author,omitempty"`
	Date     string        `json:"date,omitempty"`
	Sections []DocxSection `json:"sections"`
	Table    *DocxTable    `json:"table,omitempty"`
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
	for _, sec := range opts.Sections {
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
		Path     string        `json:"path"`
		Title    string        `json:"title"`
		Subtitle string        `json:"subtitle"`
		Author   string        `json:"author"`
		Sections []DocxSection `json:"sections"`
		Table    *DocxTable    `json:"table"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_create_docx: %v", err), false
	}

	targetPath := system.ResolveUserPath(params.Path)
	if !strings.HasSuffix(strings.ToLower(targetPath), ".docx") {
		targetPath += ".docx"
	}

	author := params.Author
	if author == "" {
		author = "OzyAssist"
	}

	err := GenerateDocxReport(targetPath, DocxReportOptions{
		Title:    params.Title,
		Subtitle: params.Subtitle,
		Author:   author,
		Sections: params.Sections,
		Table:    params.Table,
	})
	if err != nil {
		return fmt.Sprintf("Error generando documento Word (.docx): %v", err), false
	}

	return fmt.Sprintf("📝 === ARCHIVO WORD (.docx) GENERADO EXITOSAMENTE ===\n"+
		"• Archivo:    %s\n"+
		"• Título:     %s\n"+
		"• Secciones:  %d\n"+
		"• Formato:    OpenXML estándar compatible con Microsoft Word, Office 365, LibreOffice y Google Docs",
		targetPath, params.Title, len(params.Sections)), true
}
