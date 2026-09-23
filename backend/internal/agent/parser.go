package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/providers"
)

var knownToolAliases = map[string]string{
	"os_usb_devices":     "os_hardware_inspector",
	"os_list_usb_ports":  "os_hardware_inspector",
	"os_usb":             "os_hardware_inspector",
	"usb_devices":        "os_hardware_inspector",
	"os_hardware":        "os_hardware_inspector",
	"os_telemetry":       "os_hardware_inspector",
	"hardware_inspector": "os_hardware_inspector",
	"os_creating_pdf":    "os_create_pdf",
	"os_pdf_generator":   "os_create_pdf",
	"os_pdf":             "os_create_pdf",
	"create_pdf":         "os_create_pdf",
	"os_wifi":            "os_wifi_manager",
	"wifi_manager":       "os_wifi_manager",
	"os_network_wifi":    "os_wifi_manager",
	"google_search":      "web_search",
	"internet_search":    "web_search",
	"online_search":      "web_search",
	"search_web":         "web_search",
	"os_file_inspector":  "os_file_info",
	"file_inspector":     "os_file_info",
	"file_info":          "os_file_info",
	"stat_file":          "os_file_info",
	"inspect_file":       "os_file_info",
	"os_inspect_file":    "os_file_info",
	"os_file_stat":       "os_file_info",
	"file_details":       "os_file_info",
	"os_file_details":    "os_file_info",
	"os_excel":           "os_create_excel",
	"create_excel":       "os_create_excel",
	"os_create_xlsx":     "os_create_excel",
	"os_xlsx":            "os_create_excel",
	"os_word":            "os_create_docx",
	"create_word":        "os_create_docx",
	"create_docx":        "os_create_docx",
	"os_docx":            "os_create_docx",
	"os_grep":            "os_search_content",
	"search_content":     "os_search_content",
	"read_document":      "os_read_document",
	"os_read_doc":        "os_read_document",
}

func findRegisteredTool(name string) (string, bool) {
	cleanName := strings.TrimSpace(name)
	lower := strings.ToLower(cleanName)
	if canonical, ok := knownToolAliases[lower]; ok {
		return canonical, true
	}
	if strings.HasPrefix(lower, "mcp_") {
		return cleanName, true
	}
	for _, t := range AgentTools {
		if strings.ToLower(t.Name) == lower {
			return t.Name, true
		}
	}
	for _, t := range VoiceAgentTools {
		if strings.ToLower(t.Name) == lower {
			return t.Name, true
		}
	}
	// Heurística difusa para herramientas de archivos comunes alucinadas
	if (strings.Contains(lower, "file") || strings.Contains(lower, "archivo")) &&
		(strings.Contains(lower, "inspect") || strings.Contains(lower, "info") || strings.Contains(lower, "stat") || strings.Contains(lower, "meta") || strings.Contains(lower, "detail")) {
		return "os_file_info", true
	}
	return "", false
}

var invalidAppTargets = map[string]bool{
	"acci": true, "accion": true, "acción": true, "acciones": true,
	"nombre": true, "app": true, "apps": true, "aplicacion": true, "aplicación": true, "aplicaciones": true,
	"programa": true, "programas": true, "archivo": true, "archivos": true,
	"ventana": true, "ventanas": true, "documento": true, "documentos": true,
	"carpeta": true, "carpetas": true, "reporte": true, "reportes": true,
	"busqueda": true, "búsqueda": true, "nada": true, "algo": true,
	"proceso": true, "procesos": true, "instancia": true, "instancias": true,
	"prueba": true, "ejemplo": true, "sistema": true, "equipo": true,
	"la": true, "el": true, "un": true, "una": true, "los": true, "las": true,
	"experiencia de entrada": true, "experiencia_de_entrada": true, "experiencia": true, "entrada": true,
}

func isInvalidAppTarget(s string) bool {
	clean := strings.ToLower(strings.Trim(s, " .,!?:;\"'"))
	if len(clean) < 2 {
		return true
	}
	return invalidAppTargets[clean]
}

