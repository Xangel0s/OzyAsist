//go:build !windows

package system

type MMCSSHandle uintptr

func OptimizeProcessPriority(high bool) error { return nil }
func LockProcessWorkingSet(minMB, maxMB int) error { return nil }
func PinProcessToPerformanceCores(pCores int) error { return nil }
func EnableMMCSSForVoice(taskName string) (MMCSSHandle, error) { return 0, nil }
func (h MMCSSHandle) Revert() {}
