package system

import (
	"runtime"
	"testing"
)

func TestOptimizeProcessPriority(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	err := OptimizeProcessPriority(false)
	if err != nil {
		t.Logf("Notice: OptimizeProcessPriority returned (might require higher privs): %v", err)
	}
}

func TestLockProcessWorkingSet(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	err := LockProcessWorkingSet(64, 512)
	if err != nil {
		t.Logf("Notice: LockProcessWorkingSet returned: %v", err)
	}
}

func TestPinProcessToPerformanceCores(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	err := PinProcessToPerformanceCores(0)
	if err != nil {
		t.Logf("Notice: PinProcessToPerformanceCores returned: %v", err)
	}
}