func isRegisteredTool(name string) bool {
	_, found := findRegisteredTool(name)
	return found
}

type jsonBlockMatch struct {
	Block string
	Start int
	End   int
}

func extractBalancedJSONBlocks(text string) []jsonBlockMatch {
	var results []jsonBlockMatch
	n := len(text)
	for i := 0; i < n; i++ {
		if text[i] == '{' {
			depth := 0
			inString := false
			escaped := false
			start := i
			for j := i; j < n; j++ {
				ch := text[j]
				if escaped {
					escaped = false
					continue
				}
				if ch == '\\' {
					escaped = true
					continue
				}
				if ch == '"' {
					inString = !inString
					continue
				}
				if !inString {
					if ch == '{' {
						depth++
					} else if ch == '}' {
						depth--
						if depth == 0 {
							results = append(results, jsonBlockMatch{
								Block: text[start : j+1],
								Start: start,
								End:   j + 1,
							})
							i = j
							break
						}
					}
				}
			}
		}
	}
	return results
}

func extractToolCallsFromText(text string) []providers.ToolCall {
	var calls []providers.ToolCall

	// Patrón 1: <tool_call>{"name": "xyz", "arguments": {}}</tool_call>
	toolCallRe := regexp.MustCompile(`(?is)<tool_call>\s*({.*?})\s*</tool_call>`)
	matches := toolCallRe.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		var tc struct {
			Name      string `json:"name"`
			Arguments any    `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(match[1]), &tc); err == nil {
			if canonical, ok := findRegisteredTool(tc.Name); ok {
				argBytes, _ := json.Marshal(tc.Arguments)
				calls = append(calls, providers.ToolCall{
					ID:    uuid.NewString(),
					Name:  canonical,
					Input: argBytes,
				})
			}
		}
	}

	// Patrón 2: <tool name="xyz" arguments="{}"></tool> o sin cierre
	xmlRe := regexp.MustCompile(`(?is)<tool\s+name="([^"]+)"\s+arguments='([^']+)'`)
	xmlMatches := xmlRe.FindAllStringSubmatch(text, -1)
	if len(xmlMatches) == 0 {
		xmlRe = regexp.MustCompile(`(?is)<tool\s+name="([^"]+)"\s+arguments="([^"]+)"`)
		xmlMatches = xmlRe.FindAllStringSubmatch(text, -1)
	}
	for _, match := range xmlMatches {
		if canonical, ok := findRegisteredTool(match[1]); ok {
			calls = append(calls, providers.ToolCall{
				ID:    uuid.NewString(),
				Name:  canonical,
				Input: []byte(match[2]),
			})
		}
	}

	// Patrón 3: Llama 3.1 <HOLDER>{call_function{"name": "xyz", "arguments": {}}}</HOLDER>
	holderRe := regexp.MustCompile(`(?is)<HOLDER>\s*\{call_function\s*({.*?})\s*\}\s*</HOLDER>`)
	holderMatches := holderRe.FindAllStringSubmatch(text, -1)
	for _, match := range holderMatches {
		var tc struct {
			Name      string `json:"name"`
			Arguments any    `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(match[1]), &tc); err == nil {
			if canonical, ok := findRegisteredTool(tc.Name); ok {
				argBytes, _ := json.Marshal(tc.Arguments)
				calls = append(calls, providers.ToolCall{
					ID:    uuid.NewString(),
					Name:  canonical,
					Input: argBytes,
				})
			}
		}
	}

	// Patrón 4: JSON balanceado con soporte de anidación profunda (sections, arrays, etc.)
	jsonBlocks := extractBalancedJSONBlocks(text)
	for _, jm := range jsonBlocks {
		var tc struct {
			Name      string          `json:"name"`
			Call      string          `json:"call"`
			Tool      string          `json:"tool"`
			Arguments json.RawMessage `json:"arguments"`
			Args      json.RawMessage `json:"args"`
		}
		if err := json.Unmarshal([]byte(jm.Block), &tc); err == nil {
			rawName := tc.Name
			if rawName == "" {
				rawName = tc.Call
			}
			if rawName == "" {
				rawName = tc.Tool
			}
			if rawName != "" {
				if canonical, ok := findRegisteredTool(rawName); ok {
					args := tc.Arguments
					if len(args) == 0 {
						args = tc.Args
					}
					if len(args) == 0 || string(args) == "null" {
						args = []byte("{}")
					}
					var norm any
					if json.Unmarshal(args, &norm) == nil {
						if compact, err := json.Marshal(norm); err == nil {
							args = compact
						}
					}
					if canonical == "os_hardware_inspector" && strings.Contains(strings.ToLower(rawName), "usb") {
						var m map[string]any
						_ = json.Unmarshal(args, &m)
						if m == nil || m["action"] == nil {
							args = []byte(`{"action":"usb"}`)
						}
					}
					exists := false
					for _, c := range calls {
						if c.Name == canonical {
							var a1, a2 any
							_ = json.Unmarshal(c.Input, &a1)
							_ = json.Unmarshal(args, &a2)
							b1, _ := json.Marshal(a1)
							b2, _ := json.Marshal(a2)
							if string(b1) == string(b2) {
								exists = true
								break
							}
						}
					}
					if !exists {
						calls = append(calls, providers.ToolCall{
							ID:    uuid.NewString(),
							Name:  canonical,
							Input: args,
						})
					}
					continue
				}
			}
		}

		// Patrón 4.2: Bloque JSON de argumentos precedido por nombre de herramienta
		if jm.Start > 0 {
			prefixStart := jm.Start - 60
			if prefixStart < 0 {
				prefixStart = 0
			}
			prefix := strings.ToLower(text[prefixStart:jm.Start])
			for _, toolDef := range AgentTools {
				toolLower := strings.ToLower(toolDef.Name)
				if strings.Contains(prefix, toolLower) {
					var testMap map[string]any
					if json.Unmarshal([]byte(jm.Block), &testMap) == nil && len(testMap) > 0 {
						exists := false
						for _, c := range calls {
							if c.Name == toolDef.Name {
								exists = true
								break
							}
						}
						if !exists {
							calls = append(calls, providers.ToolCall{
								ID:    uuid.NewString(),
								Name:  toolDef.Name,
								Input: []byte(jm.Block),
							})
						}
						break
					}
				}
			}
		}
	}

	// Patrón 5: Llamadas estilo función de código (ej: os_launch_app("notepad", "..."))
	if len(calls) == 0 {
		funcCallRe := regexp.MustCompile(`(?m)(?:\\n)?\s*([a-zA-Z0-9_]+)\s*\((.*?)\)`)
		funcMatches := funcCallRe.FindAllStringSubmatch(text, -1)
		for _, match := range funcMatches {
			rawName := strings.TrimSpace(match[1])
			canonical, ok := findRegisteredTool(rawName)
			if !ok {
				continue
			}
			name := canonical
			argsStr := strings.TrimSpace(match[2])
			inputMap := make(map[string]any)
			if argsStr != "" {
				argTokens := parseFunctionArguments(argsStr)
				switch name {
				case "os_launch_app":
					if len(argTokens) > 0 {
						inputMap["target"] = argTokens[0]
					}
					if len(argTokens) > 1 {
						inputMap["path"] = argTokens[1]
					}
				case "os_close_window", "os_focus_window":
					if len(argTokens) > 0 {
						inputMap["title"] = argTokens[0]
					}
				case "os_kill_process":
					if len(argTokens) > 0 {
						inputMap["name"] = argTokens[0]
					}
				case "os_run_command":
					if len(argTokens) > 0 {
						inputMap["command"] = argTokens[0]
					}
					if len(argTokens) > 1 {
						inputMap["cwd"] = argTokens[1]
					}
				case "os_find_files":
					if len(argTokens) > 0 {
						inputMap["pattern"] = argTokens[0]
					}
					if len(argTokens) > 1 {
						inputMap["path"] = argTokens[1]
					}
				case "os_explore":
					if len(argTokens) > 0 {
						inputMap["path"] = argTokens[0]
					}
				case "web_search":
					if len(argTokens) > 0 {
						inputMap["query"] = argTokens[0]
					}
				default:
					if len(argTokens) > 0 {
						inputMap["target"] = argTokens[0]
					}
				}
			}
			// Si os_close_window se llamó sin argumentos (ej: os_close_window()), inferir si hay objetivo común
			if name == "os_close_window" && len(inputMap) == 0 {
				lowerText := strings.ToLower(text)
				if strings.Contains(lowerText, "administrador") || strings.Contains(lowerText, "taskmgr") {
					inputMap["title"] = "administrador de tareas"
				} else if strings.Contains(lowerText, "bloc") || strings.Contains(lowerText, "notepad") {
					inputMap["title"] = "bloc de notas"
				} else if strings.Contains(lowerText, "calc") {
					inputMap["title"] = "calculadora"
				}
			}

			inputBytes, _ := json.Marshal(inputMap)
			calls = append(calls, providers.ToolCall{
				ID:    uuid.NewString(),
				Name:  canonical,
				Input: inputBytes,
			})
		}
	}

	// Patrón 5.5: Formato etiqueta/colon (ej: "Os_creating_pdf: Cages The Elephant - Albums")
	if len(calls) == 0 {
		colonRe := regexp.MustCompile(`(?im)^\s*([a-zA-Z0-9_]+)\s*:\s*([^\r\n]+)`)
		colonMatches := colonRe.FindAllStringSubmatch(text, -1)
		for _, m := range colonMatches {
			rawName := strings.TrimSpace(m[1])
			if canonical, ok := findRegisteredTool(rawName); ok {
				arg := strings.TrimSpace(m[2])
				if arg != "" && !isInvalidAppTarget(arg) {
					inputMap := make(map[string]any)
					switch canonical {
					case "os_create_pdf":
						safeTitle := strings.Trim(arg, " .,\"'")
						docPath := filepath.Join(os.Getenv("USERPROFILE"), "Documents", strings.ReplaceAll(safeTitle, " ", "_")+".pdf")
						inputMap["path"] = docPath
						inputMap["title"] = safeTitle
						inputMap["sections"] = []map[string]string{
							{"title": "Información", "content": fmt.Sprintf("Reporte sobre %s", safeTitle)},
						}
					case "web_search":
						inputMap["query"] = arg
					case "os_launch_app":
						inputMap["target"] = arg
					case "os_close_window":
						inputMap["title"] = arg
					default:
						inputMap["target"] = arg
					}
					inputBytes, _ := json.Marshal(inputMap)
					calls = append(calls, providers.ToolCall{
						ID:    uuid.NewString(),
						Name:  canonical,
						Input: inputBytes,
					})
				}
			}
		}
	}

	// Patrón 6: Fallback heurístico para modelos pequeños
	if len(calls) == 0 {
		lower := strings.ToLower(text)

		// 6.1 Detectar "invoqué a <tool> con <target>" o "invocando <tool> con <target>"
		invokeRe := regexp.MustCompile(`(?i)(?:(?:invoqu[eé]|invocando|ejecut[eé]|ejecutando|usando)\s+(?:a\s+)?)?([a-zA-Z0-9_]+)\s+(?:con|para)\s+([^\r\n]+)`)
		if m := invokeRe.FindStringSubmatch(text); len(m) > 1 {
			rawName := strings.TrimSpace(m[1])
			if canonical, ok := findRegisteredTool(rawName); ok {
				arg := ""
				if len(m) > 2 {
					arg = strings.Trim(m[2], " .,\"'\r\n")
				}
				if strings.HasPrefix(arg, "{") && strings.HasSuffix(arg, "}") {
					var testMap map[string]any
					if json.Unmarshal([]byte(arg), &testMap) == nil {
						calls = append(calls, providers.ToolCall{
							ID:    uuid.NewString(),
							Name:  canonical,
							Input: []byte(arg),
						})
					}
				}
				if strings.HasPrefix(strings.ToLower(arg), "target ") {
					arg = strings.TrimSpace(arg[7:])
				}
				arg = strings.Trim(arg, " .,!?:;\"'")
				isAllowedTarget := !isInvalidAppTarget(arg) || (canonical == "os_launch_app" && (arg == "documents" || arg == "documentos" || arg == "descargas" || arg == "downloads"))
				if len(calls) == 0 && arg != "" && isAllowedTarget {
					inputMap := make(map[string]string)
					switch canonical {
					case "os_launch_app":
						inputMap["target"] = arg
					case "os_close_window", "os_focus_window":
						inputMap["title"] = arg
					case "os_kill_process":
						inputMap["name"] = arg
					case "os_run_command":
						inputMap["command"] = arg
					case "web_search":
						inputMap["query"] = arg
					default:
						inputMap["target"] = arg
					}
					inputBytes, _ := json.Marshal(inputMap)
					calls = append(calls, providers.ToolCall{
						ID:    uuid.NewString(),
						Name:  canonical,
						Input: inputBytes,
					})
				}
			}
		}

		// 6.2 Detectar "ha abierto <app>" o "he abierto <app>" o "abriendo <app>" o "iniciando <app>"
		if len(calls) == 0 {
			openRe := regexp.MustCompile(`(?i)(?:ozy\s+)?(?:ha\s+abierto|he\s+abierto|abriendo|abro|iniciando|inici[eé])\s+(?:(?:la|el|un|una)\s+)?([a-zA-Z0-9_\.\-]+)`)
			if m := openRe.FindStringSubmatch(lower); len(m) > 1 {
				target := strings.Trim(m[1], " .,\"'")
				if !isInvalidAppTarget(target) {
					inputBytes, _ := json.Marshal(map[string]string{"target": target})
					calls = append(calls, providers.ToolCall{
						ID:    uuid.NewString(),
						Name:  "os_launch_app",
						Input: inputBytes,
					})
				}
			} else {
				// Detectar "ha cerrado <app>" o "he cerrado <app>" o "cerrando <app>"
				closeRe := regexp.MustCompile(`(?i)(?:ozy\s+)?(?:ha\s+cerrado|he\s+cerrado|cerrando|cierro)\s+(?:(?:la|el|un|una)\s+)?([a-zA-Z0-9_\.\-\s]+)`)
				if m := closeRe.FindStringSubmatch(lower); len(m) > 1 {
					title := strings.Trim(m[1], " .,\"'")
					if !isInvalidAppTarget(title) {
						inputBytes, _ := json.Marshal(map[string]string{"title": title})
						calls = append(calls, providers.ToolCall{
							ID:    uuid.NewString(),
							Name:  "os_close_window",
							Input: inputBytes,
						})
					}
				}
			}
		}

		// 6.3 Detección directa de herramientas mencionadas conversacionalmente (USB / WiFi)
		if len(calls) == 0 {
			if strings.Contains(lower, "os_usb_devices") || strings.Contains(lower, "os_list_usb_ports") {
				calls = append(calls, providers.ToolCall{
					ID:    uuid.NewString(),
					Name:  "os_hardware_inspector",
					Input: []byte(`{"action": "usb"}`),
				})
			} else if strings.Contains(lower, "os_wifi_manager") || strings.Contains(lower, "os_network_diagnostics") || (strings.Contains(lower, "red") && (strings.Contains(lower, "análisis") || strings.Contains(lower, "analisis"))) {
				act := "status"
				if strings.Contains(lower, "scan") || strings.Contains(lower, "escan") {
					act = "scan"
				}
				inputBytes, _ := json.Marshal(map[string]any{"action": act})
				calls = append(calls, providers.ToolCall{
					ID:    uuid.NewString(),
					Name:  "os_wifi_manager",
					Input: inputBytes,
				})
			} else if (strings.Contains(lower, "excel") || strings.Contains(lower, "xlsx")) && (strings.Contains(lower, "generando") || strings.Contains(lower, "creando") || strings.Contains(lower, "hoja de c")) {
				pathRe := regexp.MustCompile(`(?i)(?:path:\s*|en\s+)?([A-Za-z]:\\[^)\s\r\n]+\.xlsx)`)
				targetPath := filepath.Join(os.Getenv("USERPROFILE"), "Documents", "Hoja_Calculo.xlsx")
				if m := pathRe.FindStringSubmatch(text); len(m) > 1 {
					targetPath = m[1]
				}
				inputBytes, _ := json.Marshal(map[string]any{
					"path":    targetPath,
					"title":   strings.TrimSuffix(filepath.Base(targetPath), filepath.Ext(targetPath)),
					"headers": []string{"Concepto", "Categoría", "Monto"},
					"rows":    [][]string{{"Item 1", "General", "100"}, {"Item 2", "Operativo", "250"}},
				})
				calls = append(calls, providers.ToolCall{
					ID:    uuid.NewString(),
					Name:  "os_create_excel",
					Input: inputBytes,
				})
			} else if (strings.Contains(lower, "word") || strings.Contains(lower, "docx")) && (strings.Contains(lower, "generando") || strings.Contains(lower, "creando") || strings.Contains(lower, "documento")) {
				pathRe := regexp.MustCompile(`(?i)(?:path:\s*|en\s+)?([A-Za-z]:\\[^)\s\r\n]+\.docx)`)
				targetPath := filepath.Join(os.Getenv("USERPROFILE"), "Documents", "Documento.docx")
				if m := pathRe.FindStringSubmatch(text); len(m) > 1 {
					targetPath = m[1]
				}
				inputBytes, _ := json.Marshal(map[string]any{
					"path":  targetPath,
					"title": strings.TrimSuffix(filepath.Base(targetPath), filepath.Ext(targetPath)),
					"sections": []map[string]string{
						{"title": "Resumen", "content": "Documento Word generado automáticamente por OzyAssist."},
					},
				})
				calls = append(calls, providers.ToolCall{
					ID:    uuid.NewString(),
					Name:  "os_create_docx",
					Input: inputBytes,
				})
			} else if strings.Contains(lower, "os_launch_app") {
				target := "documents"
				targetRe := regexp.MustCompile(`(?i)(?:target\s*['":=]+\s*)(['"]?)([a-zA-Z0-9_\-\.\s]+)\1`)
				if tm := targetRe.FindStringSubmatch(text); len(tm) > 2 {
					target = strings.TrimSpace(tm[2])
				}
				inputBytes, _ := json.Marshal(map[string]any{"target": target})
				calls = append(calls, providers.ToolCall{
					ID:    uuid.NewString(),
					Name:  "os_launch_app",
					Input: inputBytes,
				})
			} else if strings.Contains(lower, "os_port_inspector") || (strings.Contains(lower, "puerto") && strings.Contains(lower, "revis")) {
				calls = append(calls, providers.ToolCall{
					ID:    uuid.NewString(),
					Name:  "os_port_inspector",
					Input: []byte(`{"action": "list"}`),
				})
			}
		}
	}

	return calls
}

func parseFunctionArguments(argsStr string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	quoteChar := rune(0)

	runes := []rune(argsStr)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if inQuote {
			if r == '\\' && i+1 < len(runes) && runes[i+1] == quoteChar {
				cur.WriteRune(quoteChar)
				i++
				continue
			}
			if r == quoteChar {
				inQuote = false
				continue
			}
			cur.WriteRune(r)
			continue
		}
		if r == '"' || r == '\'' {
			inQuote = true
			quoteChar = r
			continue
		}
		if r == ',' {
			val := strings.TrimSpace(cur.String())
			if val != "" {
				tokens = append(tokens, val)
			}
			cur.Reset()
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		val := strings.TrimSpace(cur.String())
		if val != "" {
			tokens = append(tokens, val)
		}
	}
	return tokens
}
