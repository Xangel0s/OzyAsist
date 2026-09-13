package system

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestOSNavigator_GetKnownFolders(t *testing.T) {
	nav := NewWindowsNavigator()
	folders, err := nav.GetKnownFolders(context.Background())
	if err != nil {
		t.Fatalf("GetKnownFolders failed: %v", err)
	}

	if folders.UserDesktop == "" {
		t.Errorf("expected UserDesktop to be populated, got empty")
	}
	t.Logf("Known Folders User Desktop: %s", folders.UserDesktop)
}

func TestOSNavigator_GetDesktopItems(t *testing.T) {
	nav := NewWindowsNavigator()
	items, err := nav.GetDesktopItems(context.Background())
	if err != nil {
		t.Fatalf("GetDesktopItems failed: %v", err)
	}

	t.Logf("Desktop items found: %d", len(items))
	for _, it := range items {
		t.Logf(" - [%s] %s (%s)", it.Kind, it.Name, it.Path)
	}

	if len(items) == 0 {
		t.Errorf("expected at least standard desktop items or system icons")
	}
}

func TestOSNavigator_GetInstalledSoftware(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping Windows registry test on non-windows")
	}

	nav := NewWindowsNavigator()
	apps, err := nav.GetInstalledSoftware(context.Background(), "")
	if err != nil {
		t.Fatalf("GetInstalledSoftware failed: %v", err)
	}

	t.Logf("Total installed software found via Registry: %d", len(apps))
	if len(apps) == 0 {
		t.Errorf("expected to find installed apps in Windows registry")
	}

	// Test with a filter
	filtered, _ := nav.GetInstalledSoftware(context.Background(), "code")
	t.Logf("Filtered apps matching 'code': %d", len(filtered))
}

func TestOSNavigator_ExplorePath(t *testing.T) {
	nav := NewWindowsNavigator()
	node, err := nav.ExplorePath(context.Background(), ".", 1)
	if err != nil {
		t.Fatalf("ExplorePath failed: %v", err)
	}

	if node == nil || !node.IsDir {
		t.Fatalf("expected root node to be directory")
	}
	t.Logf("Explored directory '%s' with %d children", node.Path, len(node.Children))
}

func TestOSNavigator_FindFiles(t *testing.T) {
	nav := NewWindowsNavigator()
	matches, err := nav.FindFiles(context.Background(), ".", "*.go", 10)
	if err != nil {
		t.Fatalf("FindFiles failed: %v", err)
	}

	t.Logf("Found %d Go files in root", len(matches))
	if len(matches) == 0 {
		t.Errorf("expected to find Go files")
	}
}

