//go:build !windows

package system

import "fmt"

// ResolveLnkTarget no-op en entornos Unix
func ResolveLnkTarget(lnkPath string) (string, error) {
	return "", fmt.Errorf("accesos directos .lnk solo soportados en Windows")
}
