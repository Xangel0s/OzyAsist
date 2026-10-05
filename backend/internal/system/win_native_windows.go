//go:build windows

package system

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	procSetPriorityClass         = modKernel32.NewProc("SetPriorityClass")
	procGetCurrentProcess        = modKernel32.NewProc("GetCurrentProcess")
	procSetProcessWorkingSetSize = modKernel32.NewProc("SetProcessWorkingSetSize")
	procGetProcessAffinityMask   = modKernel32.NewProc("GetProcessAffinityMask")
	procSetProcessAffinityMask   = modKernel32.NewProc("SetProcessAffinityMask")

	modAvrt                             = syscall.NewLazyDLL("avrt.dll")
	procAvSetMmThreadCharacteristicsW   = modAvrt.NewProc("AvSetMmThreadCharacteristicsW")
	procAvRevertMmThreadCharacteristics = modAvrt.NewProc("AvRevertMmThreadCharacteristics")
)

const (
	HIGH_PRIORITY_CLASS         = 0x00000080
	ABOVE_NORMAL_PRIORITY_CLASS = 0x00008000
)

// PinProcessToPerformanceCores fija la afinidad de hilos a los núcleos físicos de alto rendimiento (P-Cores),
// evitando que el planificador de Windows 11 asigne la inferencia o procesamiento a los E-Cores.
func PinProcessToPerformanceCores(pCores int) error {
	hProc, _, _ := procGetCurrentProcess.Call()
	var procMask, sysMask uintptr
	r, _, err := procGetProcessAffinityMask.Call(
		hProc,
		uintptr(unsafe.Pointer(&procMask)),
		uintptr(unsafe.Pointer(&sysMask)),
	)
	if r == 0 {
		return err
	}

	total := runtime.NumCPU()
	if pCores <= 0 {
		if total > 8 {
			pCores = 8
		} else {
			pCores = total
		}
	}

	var targetMask uintptr
	if pCores >= 64 {
		targetMask = sysMask
	} else {
		targetMask = (uintptr(1) << pCores) - 1
	}
	targetMask = targetMask & sysMask
	if targetMask == 0 {
		targetMask = procMask
	}

	r2, _, err2 := procSetProcessAffinityMask.Call(hProc, targetMask)
	if r2 == 0 {
		return err2
	}
	return nil
}

// OptimizeProcessPriority adjusts the Windows process priority class to High or Above Normal
// to ensure the AI agent and local inference are prioritized by the Windows kernel scheduler.
func OptimizeProcessPriority(high bool) error {
	hProc, _, _ := procGetCurrentProcess.Call()
	priority := uintptr(ABOVE_NORMAL_PRIORITY_CLASS)
	if high {
		priority = uintptr(HIGH_PRIORITY_CLASS)
	}
	r, _, err := procSetPriorityClass.Call(hProc, priority)
	if r == 0 {
		return err
	}
	return nil
}

// LockProcessWorkingSet advises Windows kernel to expand and prioritize the physical memory
// working set for OzyAssist, mitigating pagefile swapping on SSDs for in-memory caches.
func LockProcessWorkingSet(minMB, maxMB int) error {
	hProc, _, _ := procGetCurrentProcess.Call()
	minBytes := uintptr(minMB * 1024 * 1024)
	maxBytes := uintptr(maxMB * 1024 * 1024)
	r, _, err := procSetProcessWorkingSetSize.Call(hProc, minBytes, maxBytes)
	if r == 0 {
		return err
	}
	return nil
}

// MMCSSHandle represents a registration with Windows Multimedia Class Scheduler Service.
type MMCSSHandle uintptr

// EnableMMCSSForVoice associates the calling thread with Windows MMCSS (e.g. "Pro Audio")
// ensuring zero jitter and real-time scheduling for streaming voice and sentence synthesis.
func EnableMMCSSForVoice(taskName string) (MMCSSHandle, error) {
	if taskName == "" {
		taskName = "Pro Audio"
	}
	taskNameUTF16, err := syscall.UTF16PtrFromString(taskName)
	if err != nil {
		return 0, err
	}
	var taskIndex uint32
	h, _, callErr := procAvSetMmThreadCharacteristicsW.Call(
		uintptr(unsafe.Pointer(taskNameUTF16)),
		uintptr(unsafe.Pointer(&taskIndex)),
	)
	if h == 0 {
		return 0, callErr
	}
	return MMCSSHandle(h), nil
}

// Revert releases the MMCSS registration.
func (h MMCSSHandle) Revert() {
	if h != 0 {
		_, _, _ = procAvRevertMmThreadCharacteristics.Call(uintptr(h))
	}
}
