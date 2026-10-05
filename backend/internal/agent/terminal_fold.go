package agent

import (
	"fmt"
	"regexp"
	"strings"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\([a-zA-Z]`)

// StripANSI elimina secuencias de escape ANSI de colores y cursores para ahorrar tokens limpios.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// SmartFoldOutput compacta la salida de consola de comandos largos (ej: npm, git, go test, cargo)
// reteniendo el encabezado inicial y el pie final con el resultado/error, ahorrando hasta un 90% de tokens.
func SmartFoldOutput(raw string, maxLines int) string {
	clean := StripANSI(raw)
	clean = strings.ReplaceAll(clean, "\r\n", "\n")
	clean = strings.ReplaceAll(clean, "\r", "\n")

	lines := strings.Split(clean, "\n")
	var filtered []string
	for _, l := range lines {
		trimmed := strings.TrimRight(l, " \t")
		// Omitir líneas completamente vacías repetidas
		if trimmed == "" && len(filtered) > 0 && filtered[len(filtered)-1] == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}

	if len(filtered) <= maxLines {
		return strings.TrimSpace(strings.Join(filtered, "\n"))
	}

	headCount := 6
	tailCount := maxLines - headCount
	if tailCount < 8 {
		tailCount = 8
	}
	if headCount+tailCount >= len(filtered) {
		return strings.TrimSpace(strings.Join(filtered, "\n"))
	}

	head := filtered[:headCount]
	tail := filtered[len(filtered)-tailCount:]
	omitted := len(filtered) - (headCount + tailCount)

	var sb strings.Builder
	sb.WriteString(strings.Join(head, "\n"))
	sb.WriteString(fmt.Sprintf("\n\n[... %d líneas intermedias omitidas para optimizar ventana de contexto ...]\n\n", omitted))
	sb.WriteString(strings.Join(tail, "\n"))

	return strings.TrimSpace(sb.String())
}
