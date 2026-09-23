package agent

import (
	"archive/zip"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

// ExcelSheetSpec define una hoja dentro de un libro de Excel.
type ExcelSheetSpec struct {
	Name    string     `json:"name"`              // Nombre de la pestaña (ej: 'Ventas', 'Inventario')
	Headers []string   `json:"headers"`           // Encabezados de columnas (ej: ['Producto', 'Cantidad', 'Precio'])
	Rows    [][]string `json:"rows"`              // Filas de datos
}

// CreateExcelParams parámetros recibidos desde la herramienta os_create_excel.
type CreateExcelParams struct {
	Path    string           `json:"path"`              // Ruta destino (ej: 'Desktop/resumen_ventas.xlsx', 'reporte.xlsx')
	Title   string           `json:"title,omitempty"`   // Título opcional del reporte
	Sheets  []ExcelSheetSpec `json:"sheets,omitempty"`  // Múltiples hojas (opcional)
	Headers []string         `json:"headers,omitempty"` // Para libro simple de una sola hoja
	Rows    [][]string       `json:"rows,omitempty"`    // Para libro simple de una sola hoja
}

// CreateExcelFile crea un archivo .xlsx 100% nativo compatible con OpenXML.
func CreateExcelFile(targetPath string, sheets []ExcelSheetSpec) (string, error) {
	resolvedPath := system.ResolveUserPath(targetPath)
	if !strings.HasSuffix(strings.ToLower(resolvedPath), ".xlsx") {
		resolvedPath += ".xlsx"
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0755); err != nil {
		return "", fmt.Errorf("error creando directorio destino: %v", err)
	}

	f, err := os.Create(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("error creando archivo excel: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	if len(sheets) == 0 {
		sheets = []ExcelSheetSpec{
			{Name: "Hoja 1", Headers: []string{"Item"}, Rows: [][]string{{"Sin datos"}}},
		}
	}

	// 1. [Content_Types].xml
	var ctSB strings.Builder
	ctSB.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
`)
	for i := range sheets {
		ctSB.WriteString(fmt.Sprintf(`  <Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`+"\n", i+1))
	}
	ctSB.WriteString(`</Types>`)
	if err := writeZipEntry(zw, "[Content_Types].xml", ctSB.String()); err != nil {
		return "", err
	}

	// 2. _rels/.rels
	relsContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
	if err := writeZipEntry(zw, "_rels/.rels", relsContent); err != nil {
		return "", err
	}

	// 3. xl/_rels/workbook.xml.rels
	var wbRelsSB strings.Builder
	wbRelsSB.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
`)
	for i := range sheets {
		wbRelsSB.WriteString(fmt.Sprintf(`  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`+"\n", i+1, i+1))
	}
	wbRelsSB.WriteString(fmt.Sprintf(`  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`+"\n", len(sheets)+1))
	wbRelsSB.WriteString(`</Relationships>`)
	if err := writeZipEntry(zw, "xl/_rels/workbook.xml.rels", wbRelsSB.String()); err != nil {
		return "", err
	}

	// 4. xl/workbook.xml
	var wbSB strings.Builder
	wbSB.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
`)
	for i, sh := range sheets {
		name := sh.Name
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("Hoja%d", i+1)
		}
		wbSB.WriteString(fmt.Sprintf(`    <sheet name="%s" sheetId="%d" r:id="rId%d"/>`+"\n", xmlEscape(name), i+1, i+1))
	}
	wbSB.WriteString(`  </sheets>
</workbook>`)
	if err := writeZipEntry(zw, "xl/workbook.xml", wbSB.String()); err != nil {
		return "", err
	}

	// 5. xl/styles.xml (Estilo OzyAssist Brand: encabezados con fondo verde neón #D1F107 y texto en negrita)
	stylesContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <fonts count="2">
    <font><sz val="11"/><name val="Segoe UI"/></font>
    <font><b/><sz val="11"/><color rgb="FF181E00"/><name val="Segoe UI"/></font>
  </fonts>
  <fills count="3">
    <fill><patternFill patternType="none"/></fill>
    <fill><patternFill patternType="gray125"/></fill>
    <fill><patternFill patternType="solid"><fgColor rgb="FFD1F107"/></patternFill></fill>
  </fills>
  <borders count="1">
    <border><left/><right/><top/><bottom/></border>
  </borders>
  <cellStyleXfs count="1">
    <xf numFmtId="0" fontId="0" fillId="0" borderId="0"/>
  </cellStyleXfs>
  <cellXfs count="2">
    <xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0" applyFont="1" applyFill="1"/>
  </cellXfs>
</styleSheet>`
	if err := writeZipEntry(zw, "xl/styles.xml", stylesContent); err != nil {
		return "", err
	}

	// 6. xl/worksheets/sheet*.xml
	for i, sh := range sheets {
		var wsSB strings.Builder
		wsSB.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
`)
		rowNum := 1

		// Encabezados
		if len(sh.Headers) > 0 {
			wsSB.WriteString(fmt.Sprintf(`    <row r="%d">`+"\n", rowNum))
			for colIdx, h := range sh.Headers {
				ref := fmt.Sprintf("%s%d", colName(colIdx), rowNum)
				wsSB.WriteString(fmt.Sprintf(`      <c r="%s" t="inlineStr" s="1"><is><t>%s</t></is></c>`+"\n", ref, xmlEscape(h)))
			}
			wsSB.WriteString(`    </row>` + "\n")
			rowNum++
		}

		// Filas de datos
		for _, row := range sh.Rows {
			wsSB.WriteString(fmt.Sprintf(`    <row r="%d">`+"\n", rowNum))
			for colIdx, val := range row {
				ref := fmt.Sprintf("%s%d", colName(colIdx), rowNum)
				trimmed := strings.TrimSpace(val)
				// Detectar fórmulas de Excel (=SUM, =AVERAGE, etc.)
				if strings.HasPrefix(trimmed, "=") {
					formula := strings.TrimPrefix(trimmed, "=")
					wsSB.WriteString(fmt.Sprintf(`      <c r="%s"><f>%s</f></c>`+"\n", ref, xmlEscape(formula)))
				} else if isNumeric(trimmed) {
					// Detectar números para preservar tipo numérico nativo en Excel
					wsSB.WriteString(fmt.Sprintf(`      <c r="%s"><v>%s</v></c>`+"\n", ref, trimmed))
				} else {
					wsSB.WriteString(fmt.Sprintf(`      <c r="%s" t="inlineStr"><is><t>%s</t></is></c>`+"\n", ref, xmlEscape(val)))
				}
			}
			wsSB.WriteString(`    </row>` + "\n")
			rowNum++
		}

		wsSB.WriteString(`  </sheetData>
</worksheet>`)
		if err := writeZipEntry(zw, fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1), wsSB.String()); err != nil {
			return "", err
		}
	}

	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("error finalizando archivo zip de excel: %v", err)
	}

	return resolvedPath, nil
}

func execOSCreateExcel(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params CreateExcelParams
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_create_excel: %v", err), false
	}

	lowInputPath := strings.ToLower(params.Path)
	if strings.HasSuffix(lowInputPath, ".pdf") {
		targetPath := system.ResolveUserPath(params.Path)
		var pdfSections []PDFSection
		var pdfTable *PDFTable
		if len(params.Headers) > 0 && len(params.Rows) > 0 {
			pdfTable = &PDFTable{Headers: params.Headers, Rows: params.Rows}
		}
		title := params.Title
		if title == "" {
			title = "Reporte"
		}
		pdfSections = append(pdfSections, PDFSection{
			Title:   title,
			Content: "Generado automáticamente desde datos tabulares.",
		})
		err := GeneratePDFReport(targetPath, PDFReportOptions{
			Title:    title,
			Author:   "OzyAssist",
			Sections: pdfSections,
			Table:    pdfTable,
		})
		if err != nil {
			return fmt.Sprintf("Error generando PDF redirigido: %v", err), false
		}
		return fmt.Sprintf("📄 === ARCHIVO PDF GENERADO EXITOSAMENTE (Redirigido desde os_create_excel) ===\n"+
			"• Archivo: %s\n"+
			"• Título:  %s\n"+
			"• Estado:  Válido (formato nativo PDF-1.3)", targetPath, title), true
	}

	if strings.HasSuffix(lowInputPath, ".docx") || strings.HasSuffix(lowInputPath, ".doc") {
		targetPath := system.ResolveUserPath(params.Path)
		var docxTable *DocxTable
		if len(params.Headers) > 0 && len(params.Rows) > 0 {
			docxTable = &DocxTable{Headers: params.Headers, Rows: params.Rows}
		}
		title := params.Title
		if title == "" {
			title = "Documento"
		}
		err := GenerateDocxReport(targetPath, DocxReportOptions{
			Title:  title,
			Author: "OzyAssist",
			Sections: []DocxSection{
				{Title: title, Content: "Generado automáticamente desde datos tabulares."},
			},
			Table: docxTable,
		})
		if err != nil {
			return fmt.Sprintf("Error generando Word redirigido: %v", err), false
		}
		return fmt.Sprintf("📝 === ARCHIVO WORD (.docx) GENERADO EXITOSAMENTE (Redirigido desde os_create_excel) ===\n"+
			"• Archivo: %s\n"+
			"• Título:  %s\n"+
			"• Formato: OpenXML estándar compatible con Microsoft Word", targetPath, title), true
	}

	sheets := params.Sheets
	if len(sheets) == 0 {
		sheetName := "Hoja 1"
		if params.Title != "" {
			sheetName = params.Title
		}
		sheets = []ExcelSheetSpec{
			{
				Name:    sheetName,
				Headers: params.Headers,
				Rows:    params.Rows,
			},
		}
	}

	outPath, err := CreateExcelFile(params.Path, sheets)
	if err != nil {
		return fmt.Sprintf("Error generando archivo Excel: %v", err), false
	}

	totalRows := 0
	for _, sh := range sheets {
		totalRows += len(sh.Rows)
	}

	return fmt.Sprintf("📊 === ARCHIVO EXCEL GENERADO EXITOSAMENTE ===\n"+
		"• Archivo:     %s\n"+
		"• Hojas:       %d (%s)\n"+
		"• Filas:       %d\n"+
		"• Estilo:      Diseño OzyAssist con cabeceras en Verde Neón (#D1F107)\n"+
		"El archivo está disponible y listo para abrir en Microsoft Excel, LibreOffice o Google Sheets.",
		outPath, len(sheets), sheets[0].Name, totalRows), true
}

func writeZipEntry(zw *zip.Writer, name, content string) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("error creando %s en zip: %v", name, err)
	}
	_, err = w.Write([]byte(content))
	return err
}

func colName(idx int) string {
	name := ""
	idx++
	for idx > 0 {
		idx--
		name = string(rune('A'+(idx%26))) + name
		idx /= 26
	}
	return name
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func xmlEscape(s string) string {
	var buf strings.Builder
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}
