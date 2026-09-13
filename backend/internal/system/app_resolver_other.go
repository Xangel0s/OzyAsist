//go:build !windows

package system

import "fmt"

type AppLaunchInfo struct {
	Command string
	IsAUMID bool
}

func ResolveAppExecutable(name string) (*AppLaunchInfo, error) {
	if name == "" {
		return nil, fmt.Errorf("app name cannot be empty")
	}
	return &AppLaunchInfo{Command: name}, nil
}
