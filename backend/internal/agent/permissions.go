package agent

import (
	"regexp"
	"strings"
)

type PermissionLevel string

const (
	ReadOnly  PermissionLevel = "read_only"
	Sandboxed PermissionLevel = "sandboxed"
	Trusted   PermissionLevel = "trusted"
)

var destructivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\brm\s+-rf\b`),
	regexp.MustCompile(`(?i)\brmdir\b`),
	regexp.MustCompile(`(?i)\bformat\b`),
	regexp.MustCompile(`(?i)\bdd\b`),
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\bfdisk\b`),
	regexp.MustCompile(`(?i)\bfsck\b`),
	regexp.MustCompile(`(?i)\bchmod\s+777\b`),
	regexp.MustCompile(`(?i)\bchown\b`),
	regexp.MustCompile(`(?i)\bdel\s+/[fqs]\b`),
	regexp.MustCompile(`(?i)\brmdir\s+/s\b`),
}

// IsDestructiveCommand verifica si un comando contiene patrones destructivos conocidos.
func IsDestructiveCommand(command string) bool {
	for _, p := range destructivePatterns {
		if p.MatchString(command) {
			return true
		}
	}
	return false
}

// RequiresConfirmation retorna true si la acción requiere confirmación del usuario
// según el level de permiso y el tipo de acción.
func RequiresConfirmation(level PermissionLevel, actionType, target string) bool {
	switch level {
	case ReadOnly:
		return actionType != "file_read"
	case Sandboxed:
		switch actionType {
		case "file_read":
			return false
		case "file_write":
			return strings.Contains(target, "..") // path traversal
		case "command_exec":
			return IsDestructiveCommand(target)
		case "browser_action":
			return false
		default:
			return true
		}
	case Trusted:
		return false
	}
	return true
}