func TestOSNavigator_FileManipulation(t *testing.T) {
	nav := NewWindowsNavigator()
	ctx := context.Background()

	tmpDir := t.TempDir()
	testFolder := tmpDir + "/test_ozy_subfolder"

	// 1. Create directory
	if err := nav.CreateDirectory(ctx, testFolder); err != nil {
		t.Fatalf("CreateDirectory failed: %v", err)
	}

	// 2. Copy and move test files
	fileA := testFolder + "/fileA.txt"
	fileB := testFolder + "/fileB.txt"
	fileMoved := testFolder + "/fileA_moved.txt"

	_ = nav.CreateDirectory(ctx, testFolder)
	if err := os.WriteFile(fileA, []byte("Hello Ozy"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := nav.CopyItem(ctx, fileA, fileB); err != nil {
		t.Fatalf("CopyItem failed: %v", err)
	}

	if err := nav.MoveItem(ctx, fileA, fileMoved); err != nil {
		t.Fatalf("MoveItem failed: %v", err)
	}

	// 3. Delete file
	if err := nav.DeleteItem(ctx, fileB, false); err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
}

func TestOSNavigator_OrganizeFolder(t *testing.T) {
	nav := NewWindowsNavigator()
	ctx := context.Background()

	tmpDir := t.TempDir()

	// Crear archivos de distintas extensiones
	_ = os.WriteFile(tmpDir+"/doc1.pdf", []byte("pdf"), 0644)
	_ = os.WriteFile(tmpDir+"/img1.png", []byte("png"), 0644)
	_ = os.WriteFile(tmpDir+"/code1.go", []byte("go"), 0644)
	_ = os.WriteFile(tmpDir+"/installer.exe", []byte("exe"), 0644)

	res, err := nav.OrganizeFolder(ctx, tmpDir, "extension")
	if err != nil {
		t.Fatalf("OrganizeFolder failed: %v", err)
	}

	t.Logf("Organize result: %d files moved into %d folders", res.MovedFiles, len(res.CreatedFolders))
	if res.MovedFiles != 4 {
		t.Errorf("expected 4 moved files, got %d", res.MovedFiles)
	}
}

func TestOSNavigator_LaunchAndDetectWindow(t *testing.T) {
	nav := NewWindowsNavigator()
	ctx := context.Background()

	err := nav.LaunchApplication(ctx, "explorer", []string{"documents"})
	if err != nil {
		t.Fatalf("LaunchApplication failed: %v", err)
	}

	var foundWin *WindowInfo
	var lastWins []WindowInfo
	for attempt := 0; attempt < 5; attempt++ {
		time.Sleep(500 * time.Millisecond)
		wins, err := nav.GetActiveWindows(ctx)
		if err != nil {
			continue
		}
		lastWins = wins
		for _, w := range wins {
			if strings.Contains(strings.ToLower(w.Title), "documents") || strings.Contains(strings.ToLower(w.Title), "documentos") || strings.Contains(strings.ToLower(w.Title), "explorador") {
				copyWin := w
				foundWin = &copyWin
				break
			}
		}
		if foundWin != nil {
			break
		}
	}

	if foundWin == nil {
		for _, w := range lastWins {
			t.Logf("Window: HWND=%d, PID=%d, Title='%s'", w.Handle, w.ProcessID, w.Title)
		}
		t.Fatalf("Explorer window was not found in %d active windows", len(lastWins))
	}

	t.Logf("SUCCESS: Found window: HWND=%d, Title='%s'", foundWin.Handle, foundWin.Title)
	if err := nav.FocusWindow(ctx, foundWin.Handle); err != nil {
		t.Errorf("FocusWindow failed: %v", err)
	}
}

func TestOSNavigator_LaunchCalculator(t *testing.T) {
	nav := NewWindowsNavigator()
	ctx := context.Background()

	_ = exec.Command("powershell", "-NoProfile", "-Command", "Stop-Process -Name CalculatorApp, calc -Force -ErrorAction SilentlyContinue").Run()
	time.Sleep(500 * time.Millisecond)

	err := nav.LaunchApplication(ctx, "calc.exe", nil)
	if err != nil {
		t.Fatalf("LaunchApplication failed: %v", err)
	}

	time.Sleep(2000 * time.Millisecond)

	wins, err := nav.GetActiveWindows(ctx)
	if err != nil {
		t.Fatalf("GetActiveWindows failed: %v", err)
	}

	var foundCalc *WindowInfo
	for _, w := range wins {
		if strings.Contains(strings.ToLower(w.Title), "calculadora") || strings.Contains(strings.ToLower(w.Title), "calc") {
			copyWin := w
			foundCalc = &copyWin
			break
		}
	}

	if foundCalc == nil {
		t.Fatalf("Calculator window was not found in %d active windows", len(wins))
	}

	t.Logf("SUCCESS: Found Calculator window: HWND=%d, Title='%s'", foundCalc.Handle, foundCalc.Title)
	_ = exec.Command("powershell", "-NoProfile", "-Command", "Stop-Process -Name CalculatorApp, calc -Force -ErrorAction SilentlyContinue").Run()
}

