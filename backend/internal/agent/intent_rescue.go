package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/providers"
)

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

// checkPendingTaskRequirements evalúa si el objetivo original del usuario contiene acciones explícitas
// (ej: redactar correo, enviar WhatsApp/Telegram, notificar) que aún no hayan sido invocadas por el LLM.
func checkPendingTaskRequirements(userMessage string, executedCalls []providers.ToolCall) string {
	executed := make(map[string]bool)
	for _, tc := range executedCalls {
		executed[tc.Name] = true
	}

	lower := strings.TrimSpace(strings.ToLower(userMessage))

	// No exigir herramientas para preguntas informativas o conceptuales
	if strings.HasPrefix(lower, "cómo") || strings.HasPrefix(lower, "como") ||
		strings.HasPrefix(lower, "qué") || strings.HasPrefix(lower, "que") ||
		strings.HasPrefix(lower, "cuál") || strings.HasPrefix(lower, "cual") ||
		strings.HasPrefix(lower, "explica") || strings.HasPrefix(lower, "dime") {
		return ""
	}

	hasActionVerb := strings.Contains(lower, "redacta") || strings.Contains(lower, "envía") || strings.Contains(lower, "envia") || strings.Contains(lower, "prepara") || strings.Contains(lower, "escribe")

	// 1. Verificación de Correo Electrónico (solo con destinatario explícito o verbo de acción)
	emailAddr := emailRegex.FindString(userMessage)
	hasEmailIntent := (strings.Contains(lower, "correo") || strings.Contains(lower, "email") || strings.Contains(lower, "gmail")) && hasActionVerb
	if (emailAddr != "" || hasEmailIntent) && !executed["os_draft_email"] {
		recipient := emailAddr
		if recipient == "" {
			recipient = "el destinatario solicitado en tu objetivo"
		}
		return fmt.Sprintf("[SISTEMA AUTÓNOMO - MOTOR DE TAREAS]: Has completado la primera etapa del objetivo. Ahora ejecuta inmediatamente el siguiente paso pendiente: redacta el resumen y abre el borrador de correo con 'os_draft_email' para %s.", recipient)
	}

	// 2. Verificación de WhatsApp (solo con verbo de acción)
	hasWhatsAppIntent := (strings.Contains(lower, "whatsapp") || strings.Contains(lower, "wasap")) && hasActionVerb
	if hasWhatsAppIntent && !executed["os_draft_whatsapp"] {
		return "[SISTEMA AUTÓNOMO - MOTOR DE TAREAS]: Ejecuta ahora el siguiente paso pendiente del plan: prepara y abre el mensaje en WhatsApp Web con 'os_draft_whatsapp'."
	}

	// 3. Verificación de Telegram (solo con verbo de acción)
	hasTelegramIntent := strings.Contains(lower, "telegram") && hasActionVerb
	if hasTelegramIntent && !executed["os_draft_telegram"] && !executed["telegram_send_message"] {
		return "[SISTEMA AUTÓNOMO - MOTOR DE TAREAS]: Ejecuta ahora el siguiente paso pendiente del plan: prepara el mensaje de Telegram usando 'os_draft_telegram'."
	}

	// 4. Verificación de Notificación de Escritorio
	hasNotifyIntent := strings.Contains(lower, "notifica") || strings.Contains(lower, "notifícame") || strings.Contains(lower, "avísame al finalizar")
	if hasNotifyIntent && !executed["os_notify"] && len(executedCalls) > 0 {
		return "[SISTEMA AUTÓNOMO - MOTOR DE TAREAS]: Las acciones principales han finalizado. Envía ahora la notificación de escritorio al usuario con 'os_notify'."
	}
	return ""
}

