package agent

import (
	"fmt"
	"strings"
	"testing"
)

func TestSmartFoldOutput_ANSIAndLines(t *testing.T) {
	// 1. Prueba de eliminación de secuencias ANSI
	ansiText := "\x1b[32mSUCCESS:\x1b[0m Archivo compilado correctamente \x1b[1;34m[OK]\x1b[0m"
	stripped := StripANSI(ansiText)
	if strings.Contains(stripped, "\x1b") {
		t.Fatalf("StripANSI no eliminó todas las secuencias ANSI: %s", stripped)
	}
	if stripped != "SUCCESS: Archivo compilado correctamente [OK]" {
		t.Fatalf("Salida inesperada tras StripANSI: %s", stripped)
	}
	t.Logf("✓ StripANSI limpió secuencias correctamente: %s", stripped)

	// 2. Prueba de plegado de 100 líneas a menos de 25 líneas
	var longOutput strings.Builder
	for i := 1; i <= 100; i++ {
		longOutput.WriteString(fmt.Sprintf("Línea %d de ejecución...\n", i))
	}
	longOutput.WriteString("COMPILACIÓN EXITOSA: 0 errores encontrados.")

	folded := SmartFoldOutput(longOutput.String(), 20)
	foldedLines := strings.Split(folded, "\n")

	t.Logf("✓ Salida original de 101 líneas compactada a %d líneas", len(foldedLines))

	if len(foldedLines) > 30 {
		t.Fatalf("SmartFoldOutput no redujo las líneas adecuadamente (%d líneas)", len(foldedLines))
	}
	if !strings.Contains(folded, "Línea 1 de ejecución") {
		t.Fatalf("SmartFoldOutput perdió las líneas iniciales de encabezado")
	}
	if !strings.Contains(folded, "COMPILACIÓN EXITOSA") {
		t.Fatalf("SmartFoldOutput perdió la línea final de resultado")
	}
	if !strings.Contains(folded, "líneas intermedias omitidas") {
		t.Fatalf("SmartFoldOutput no incluyó el marcador de omisión")
	}
}
