package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCallRustSkeletonize(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Crear archivo Go temporal de prueba con funciones y estructuras
	tmpDir := t.TempDir()
	sampleGo := filepath.Join(tmpDir, "sample.go")
	content := `package sample

import (
	"fmt"
	"strings"
)

type User struct {
	ID   int
	Name string
}

func ProcessUser(u User) string {
	val := strings.ToUpper(u.Name)
	for i := 0; i < 10; i++ {
		fmt.Println(val)
	}
	return val
}
`
	if err := os.WriteFile(sampleGo, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write sample Go file: %v", err)
	}

	out, err := CallRustSkeletonize(ctx, sampleGo)
	if err != nil {
		t.Fatalf("CallRustSkeletonize failed: %v", err)
	}

	t.Logf("Rust Skeletonizer output:\n%s", out)

	if !strings.Contains(out, "SKELETON CODE OUTLINE") {
		t.Errorf("expected skeleton marker in output")
	}
	if !strings.Contains(out, "func ProcessUser(u User) string { ... }") {
		t.Errorf("expected function body to be collapsed to signature + { ... }")
	}
	if !strings.Contains(out, "type User struct") {
		t.Errorf("expected type User struct to be preserved")
	}
}

func TestCallRustInspectUI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := CallRustInspectUI(ctx)
	if err != nil {
		t.Fatalf("CallRustInspectUI failed: %v", err)
	}

	t.Logf("Rust UI Inspect output:\n%s", out)
	if out == "" {
		t.Errorf("expected non-empty output from inspect-ui")
	}
}