// rescueDirectUserIntent actúa como red de seguridad agéntica cuando un modelo pequeño (ej: 1.5B/3B)
// alucina una respuesta evasiva ("Ya tengo abierto...", "Lo siento, no puedo acceder...")
// a pesar de que el usuario emitió una orden explícita y directa de acción en el sistema.
func rescueDirectUserIntent(userMessage, turnText string) []providers.ToolCall {
	u := strings.TrimSpace(strings.ToLower(userMessage))
	cleanU := u
	for _, p := range []string{"hola ozy ", "hola ", "por favor ", "porfa ", "oye ozy ", "oye ", "buenas tardes ", "buenos dias ", "buenas "} {
		if strings.HasPrefix(cleanU, p) {
			cleanU = strings.TrimSpace(cleanU[len(p):])
		}
	}

	// No interceptar preguntas informativas o conceptuales salvo que indaguen hardware o estado del sistema
	isSystemInquiry := strings.Contains(u, "usb") || strings.Contains(u, "wifi") || strings.Contains(u, "wi-fi") ||
		strings.Contains(u, "hardware") || strings.Contains(u, "disco") || strings.Contains(u, "archivo") ||
		strings.Contains(u, "informe") || strings.Contains(u, "ventana") || strings.Contains(u, "proceso") ||
		strings.Contains(u, "puerto") || strings.Contains(u, "salud") || strings.Contains(u, "recurso") ||
		strings.Contains(u, "cread") || strings.Contains(u, "modific") || strings.Contains(u, "fecha") || strings.Contains(u, "peso")

	if !isSystemInquiry {
		if strings.HasPrefix(u, "¿cómo") || strings.HasPrefix(u, "cómo") || strings.HasPrefix(u, "como") ||
			strings.HasPrefix(u, "¿qué") || strings.HasPrefix(u, "qué") || strings.HasPrefix(u, "que ") ||
			strings.HasPrefix(u, "explica") {
			return nil
		}
	}

	// 0.1 Detección de consulta de metadatos o creación de archivo ("cuando fueron creados", "fecha de creacion", "cuanto pesa")
	isMetadataInquiry := (strings.Contains(u, "cuando") || strings.Contains(u, "cuándo")) &&
		(strings.Contains(u, "cread") || strings.Contains(u, "creó") || strings.Contains(u, "creo") || strings.Contains(u, "modific")) ||
		strings.Contains(u, "fecha de creaci") || strings.Contains(u, "fecha de modificaci") ||
		strings.Contains(u, "cuanto pesa") || strings.Contains(u, "cuánto pesa") || strings.Contains(u, "tamaño del archivo")

	if isMetadataInquiry {
		if targetFile, ok := GetFocusedFile(); ok && targetFile != "" {
			inputBytes, _ := json.Marshal(map[string]string{"path": targetFile})
			return []providers.ToolCall{
				{
					ID:    uuid.NewString(),
					Name:  "os_file_info",
					Input: inputBytes,
				},
			}
		}
	}

	// 0.2 Corrección o apertura contextual con "el de X", "no el de X", "abriste X no el de Y"
	if strings.Contains(u, "el de ") {
		idx := strings.Index(u, "el de ")
		topicTarget := strings.TrimSpace(u[idx+6:])
		for _, stop := range []string{" por favor", " porfa", " ya", " ahora", "."} {
			topicTarget = strings.TrimSuffix(topicTarget, stop)
		}
		if topicTarget != "" && len(topicTarget) > 1 && !isInvalidAppTarget(topicTarget) {
			docsDir := filepath.Join(os.Getenv("USERPROFILE"), "Documents")
			if entries, err := os.ReadDir(docsDir); err == nil {
				topicLower := strings.ToLower(topicTarget)
				var matchedFile string
				var newestMod time.Time
				for _, e := range entries {
					if !e.IsDir() {
						eLower := strings.ToLower(e.Name())
						if strings.Contains(eLower, topicLower) {
							if info, iErr := e.Info(); iErr == nil && info.ModTime().After(newestMod) {
								newestMod = info.ModTime()
								matchedFile = filepath.Join(docsDir, e.Name())
							}
						}
					}
				}
				if matchedFile != "" {
					SetFocusedFile(matchedFile, topicTarget)
					inputBytes, _ := json.Marshal(map[string]string{"target": matchedFile})
					return []providers.ToolCall{
						{
							ID:    uuid.NewString(),
							Name:  "os_launch_app",
							Input: inputBytes,
						},
					}
				}
			}
		}
	}

	// Diccionario de aplicaciones comunes de Windows
	knownAppAliases := []struct{ Pattern, Target string }{
		{"calculadora", "calc"}, {"calc", "calc"}, {"bloc de notas", "notepad"}, {"notepad", "notepad"},
		{"antigravity ide", "antigravity"}, {"antigravity", "antigravity"},
		{"administrador de tareas", "taskmgr"}, {"admin tareas", "taskmgr"}, {"admin de tareas", "taskmgr"}, {"taskmgr", "taskmgr"},
		{"explorador de archivos", "explorer"}, {"explorador", "explorer"},
		{"spotify", "spotify"}, {"paint", "mspaint"}, {"terminal", "wt"}, {"roblox", "roblox"},
		{"documentos", "documents"}, {"descargas", "downloads"},
	}

	var calls []providers.ToolCall
	seenTargets := make(map[string]bool)

	// 1. Detectar intención de apertura explícita (abre, abrir, inicia, iniciar, ejecuta, ejecutar, lanza, lanzar)
	isOpenIntent := strings.HasPrefix(cleanU, "abre ") || strings.HasPrefix(cleanU, "abrir ") ||
		strings.HasPrefix(cleanU, "inicia ") || strings.HasPrefix(cleanU, "iniciar ") ||
		strings.HasPrefix(cleanU, "ejecuta ") || strings.HasPrefix(cleanU, "ejecutar ") ||
		strings.HasPrefix(cleanU, "lanza ") || strings.HasPrefix(cleanU, "lanzar ") ||
		cleanU == "ábrelo" || cleanU == "abrelo" || cleanU == "ábrela" || cleanU == "abrela" || cleanU == "abrirlo" ||
		strings.HasPrefix(cleanU, "ábrelo") || strings.HasPrefix(cleanU, "abrelo")

	if isOpenIntent {
		for _, app := range knownAppAliases {
			if strings.Contains(cleanU, app.Pattern) {
				if seenTargets[app.Target] {
					continue
				}
				seenTargets[app.Target] = true
				inputBytes, _ := json.Marshal(map[string]string{"target": app.Target})
				calls = append(calls, providers.ToolCall{
					ID:    uuid.NewString(),
					Name:  "os_launch_app",
					Input: inputBytes,
				})
			}
		}
		// Si no coincidió con ningún alias conocido, extraer el objetivo arbitrario directamente
		if len(calls) == 0 {
			cleanTarget := u
			for _, prefix := range []string{"abre ", "abrir ", "inicia ", "iniciar ", "ejecuta ", "ejecutar ", "lanza ", "lanzar "} {
				if strings.HasPrefix(cleanTarget, prefix) {
					cleanTarget = strings.TrimSpace(cleanTarget[len(prefix):])
					break
				}
			}
			for _, art := range []string{"el ", "la ", "los ", "las ", "un ", "una "} {
				if strings.HasPrefix(cleanTarget, art) {
					cleanTarget = strings.TrimSpace(cleanTarget[len(art):])
					break
				}
			}
			cleanTarget = strings.Trim(cleanTarget, " .,!?:;\"'")
			if cleanTarget == "archivo" || cleanTarget == "informe" || cleanTarget == "pdf" || cleanTarget == "documento" ||
				strings.Contains(u, "abre el archivo") || strings.Contains(u, "abrir el archivo") ||
				strings.Contains(u, "abre el informe") || strings.Contains(u, "abrir el informe") ||
				strings.Contains(u, "en pantalla") || u == "ábrelo" || u == "abrelo" || u == "ábrela" || u == "abrela" || u == "abrirlo" ||
				strings.HasPrefix(u, "ábrelo") || strings.HasPrefix(u, "abrelo") {

				// Prioridad 1: Si hay un archivo en foco en la conversación, abrirlo directamente
				if focused, ok := GetFocusedFile(); ok && focused != "" {
					inputBytes, _ := json.Marshal(map[string]string{"target": focused})
					return []providers.ToolCall{
						{
							ID:    uuid.NewString(),
							Name:  "os_launch_app",
							Input: inputBytes,
						},
					}
				}

				docsDir := filepath.Join(os.Getenv("USERPROFILE"), "Documents")
				entries, err := os.ReadDir(docsDir)
				if err == nil {
					var newestFile string
					var newestMod time.Time
					for _, e := range entries {
						if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".pdf") {
							info, iErr := e.Info()
							if iErr == nil && info.ModTime().After(newestMod) {
								newestMod = info.ModTime()
								newestFile = filepath.Join(docsDir, e.Name())
							}
						}
					}
					if newestFile != "" {
						SetFocusedFile(newestFile)
						inputBytes, _ := json.Marshal(map[string]string{"target": newestFile})
						calls = append(calls, providers.ToolCall{
							ID:    uuid.NewString(),
							Name:  "os_launch_app",
							Input: inputBytes,
						})
					}
				}
			} else if cleanTarget != "" && len(cleanTarget) > 1 && !isInvalidAppTarget(cleanTarget) {
				inputBytes, _ := json.Marshal(map[string]string{"target": cleanTarget})
				calls = append(calls, providers.ToolCall{
					ID:    uuid.NewString(),
					Name:  "os_launch_app",
					Input: inputBytes,
				})
			}
		}
		if len(calls) > 0 {
			return calls
		}
	}

	// 2. Detectar intención de cierre explícito (cierra, cerrar, mata, matar, termina, terminar)
	isCloseIntent := strings.HasPrefix(cleanU, "cierra ") || strings.HasPrefix(cleanU, "cerrar ") ||
		strings.HasPrefix(cleanU, "mata ") || strings.HasPrefix(cleanU, "matar ") ||
		strings.HasPrefix(cleanU, "termina ") || strings.HasPrefix(cleanU, "terminar ")

	if isCloseIntent {
		for _, app := range knownAppAliases {
			if strings.Contains(cleanU, app.Pattern) {
				if seenTargets[app.Target] {
					continue
				}
				seenTargets[app.Target] = true
				inputBytes, _ := json.Marshal(map[string]string{"title": app.Pattern})
				calls = append(calls, providers.ToolCall{ID: uuid.NewString(), Name: "os_close_window", Input: inputBytes})
			}
		}
		// Si no coincidió con ningún alias conocido, extraer el título arbitrario directamente
		if len(calls) == 0 {
			cleanTitle := cleanU
			for _, prefix := range []string{"cierra ", "cerrar ", "mata ", "matar ", "termina ", "terminar "} {
				if strings.HasPrefix(cleanTitle, prefix) {
					cleanTitle = strings.TrimSpace(cleanTitle[len(prefix):])
					break
				}
			}
			for _, art := range []string{"el ", "la ", "los ", "las ", "un ", "una "} {
				if strings.HasPrefix(cleanTitle, art) {
					cleanTitle = strings.TrimSpace(cleanTitle[len(art):])
					break
				}
			}
			cleanTitle = strings.Trim(cleanTitle, " .,!?:;\"'")
			if cleanTitle != "" && len(cleanTitle) > 1 && !isInvalidAppTarget(cleanTitle) {
				inputBytes, _ := json.Marshal(map[string]string{"title": cleanTitle})
				calls = append(calls, providers.ToolCall{ID: uuid.NewString(), Name: "os_close_window", Input: inputBytes})
			}
		}
		if len(calls) > 0 {
			return calls
		}
	}

	// 2.8 Detectar intención de eliminación de archivos ("elimina los pdf", "borra los 3 pdf en documentos")
	isDeleteIntent := strings.HasPrefix(cleanU, "elimina ") || strings.HasPrefix(cleanU, "eliminar ") ||
		strings.HasPrefix(cleanU, "borra ") || strings.HasPrefix(cleanU, "borrar ")
	if isDeleteIntent && (strings.Contains(cleanU, "pdf") || strings.Contains(cleanU, "archivo") || strings.Contains(cleanU, "documento")) {
		docsDir := filepath.Join(os.Getenv("USERPROFILE"), "Documents")
		if entries, err := os.ReadDir(docsDir); err == nil {
			type fileEntry struct { path string; modTime time.Time }
			var pdfs []fileEntry
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".pdf") {
					if info, err := e.Info(); err == nil {
						pdfs = append(pdfs, fileEntry{path: filepath.Join(docsDir, e.Name()), modTime: info.ModTime()})
					}
				}
			}
			sort.Slice(pdfs, func(i, j int) bool { return pdfs[i].modTime.After(pdfs[j].modTime) })
			var matched []fileEntry
			var searchWord string
			for _, w := range strings.Fields(cleanU) {
				if len(w) > 3 && !strings.Contains("elimina eliminar borra borrar los las un una el la pdf pdfs archivo archivos documento documentos en de del", w) {
					searchWord = w; break
				}
			}
			if searchWord != "" {
				for _, p := range pdfs {
					if strings.Contains(strings.ToLower(filepath.Base(p.path)), searchWord) { matched = append(matched, p) }
				}
				if len(matched) > 0 { pdfs = matched }
			}
			count := len(pdfs)
			if strings.Contains(cleanU, " 1 ") || strings.Contains(cleanU, " un ") { count = 1 }
			if strings.Contains(cleanU, " 2 ") || strings.Contains(cleanU, " dos ") { count = 2 }
			if strings.Contains(cleanU, " 3 ") || strings.Contains(cleanU, " tres ") { count = 3 }
			if len(pdfs) == 0 {
				for i := 0; i < count; i++ {
					inBytes, _ := json.Marshal(map[string]any{"path": filepath.Join(docsDir, fmt.Sprintf("documento_%d.pdf", i+1)), "permanent": false})
					calls = append(calls, providers.ToolCall{ID: uuid.NewString(), Name: "os_delete_item", Input: inBytes})
				}
				return calls
			}
			if count > len(pdfs) { count = len(pdfs) }
			for i := 0; i < count; i++ {
				inBytes, _ := json.Marshal(map[string]any{"path": pdfs[i].path, "permanent": false})
				calls = append(calls, providers.ToolCall{ID: uuid.NewString(), Name: "os_delete_item", Input: inBytes})
			}
			if len(calls) > 0 { return calls }
		}
	}

	// 3. Detectar intención de Hardware / Puertos USB y Puertos Físicos
	if strings.Contains(u, "usb") || strings.Contains(u, "puerto fisico") || strings.Contains(u, "puerto físico") || strings.Contains(u, "dispositivos fisicos") || strings.Contains(u, "dispositivos físicos") {
		return []providers.ToolCall{{ID: uuid.NewString(), Name: "os_hardware_inspector", Input: []byte(`{"action": "usb"}`)}}
	}

	// 3.5 Detectar intención de puertos lógicos / puertos de red
	if (strings.Contains(u, "puerto") || strings.Contains(u, "puertos")) && !strings.Contains(u, "usb") && !strings.Contains(u, "fisic") && !strings.Contains(u, "físic") {
		return []providers.ToolCall{{ID: uuid.NewString(), Name: "os_port_inspector", Input: []byte(`{"action": "list"}`)}}
	}

	// 4. Detectar intención de WiFi o análisis de red
	isWiFiIntent := strings.Contains(u, "wifi") || strings.Contains(u, "wi-fi") || strings.Contains(u, "red wifi") ||
		strings.Contains(u, "análisis de red") || strings.Contains(u, "analisis de red") || strings.Contains(u, "estado de red") ||
		strings.Contains(u, "diagnóstico de red") || strings.Contains(u, "diagnostico de red")
	if isWiFiIntent {
		action := "status"
		if strings.Contains(u, "escan") || strings.Contains(u, "redes") {
			action = "scan"
		}
		inputBytes, _ := json.Marshal(map[string]string{"action": action})
		return []providers.ToolCall{{ID: uuid.NewString(), Name: "os_wifi_manager", Input: inputBytes}}
	}

	// 5. Detectar intención de salud del sistema o telemetría
	isHealthIntent := strings.Contains(u, "salud del sistema") || strings.Contains(u, "chequeo de salud") ||
		strings.Contains(u, "recursos del equipo") || strings.Contains(u, "temperatura") ||
		strings.Contains(u, "rendimiento del equipo")
	if isHealthIntent {
		action := "health"
		if strings.Contains(u, "recursos") || strings.Contains(u, "temperatura") {
			action = "telemetry"
		}
		inputBytes, _ := json.Marshal(map[string]string{"action": action})
		return []providers.ToolCall{{ID: uuid.NewString(), Name: "os_hardware_inspector", Input: inputBytes}}
	}

	// 6. Detectar intención de búsqueda web explícita
	isSearchIntent := strings.HasPrefix(u, "busca en la web ") || strings.HasPrefix(u, "busca en internet ") ||
		strings.HasPrefix(u, "buscar en la web ") || strings.HasPrefix(u, "buscar en internet ") ||
		strings.HasPrefix(u, "investiga sobre ") || strings.HasPrefix(u, "investigar sobre ") ||
		strings.HasPrefix(u, "busca información sobre ") || strings.HasPrefix(u, "busca informacion sobre ")
	if isSearchIntent {
		query := u
		for _, prefix := range []string{"busca en la web ", "busca en internet ", "buscar en la web ", "buscar en internet ", "investiga sobre ", "investigar sobre ", "busca información sobre ", "busca informacion sobre "} {
			if strings.HasPrefix(query, prefix) { query = strings.TrimSpace(query[len(prefix):]); break }
		}
		if query != "" {
			inputBytes, _ := json.Marshal(map[string]string{"query": query})
			return []providers.ToolCall{{ID: uuid.NewString(), Name: "web_search", Input: inputBytes}}
		}
	}

	// 6.8 Detectar intención de creación de archivo Excel (.xlsx)
	isExcelIntent := (strings.Contains(u, "excel") || strings.Contains(u, "xlsx") || strings.Contains(u, "hoja de c")) &&
		(strings.Contains(u, "crea") || strings.Contains(u, "haz") || strings.Contains(u, "genera") || strings.Contains(u, "construye"))
	if isExcelIntent {
		fileName := "Hoja_Calculo_2026.xlsx"
		if m := regexp.MustCompile(`([a-zA-Z0-9_\-]+\.xlsx)`).FindString(userMessage); m != "" { fileName = m }
		excelPath := filepath.Join(os.Getenv("USERPROFILE"), "Documents", fileName)
		inputBytes, _ := json.Marshal(map[string]any{
			"path": excelPath, "title": strings.TrimSuffix(fileName, ".xlsx"),
			"headers": []string{"Concepto", "Categoría", "Monto"},
			"rows": [][]string{{"Operaciones", "Infraestructura", "1500"}, {"Licencias", "Software", "750"}, {"Servidores", "Cloud", "1200"}},
		})
		return []providers.ToolCall{{ID: uuid.NewString(), Name: "os_create_excel", Input: inputBytes}}
	}

	// 6.9 Detectar intención de creación de archivo Word (.docx)
	isWordIntent := (strings.Contains(u, "word") || strings.Contains(u, "docx")) &&
		(strings.Contains(u, "crea") || strings.Contains(u, "haz") || strings.Contains(u, "genera"))
	if isWordIntent {
		fileName := "Documento_2026.docx"
		if m := regexp.MustCompile(`([a-zA-Z0-9_\-]+\.docx)`).FindString(userMessage); m != "" { fileName = m }
		targetPages := 1
		if pm := regexp.MustCompile(`(\d+)\s*p[aá]g`).FindStringSubmatch(u); len(pm) > 1 { fmt.Sscanf(pm[1], "%d", &targetPages) }
		docxPath := filepath.Join(os.Getenv("USERPROFILE"), "Documents", fileName)
		inputBytes, _ := json.Marshal(map[string]any{
			"path": docxPath, "title": strings.TrimSuffix(fileName, ".docx"), "target_pages": targetPages,
			"sections": []map[string]string{{"title": "Resumen Ejecutivo", "content": "Documento generado autónomamente por OzyAssist."}},
		})
		return []providers.ToolCall{{ID: uuid.NewString(), Name: "os_create_docx", Input: inputBytes}}
	}

	// 7. Detectar intención de creación de reporte o PDF con búsqueda implícita
	isReportIntent := (strings.Contains(u, "informe") || strings.Contains(u, "reporte") || strings.Contains(u, "pdf")) &&
		(strings.Contains(u, "crea") || strings.Contains(u, "haz") || strings.Contains(u, "genera") || strings.Contains(u, "realiza"))
	if isReportIntent {
		topic := u
		noisePrefixes := []string{
			"crea un informe sobre ", "crea un reporte sobre ", "crea un pdf sobre ", "crea un informe de ", "crea un reporte de ", "crea un pdf de ",
			"haz un informe sobre ", "haz un reporte sobre ", "haz un pdf sobre ", "haz un informe de ", "haz un reporte de ", "haz un pdf de ",
			"genera un informe sobre ", "genera un reporte sobre ", "genera un pdf sobre ", "realiza un informe sobre ", "realiza un reporte sobre ", "realiza un pdf sobre ",
		}
		for _, noise := range noisePrefixes {
			if strings.Contains(topic, noise) {
				idx := strings.Index(topic, noise)
				topic = strings.TrimSpace(topic[idx+len(noise):])
				break
			}
		}
		// Limpiar sufijos comunes
		for _, suffix := range []string{" en pdf", " en una ruta de documentos", " en documentos", " y su informacion de la web", " de la web", " para revisarlo"} {
			if strings.Contains(topic, suffix) {
				topic = strings.ReplaceAll(topic, suffix, "")
			}
		}
		topic = strings.Trim(topic, " .,!?:;\"'")
		if topic != "" && len(topic) > 2 {
			// Si pide buscar en la web, ejecutar web_search primero
			if strings.Contains(u, "web") || strings.Contains(u, "internet") || strings.Contains(u, "informacion") || strings.Contains(u, "información") {
				inputBytes, _ := json.Marshal(map[string]string{"query": topic})
				return []providers.ToolCall{{
					ID:    uuid.NewString(),
					Name:  "web_search",
					Input: inputBytes,
				}}
			}

			// Si pide PDF directamente
			docPath := filepath.Join(os.Getenv("USERPROFILE"), "Documents", strings.ReplaceAll(topic, " ", "_")+"_reporte.pdf")
			inputBytes, _ := json.Marshal(map[string]any{
				"path":  docPath,
				"title": strings.Title(topic),
				"sections": []map[string]string{
					{"title": "Resumen Ejecutivo", "content": fmt.Sprintf("Reporte detallado sobre %s", topic)},
				},
			})
			return []providers.ToolCall{{
				ID:    uuid.NewString(),
				Name:  "os_create_pdf",
				Input: inputBytes,
			}}
		}
	}

	// 7.9 Detectar búsqueda de contenido dentro de archivos (os_search_content / grep)
	isContentSearch := (strings.Contains(u, "dentro de") || strings.Contains(u, "en el contenido") ||
		strings.Contains(u, "la palabra ") || strings.Contains(u, "el texto ") ||
		strings.Contains(u, "que contenga ") || strings.Contains(u, "que contengan ")) &&
		(strings.Contains(u, "busca") || strings.Contains(u, "encuentra") || strings.Contains(u, "grep"))
	if isContentSearch {
		query := ""
		for _, prefix := range []string{"la palabra ", "el texto ", "que contenga "} {
			if idx := strings.Index(u, prefix); idx != -1 {
				rem := strings.TrimSpace(u[idx+len(prefix):])
				words := strings.Fields(rem)
				if len(words) > 0 {
					query = strings.Trim(words[0], " .,!?:;\"'")
				}
				break
			}
		}
		if query == "" {
			query = extractFileSearchPattern(u)
		}
		if query != "" {
			root := filepath.Join(os.Getenv("USERPROFILE"), "Documents")
			if strings.Contains(u, "desktop") || strings.Contains(u, "escritorio") {
				root = filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
			}
			inputBytes, _ := json.Marshal(map[string]any{
				"root":        root,
				"query":       query,
				"max_results": 20,
			})
			return []providers.ToolCall{{
				ID:    uuid.NewString(),
				Name:  "os_search_content",
				Input: inputBytes,
			}}
		}
	}

	// 8. Detectar intención de búsqueda de archivos en disco o consulta de ubicación de archivos
	isSearchAction := strings.Contains(u, "busca") || strings.Contains(u, "encuentra") ||
		strings.Contains(u, "donde") || strings.Contains(u, "dónde") ||
		strings.Contains(u, "ruta") || strings.Contains(u, "ubicacion") || strings.Contains(u, "ubicación") ||
		strings.Contains(u, "direccion") || strings.Contains(u, "dirección")
	isDocTarget := strings.Contains(u, "informe") || strings.Contains(u, "reporte") ||
		strings.Contains(u, "archivo") || strings.Contains(u, "documento") ||
		strings.Contains(u, "pdf") || strings.Contains(u, "disco")

	isFileSearchIntent := isSearchAction && isDocTarget

	if isFileSearchIntent {
		kw := extractFileSearchPattern(u)
		pattern := "*.pdf"
		if kw != "" {
			pattern = "*" + strings.ReplaceAll(kw, " ", "*") + "*"
		}
		inputBytes, _ := json.Marshal(map[string]any{
			"root":        filepath.Join(os.Getenv("USERPROFILE"), "Documents"),
			"pattern":     pattern,
			"max_results": 10,
		})
		return []providers.ToolCall{{
			ID:    uuid.NewString(),
			Name:  "os_find_files",
			Input: inputBytes,
		}}
	}

	return nil
}

