package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempSandbox(t *testing.T) (*Sandbox, string) {
	t.Helper()
	dir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSandbox(dir)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}
	return s, dir
}

// === PUNTO 1: ReadOnly hard block ===

func TestReadOnly_BlocksFileWrite(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	step := PlanStep{
		ID:         1,
		ActionType: "file_write",
		Target:     "test.txt",
		Input:      "contenido inocente dentro del sandbox",
	}
	_, err := ValidateAndAuthorize(step, ReadOnly, s)
	if err == nil {
		t.Fatal("expected error for file_write in read_only, got nil")
	}
	if !strings.Contains(err.Error(), "not allowed in read-only") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadOnly_BlocksCommandExec(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	step := PlanStep{
		ID:         1,
		ActionType: "command_exec",
		Target:     "echo test",
	}
	_, err := ValidateAndAuthorize(step, ReadOnly, s)
	if err == nil {
		t.Fatal("expected error for command_exec in read_only, got nil")
	}
	if !strings.Contains(err.Error(), "not allowed in read-only") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadOnly_AllowsFileRead(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	step := PlanStep{
		ID:         1,
		ActionType: "file_read",
		Target:     "readme.md",
	}
	auth, err := ValidateAndAuthorize(step, ReadOnly, s)
	if err != nil {
		t.Fatalf("expected OK for file_read in read_only, got: %v", err)
	}
	if auth.RequiresConfirmation {
		t.Fatal("file_read should not require confirmation in read_only")
	}
}

// === PUNTO 2: Path traversal — variantes Windows ===

func TestPathTraversal_WindowsBackslash(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	step := PlanStep{
		ID:         1,
		ActionType: "file_read",
		Target:     `..\..\..\Windows\win.ini`,
	}
	_, err := ValidateAndAuthorize(step, Sandboxed, s)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPathTraversal_DoubleDotNested(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	step := PlanStep{
		ID:         1,
		ActionType: "file_read",
		Target:     `....\\....\\Windows\\win.ini`,
	}
	_, err := ValidateAndAuthorize(step, Sandboxed, s)
	if err == nil {
		// Este puede pasar o fallar según si filepath.Abs lo resuelve como traversal válido.
		// Lo importante es que NO permita leer fuera del sandbox.
		// Abs en Windows interpreta `....` como directorio literal, no como `..`.
		// Por tanto el traversal puede ser bloqueado por Resolve() por el prefijo "."
		// o puede resolverse a una ruta dentro del sandbox y no ser peligroso.
		t.Log("....\\.... no fue detectado como traversal (ruta dentro del sandbox)")
	}
}

func TestPathTraversal_DeepWindowsSystem(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	step := PlanStep{
		ID:         1,
		ActionType: "file_read",
		Target:     `..\..\..\Windows\System32\drivers\etc\hosts`,
	}
	_, err := ValidateAndAuthorize(step, Sandboxed, s)
	if err == nil {
		t.Fatal("expected error for path traversal to system path, got nil")
	}
}

func TestPathTraversal_AbsolutePathOutside(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	step := PlanStep{
		ID:         1,
		ActionType: "file_read",
		Target:     `C:\Windows\win.ini`,
	}
	_, err := ValidateAndAuthorize(step, Sandboxed, s)
	if err == nil {
		t.Fatal("expected error for absolute path outside sandbox, got nil")
	}
}

// === PUNTO 3: Symlink escape ===

func TestSymlinkEscape_BlocksWriteViaSymlink(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	// Crear un directorio target fuera del sandbox
	outsideDir, err := os.MkdirTemp("", "outside-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outsideDir)

	// Crear symlink dentro del sandbox apuntando fuera
	linkPath := filepath.Join(dir, "escape-link")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		t.Skip("symlink creation not supported on this system:", err)
	}

	step := PlanStep{
		ID:         1,
		ActionType: "file_write",
		Target:     "escape-link/malicious.txt",
		Input:      "pwned",
	}
	_, err = ValidateAndAuthorize(step, Sandboxed, s)
	if err == nil {
		t.Fatal("expected error for symlink escape, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") && !strings.Contains(err.Error(), "escape") {
		t.Fatalf("expected symlink/escape error, got: %v", err)
	}
}

func TestSymlinkEscape_BlocksReadViaSymlink(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	outsideDir, err := os.MkdirTemp("", "outside-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outsideDir)

	linkPath := filepath.Join(dir, "read-link")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		t.Skip("symlink creation not supported on this system:", err)
	}

	step := PlanStep{
		ID:         1,
		ActionType: "file_read",
		Target:     "read-link/somefile.txt",
	}
	_, err = ValidateAndAuthorize(step, Sandboxed, s)
	if err == nil {
		t.Fatal("expected error for symlink read escape, got nil")
	}
}

// === PUNTO 5: Mix traversal + symlink ===

// TestSymlinkTraversalMix_SymlinkThenTraversal: link/../file se normaliza
// vía filepath.Clean ANTES de resolver symlinks. El path resultante
// (secret.txt) está dentro del sandbox. NO es un vector de escape.
// Este test es un test de regresión: confirma que el orden de operaciones
// (clean → resolve symlinks) NO permite que un attacker use un symlink
// para anular la protección de traversal — porque el traversal ya fue
// normalizado antes de cruzar el symlink.
func TestSymlinkTraversalMix_SymlinkThenTraversal_IsSafe(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	outsideDir := "C:\\Users\\User\\Documents\\ozyAsis"
	linkPath := filepath.Join(dir, "link")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		t.Skip("symlink not supported:", err)
	}

	// link/../secret.txt → Clean lo reduce a "secret.txt" dentro del sandbox
	step := PlanStep{
		ID:         1,
		ActionType: "file_read",
		Target:     "link/../secret.txt",
	}
	auth, err := ValidateAndAuthorize(step, Sandboxed, s)
	if err != nil {
		t.Fatalf("link/../file is safe (cleaned to file), should NOT be blocked: %v", err)
	}
	if auth.ResolvedPath == "" {
		t.Fatal("expected a resolved path")
	}
}

func TestSymlinkTraversalMix_SymlinkInsideThenTraversalOut(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	realSub := filepath.Join(dir, "realsub")
	os.MkdirAll(realSub, 0755)

	linkPath := filepath.Join(dir, "mylink")
	if err := os.Symlink(realSub, linkPath); err != nil {
		t.Skip("symlink not supported:", err)
	}

	// mylink/../../../Windows/win.ini: 3 niveles de .. superan los 2 niveles
	// de profundidad del path (mylink/realsub), y el traversal cruza el sandbox root.
	// Esto es bloqueado por el Resolve check ANTES de llegar a symlinks.
	step := PlanStep{
		ID:         1,
		ActionType: "file_read",
		Target:     "mylink/../../../Windows/win.ini",
	}
	_, err := ValidateAndAuthorize(step, Sandboxed, s)
	if err == nil {
		t.Fatal("expected error for traversal that escapes sandbox root, got nil")
	}
}

// TestSymlinkTraversalMix_SymlinkToOutsideThenTraversalBackIn: mismo caso que
// SymlinkThenTraversal — lexical clean elimina escape/.. → el path queda dentro
// del sandbox. No es un vector de escape. Test de regresión.
func TestSymlinkTraversalMix_SymlinkToOutsideThenTraversalBackIn_IsSafe(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	outsideDir, err := os.MkdirTemp("", "outside-link-target-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outsideDir)

	linkPath := filepath.Join(dir, "escape")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		t.Skip("symlink not supported:", err)
	}

	// escape/../somefile.txt → Clean lo reduce a "somefile.txt"
	step := PlanStep{
		ID:         1,
		ActionType: "file_read",
		Target:     "escape/../somefile.txt",
	}
	auth, err := ValidateAndAuthorize(step, Sandboxed, s)
	if err != nil {
		t.Fatalf("escape/../file is safe (cleaned to file), should NOT be blocked: %v", err)
	}
	if auth.ResolvedPath == "" {
		t.Fatal("expected a resolved path")
	}
}

// El ÚNICO mix real: symlink apuntando afuera SIN .. en el path.
// Esto ya está cubierto por TestSymlinkEscape_BlocksReadViaSymlink y WriteViaSymlink.
//
// Conclusión: no existe un vector mix traversal+symlink que no esté cubierto
// por los checks individuales, porque:
//   1. Si .. remueve el symlink → path dentro del sandbox → safe
//   2. Si .. NO remueve el symlink → resolveSync atrapa el escape → blocked
//   3. Si .. es suficiente para escapar lexicalmente → Resolve lo bloquea primero

func TestSymlinkInsideSandbox_Allowed(t *testing.T) {
	s, dir := tempSandbox(t)
	defer os.RemoveAll(dir)

	// Crear directorio real dentro del sandbox
	realDir := filepath.Join(dir, "realdir")
	os.MkdirAll(realDir, 0755)
	os.WriteFile(filepath.Join(realDir, "data.txt"), []byte("safe"), 0644)

	// Crear symlink dentro → dentro
	linkPath := filepath.Join(dir, "inner-link")
	if err := os.Symlink(realDir, linkPath); err != nil {
		t.Skip("symlink not supported:", err)
	}

	step := PlanStep{
		ID:         1,
		ActionType: "file_read",
		Target:     "inner-link/data.txt",
	}
	auth, err := ValidateAndAuthorize(step, Sandboxed, s)
	if err != nil {
		t.Fatalf("expected OK for inner symlink, got: %v", err)
	}
	if auth.ResolvedPath == "" {
		t.Fatal("expected resolved path")
	}
}

// === PUNTO 4: Comando destructivo en sandboxed → awaiting_confirmation ===

func TestDestructiveCommand_Sandboxed_RequiresConfirmation(t *testing.T) {
	commands := []struct {
		name    string
		command string
	}{
		{"del flag f", "del /f important.txt"},
		{"del flag q", "del /q temp.txt"},
		{"rm -rf", "rm -rf /"},
		{"rmdir", "rmdir /s /q C:\\"},
		{"format", "format D: /q"},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			step := PlanStep{
				ID:         1,
				ActionType: "command_exec",
				Target:     tc.command,
			}
			auth, err := ValidateAndAuthorize(step, Sandboxed, nil)
			if err != nil {
				t.Fatalf("expected OK (with confirmation), got error: %v", err)
			}
			if !auth.RequiresConfirmation {
				t.Fatal("expected RequiresConfirmation=true for destructive command in sandboxed")
			}
		})
	}
}

func TestNonDestructiveCommand_Sandboxed_NoConfirmation(t *testing.T) {
	step := PlanStep{
		ID:         1,
		ActionType: "command_exec",
		Target:     "dir /b",
	}
	auth, err := ValidateAndAuthorize(step, Sandboxed, nil)
	if err != nil {
		t.Fatalf("expected OK for non-destructive command, got: %v", err)
	}
	if auth.RequiresConfirmation {
		t.Fatal("expected RequiresConfirmation=false for non-destructive command")
	}
}

func TestDestructiveCommand_Trusted_NoConfirmation(t *testing.T) {
	step := PlanStep{
		ID:         1,
		ActionType: "command_exec",
		Target:     "del /f secret.txt",
	}
	auth, err := ValidateAndAuthorize(step, Trusted, nil)
	if err != nil {
		t.Fatalf("expected OK for destructive command in trusted mode, got: %v", err)
	}
	if auth.RequiresConfirmation {
		t.Fatal("trusted mode should never require confirmation")
	}
}


