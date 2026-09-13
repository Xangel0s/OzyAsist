package agent

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/ozyassist/backend/internal/providers"
)

type ReadDocumentParams struct {
	Path      string `json:"path"`                 // Ruta al documento (PDF, DOCX, XLSX, CSV, TXT, MD)
	MaxLength int    `json:"max_length,omitempty"` // Límite de caracteres (default: 8000)
}

type ReadDocumentResult struct {
	Path        string `json:"path"`
	FileType    string `json:"file_type"`
	TotalBytes  int64  `json:"total_bytes"`
	TotalChars  int    `json:"total_chars"`
	Pages       int    `json:"pages,omitempty"`
	Content     string `json:"content"`
	IsTruncated bool   `json:"is_truncated"`
}

// ExtractTextFromDocument extrae texto de PDFs, DOCX, XLSX, CSV y texto plano
func ExtractTextFromDocument(filePath string, maxLength int) (*ReadDocumentResult, error) {
	cleanPath := filepath.Clean(filePath)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo acceder al archivo %s: %v", cleanPath, err)
	}

	if maxLength <= 0 {
		maxLength = 8000
	}

	ext := strings.ToLower(filepath.Ext(cleanPath))
	var extractedText string
	var pages int

	switch ext {
	case ".pdf":
		text, pageCount, err := extractPDF(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("error leyendo PDF: %v", err)
		}
		extractedText = text
		pages = pageCount

	case ".docx":
		text, err := extractDocx(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("error leyendo DOCX: %v", err)
		}
		extractedText = text

	case ".xlsx", ".xlsm":
		text, err := extractXLSX(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("error leyendo Excel: %v", err)
		}
		extractedText = text

	case ".csv":
		text, err := extractCSV(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("error leyendo CSV: %v", err)
		}
		extractedText = text

	default: // Archivos de texto plano (.txt, .md, .json, .log, .yaml, .xml, etc.)
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("error leyendo archivo de texto: %v", err)
		}
		extractedText = string(data)
	}

	totalChars := len(extractedText)
	isTruncated := false
	content := extractedText

	if totalChars > maxLength {
		content = extractedText[:maxLength] + fmt.Sprintf("\n\n... [Contenido truncado. Se extrajeron %d de %d caracteres]", maxLength, totalChars)
		isTruncated = true
	}

	return &ReadDocumentResult{
		Path:        cleanPath,
		FileType:    ext,
		TotalBytes:  info.Size(),
		TotalChars:  totalChars,
		Pages:       pages,
		Content:     content,
		IsTruncated: isTruncated,
	}, nil
}

func extractPDF(filePath string) (string, int, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	totalPage := r.NumPage()
	var buf bytes.Buffer
	reader, err := r.GetPlainText()
	if err != nil {
		return "", totalPage, err
	}

	_, err = buf.ReadFrom(reader)
	if err != nil {
		return "", totalPage, err
	}

	clean := strings.TrimSpace(buf.String())
	return clean, totalPage, nil
}

func extractDocx(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}

			return cleanDocxXML(string(content)), nil
		}
	}
	return "", fmt.Errorf("documento DOCX no contiene word/document.xml")
}

func cleanDocxXML(xmlStr string) string {
	xmlStr = strings.ReplaceAll(xmlStr, "</w:p>", "\n")
	tagRegex := regexp.MustCompile(`<[^>]+>`)
	cleaned := tagRegex.ReplaceAllString(xmlStr, "")

	lines := strings.Split(cleaned, "\n")
	var result []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return strings.Join(result, "\n")
}