var fileSearchStopwords = map[string]bool{
	"caul": true, "cual": true, "cuál": true, "que": true, "qué": true, "quien": true, "quién": true,
	"donde": true, "dónde": true, "en": true, "como": true, "cómo": true, "es": true, "son": true,
	"fue": true, "fueron": true, "esta": true, "está": true, "estan": true, "están": true, "se": true,
	"guardo": true, "guardó": true, "creo": true, "creó": true, "genero": true, "generó": true,
	"hizo": true, "encuentra": true, "encontrar": true, "busca": true, "buscar": true, "dime": true,
	"comentame": true, "coméntame": true, "por": true, "favor": true, "puedes": true, "mostrar": true,
	"ver": true, "abrir": true, "revisar": true, "la": true, "el": true, "los": true, "las": true,
	"un": true, "una": true, "unos": true, "unas": true, "de": true, "del": true, "al": true,
	"y": true, "e": true, "o": true, "u": true, "para": true, "con": true, "sobre": true, "acerca": true,
	"ruta": true, "rutas": true, "direccion": true, "dirección": true, "direcciones": true,
	"ubicacion": true, "ubicación": true, "ubicaciones": true, "archivo": true, "archivos": true,
	"informe": true, "informes": true, "reporte": true, "reportes": true, "documento": true,
	"documentos": true, "fichero": true, "ficheros": true, "carpeta": true, "carpetas": true,
	"disco": true, "duro": true, "pdf": true, "mi": true, "mis": true, "tu": true, "tus": true,
	"su": true, "sus": true, "ultimo": true, "último": true, "reciente": true,
	"correcto": true, "pantalla": true, "equipo": true, "sistema": true,
}

