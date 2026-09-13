package agent

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractTextFromDocument_TextAndCSV(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Probar archivo de texto plano
	txtPath := filepath.Join(tempDir, "prueba.txt")
	err := os.WriteFile(txtPath, []byte("Contenido de prueba para OzyAssist Document Reader."), 0644)
	if err != nil {
		t.Fatalf("error creando txt: %v", err)
	}

	res, err := ExtractTextFromDocument(txtPath, 1000)
	if err != nil {
		t.Fatalf("ExtractTextFromDocument falló con txt: %v", err)
	}
	if res.FileType != ".txt" || !res.IsTruncated && res.TotalChars == 0 {
		t.Errorf("Resultado txt inesperado: %+v", res)
	}

	// 2. Probar archivo CSV
	csvPath := filepath.Join(tempDir, "datos.csv")
	err = os.WriteFile(csvPath, []byte("Nombre,Rol,Email\nCarlos,Dev,carlos@test.com\nAna,QA,ana@test.com"), 0644)
	if err != nil {
		t.Fatalf("error creando csv: %v", err)
	}

	resCSV, err := ExtractTextFromDocument(csvPath, 1000)
	if err != nil {
		t.Fatalf("ExtractTextFromDocument falló con csv: %v", err)
	}
	if resCSV.FileType != ".csv" {
		t.Errorf("Esperaba FileType=.csv, obtuve %s", resCSV.FileType)
	}
}

func TestExtractTextFromDocument_Docx(t *testing.T) {
	tempDir := t.TempDir()
	docxPath := filepath.Join(tempDir, "documento.docx")

	// Crear un docx sintético (archivo zip con word/document.xml)
	f, err := os.Create(docxPath)
	if err != nil {
		t.Fatalf("error creando archivo docx: %v", err)
	}

	zw := zip.NewWriter(f)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("error creando entry en zip: %v", err)
	}

	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
    <w:body>
        <w:p><w:r><w:t>Propuesta de Desarrollo de Software</w:t></w:r></w:p>
        <w:p><w:r><w:t>Cliente: Geofal CRM</w:t></w:r></w:p>
    </w:body>
</w:document>`

	if _, err := w.Write([]byte(xmlContent)); err != nil {
		t.Fatalf("error escribiendo xml: %v", err)
	}
	zw.Close()
	f.Close()

	res, err := ExtractTextFromDocument(docxPath, 1000)
	if err != nil {
		t.Fatalf("ExtractTextFromDocument falló con docx: %v", err)
	}

	if res.FileType != ".docx" {
		t.Errorf("Esperaba FileType=.docx, obtuve %s", res.FileType)
	}
	if res.Content == "" {
		t.Errorf("El contenido de DOCX no debería estar vacío")
	}
}

func TestExtractTextFromDocument_Xlsx(t *testing.T) {
	tempDir := t.TempDir()
	xlsxPath := filepath.Join(tempDir, "balance.xlsx")

	f, err := os.Create(xlsxPath)
	if err != nil {
		t.Fatalf("error creando archivo xlsx: %v", err)
	}

	zw := zip.NewWriter(f)

	// 1. xl/sharedStrings.xml
	wStrings, err := zw.Create("xl/sharedStrings.xml")
	if err != nil {
		t.Fatalf("error creando sharedStrings: %v", err)
	}
	sharedXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="3" uniqueCount="3">
    <si><t>Producto</t></si>
    <si><t>Precio</t></si>
    <si><t>Laptop Dell</t></si>
</sst>`
	_, _ = wStrings.Write([]byte(sharedXML))

	// 2. xl/workbook.xml
	wBook, err := zw.Create("xl/workbook.xml")
	if err != nil {
		t.Fatalf("error creando workbook: %v", err)
	}
	bookXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
    <sheets>
        <sheet name="Ventas 2026" sheetId="1" id="rId1"/>
    </sheets>
</workbook>`
	_, _ = wBook.Write([]byte(bookXML))

	// 3. xl/worksheets/sheet1.xml
	wSheet, err := zw.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatalf("error creando sheet1: %v", err)
	}
	sheetXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
    <sheetData>
        <row r="1">
            <c r="A1" t="s"><v>0</v></c>
            <c r="B1" t="s"><v>1</v></c>
        </row>
        <row r="2">
            <c r="A2" t="s"><v>2</v></c>
            <c r="B2"><v>1250.50</v></c>
        </row>
    </sheetData>
</worksheet>`
	_, _ = wSheet.Write([]byte(sheetXML))

	zw.Close()
	f.Close()

	res, err := ExtractTextFromDocument(xlsxPath, 2000)
	if err != nil {
		t.Fatalf("ExtractTextFromDocument falló con xlsx: %v", err)
	}

	if res.FileType != ".xlsx" {
		t.Errorf("Esperaba FileType=.xlsx, obtuve %s", res.FileType)
	}
	if !strings.Contains(res.Content, "Ventas 2026") {
		t.Errorf("El contenido debería incluir el nombre de la hoja 'Ventas 2026', obtuve: %s", res.Content)
	}
	if !strings.Contains(res.Content, "Laptop Dell") || !strings.Contains(res.Content, "1250.50") {
		t.Errorf("El contenido debería incluir las celdas extraídas, obtuve: %s", res.Content)
	}
}