func extractCSV(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for i, row := range records {
		if i > 100 {
			sb.WriteString(fmt.Sprintf("... (%d filas restantes omitidas)\n", len(records)-i))
			break
		}
		sb.WriteString(strings.Join(row, " | "))
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

type sstXML struct {
	XMLName xml.Name `xml:"sst"`
	SI      []struct {
		T string `xml:"t"`
		R []struct {
			T string `xml:"t"`
		} `xml:"r"`
	} `xml:"si"`
}

type worksheetXML struct {
	XMLName   xml.Name `xml:"worksheet"`
	SheetData struct {
		Rows []struct {
			R int `xml:"r,attr"`
			C []struct {
				R  string `xml:"r,attr"`
				T  string `xml:"t,attr"`
				V  string `xml:"v"`
				IS struct {
					T string `xml:"t"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"row"`
	} `xml:"sheetData"`
}

type workbookXML struct {
	XMLName xml.Name `xml:"workbook"`
	Sheets  struct {
		Sheet []struct {
			Name    string `xml:"name,attr"`
			SheetID string `xml:"sheetId,attr"`
			RID     string `xml:"id,attr"`
		} `xml:"sheet"`
	} `xml:"sheets"`
}

func extractXLSX(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// 1. Extraer sharedStrings.xml si existe
	var sharedStrings []string
	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, err := f.Open()
			if err == nil {
				var sst sstXML
				if err := xml.NewDecoder(rc).Decode(&sst); err == nil {
					for _, si := range sst.SI {
						if si.T != "" {
							sharedStrings = append(sharedStrings, si.T)
						} else if len(si.R) > 0 {
							var sb strings.Builder
							for _, rItem := range si.R {
								sb.WriteString(rItem.T)
							}
							sharedStrings = append(sharedStrings, sb.String())
						} else {
							sharedStrings = append(sharedStrings, "")
						}
					}
				}
				rc.Close()
			}
			break
		}
	}

	// 2. Extraer nombres de hojas de xl/workbook.xml
	sheetNames := make(map[string]string)
	for _, f := range r.File {
		if f.Name == "xl/workbook.xml" {
			rc, err := f.Open()
			if err == nil {
				var wb workbookXML
				if err := xml.NewDecoder(rc).Decode(&wb); err == nil {
					for idx, sh := range wb.Sheets.Sheet {
						key := fmt.Sprintf("sheet%d.xml", idx+1)
						sheetNames[key] = sh.Name
					}
				}
				rc.Close()
			}
			break
		}
	}

	// 3. Extraer celdas de cada hoja
	var sb strings.Builder
	sheetCount := 0

	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			sheetBase := filepath.Base(f.Name)
			name, ok := sheetNames[sheetBase]
			if !ok {
				name = fmt.Sprintf("Hoja %d", sheetCount+1)
			}

			rc, err := f.Open()
			if err != nil {
				continue
			}

			var ws worksheetXML
			if err := xml.NewDecoder(rc).Decode(&ws); err != nil {
				rc.Close()
				continue
			}
			rc.Close()

			if len(ws.SheetData.Rows) == 0 {
				continue
			}

			sheetCount++
			sb.WriteString(fmt.Sprintf("\n📊 [HOJA: %s]\n", name))

			for rowIdx, row := range ws.SheetData.Rows {
				if rowIdx >= 100 {
					sb.WriteString(fmt.Sprintf("... (%d filas restantes omitidas)\n", len(ws.SheetData.Rows)-rowIdx))
					break
				}

				var rowVals []string
				for _, cell := range row.C {
					val := strings.TrimSpace(cell.V)
					if cell.T == "s" {
						idx, err := strconv.Atoi(val)
						if err == nil && idx >= 0 && idx < len(sharedStrings) {
							val = sharedStrings[idx]
						}
					} else if cell.T == "inlineStr" {
						val = cell.IS.T
					}
					rowVals = append(rowVals, val)
				}
				if len(rowVals) > 0 {
					sb.WriteString(strings.Join(rowVals, " | "))
					sb.WriteString("\n")
				}
			}
		}
	}

	if sheetCount == 0 {
		return "", fmt.Errorf("el archivo Excel no contiene hojas de cálculo con datos válidos")
	}

	return strings.TrimSpace(sb.String()), nil
}

func execOSReadDocument(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params ReadDocumentParams
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_read_document: %v", err), false
	}

	res, err := ExtractTextFromDocument(params.Path, params.MaxLength)
	if err != nil {
		return fmt.Sprintf("Error extrayendo documento: %v", err), false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📄 === DOCUMENTO EXTRAÍDO: %s ===\n", filepath.Base(res.Path)))
	sb.WriteString(fmt.Sprintf("• Tipo:        %s\n", res.FileType))
	sb.WriteString(fmt.Sprintf("• Tamaño:      %.2f KB\n", float64(res.TotalBytes)/1024.0))
	if res.Pages > 0 {
		sb.WriteString(fmt.Sprintf("• Páginas:     %d\n", res.Pages))
	}
	sb.WriteString(fmt.Sprintf("• Caracteres:  %d\n", res.TotalChars))
	sb.WriteString("\n--- CONTENIDO DEL DOCUMENTO ---\n")
	sb.WriteString(res.Content)

	return sb.String(), true
}
