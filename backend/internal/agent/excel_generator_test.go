package agent

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateExcel_AndReadBack(t *testing.T) {
	tempDir := t.TempDir()
	excelFile := filepath.Join(tempDir, "ventas_mensuales.xlsx")

	sheets := []ExcelSheetSpec{
		{
			Name:    "Ventas Q1",
			Headers: []string{"ID", "Cliente", "Monto", "Estado"},
			Rows: [][]string{
				{"1", "Geofal SAC", "4500.50", "Completado"},
				{"2", "Inmobiliaria Almasa", "12800.00", "En Proceso"},
				{"3", "Constructora Norte", "3200.75", "Pendiente"},
			},
		},
		{
			Name:    "Gastos",
			Headers: []string{"Concepto", "Costo"},
			Rows: [][]string{
				{"Servidores Cloud", "350.00"},
				{"Licencias", "120.00"},
			},
		},
	}

	createdPath, err := CreateExcelFile(excelFile, sheets)
	if err != nil {
		t.Fatalf("CreateExcelFile falló: %v", err)
	}

	if !strings.HasSuffix(createdPath, ".xlsx") {
		t.Errorf("La ruta creada debería terminar en .xlsx, obtuve: %s", createdPath)
	}

	// Leer el archivo generado usando el extractor de documentos de OzyAssist
	docRes, err := ExtractTextFromDocument(createdPath, 4000)
	if err != nil {
		t.Fatalf("ExtractTextFromDocument falló al leer el Excel generado: %v", err)
	}

	if docRes.FileType != ".xlsx" {
		t.Errorf("Tipo de archivo esperado .xlsx, obtuve: %s", docRes.FileType)
	}

	// Comprobar que contiene los nombres de las hojas y los datos
	content := docRes.Content
	if !strings.Contains(content, "Ventas Q1") {
		t.Errorf("El contenido debería incluir la hoja 'Ventas Q1', obtuve:\n%s", content)
	}
	if !strings.Contains(content, "Geofal SAC") || !strings.Contains(content, "4500.50") {
		t.Errorf("El contenido debería contener 'Geofal SAC' y '4500.50', obtuve:\n%s", content)
	}
	if !strings.Contains(content, "Gastos") || !strings.Contains(content, "Servidores Cloud") {
		t.Errorf("El contenido debería contener 'Gastos' y 'Servidores Cloud', obtuve:\n%s", content)
	}
}