func extractFileSearchPattern(raw string) string {
	clean := strings.ToLower(raw)
	for _, ch := range []string{"¿", "?", "¡", "!", ",", ".", ";", ":", "\"", "'", "(", ")", "[", "]"} {
		clean = strings.ReplaceAll(clean, ch, " ")
	}
	words := strings.Fields(clean)
	var filtered []string
	for _, w := range words {
		if !fileSearchStopwords[w] && len(w) > 1 {
			filtered = append(filtered, w)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	return strings.Join(filtered, " ")
}

// isLeakedToolJSON detecta si el modelo emitió un bloque JSON de llamada a herramienta
// que no fue parseado por los analizadores estándar.
func isLeakedToolJSON(text string) bool {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return false
	}
	var tc struct {
		Name string `json:"name"`
		Call string `json:"call"`
		Tool string `json:"tool"`
	}
	if err := json.Unmarshal([]byte(trimmed), &tc); err == nil {
		if tc.Name != "" || tc.Call != "" || tc.Tool != "" {
			return true
		}
	}
	return false
}

// tryRescueLeakedJSON intenta recuperar semánticamente una llamada a herramienta alucinada
// antes de que se escape al chat del usuario como texto crudo.
func tryRescueLeakedJSON(text string) []providers.ToolCall {
	trimmed := strings.TrimSpace(text)
	var tc struct {
		Name      string          `json:"name"`
		Call      string          `json:"call"`
		Tool      string          `json:"tool"`
		Arguments json.RawMessage `json:"arguments"`
		Args      json.RawMessage `json:"args"`
	}
	if err := json.Unmarshal([]byte(trimmed), &tc); err != nil {
		return nil
	}

	name := tc.Name
	if name == "" {
		name = tc.Call
	}
	if name == "" {
		name = tc.Tool
	}
	if name == "" {
		return nil
	}

	rawArgs := tc.Arguments
	if len(rawArgs) == 0 {
		rawArgs = tc.Args
	}
	if len(rawArgs) == 0 {
		rawArgs = []byte("{}")
	}

	// 1. Si es una herramienta registrada o un alias conocido
	if canonical, ok := findRegisteredTool(name); ok {
		return []providers.ToolCall{
			{
				ID:    uuid.NewString(),
				Name:  canonical,
				Input: rawArgs,
			},
		}
	}

	// 2. Heurística de recuperación semántica
	lowerName := strings.ToLower(name)
	var argMap map[string]any
	_ = json.Unmarshal(rawArgs, &argMap)
	pathVal, _ := argMap["path"].(string)
	if pathVal == "" {
		pathVal, _ = argMap["target"].(string)
	}

	// Metadatos o inspección de archivo
	if strings.Contains(lowerName, "file") || strings.Contains(lowerName, "inspect") ||
		strings.Contains(lowerName, "info") || strings.Contains(lowerName, "stat") ||
		strings.Contains(lowerName, "meta") || strings.Contains(lowerName, "detail") {
		if pathVal == "" {
			if foc, ok := GetFocusedFile(); ok {
				pathVal = foc
			}
		}
		if pathVal != "" {
			inputBytes, _ := json.Marshal(map[string]string{"path": pathVal})
			return []providers.ToolCall{
				{
					ID:    uuid.NewString(),
					Name:  "os_file_info",
					Input: inputBytes,
				},
			}
		}
	}

	// Abrir archivo o programa
	if strings.Contains(lowerName, "open") || strings.Contains(lowerName, "launch") || strings.Contains(lowerName, "start") {
		if pathVal != "" {
			inputBytes, _ := json.Marshal(map[string]string{"target": pathVal})
			return []providers.ToolCall{
				{
					ID:    uuid.NewString(),
					Name:  "os_launch_app",
					Input: inputBytes,
				},
			}
		}
	}

	return nil
}
