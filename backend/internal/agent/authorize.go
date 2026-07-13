package agent

import (
	"fmt"
	"path/filepath"
)

// AuthorizedAction es el resultado de ValidateAndAuthorize.
// Se pasa como parámetro REQUERIDO a cada exec* — no hay camino
// de ejecución que no pase por ValidateAndAuthorize.
type AuthorizedAction struct {
	Step                PlanStep
	ResolvedPath        string // file ops: ruta real (symlinks resueltos)
	RequiresConfirmation bool  // true si en sandboxed + comando destructivo
}

// ValidateAndAuthorize es la ÚNICA puerta de entrada para cualquier
// operación del agente. Resuelve la ruta (incluyendo symlinks) y
// verifica el nivel de permiso antes de permitir la ejecución.
//
// Retorna:
//   - AuthorizedAction si la operación está permitida (con o sin confirmación)
//   - error si la operación está bloqueada por completo (ReadOnly, path escape, symlink fuera del sandbox)
// resolveSync recursively resolves symlinks on the deepest existing ancestor of path.
// For file_write targets that don't exist yet, this still catches symlinks in parent dirs.
func resolveSync(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	// File may not exist — walk up until we find something that does
	ancestor := filepath.Dir(path)
	if ancestor == path {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}
	resolvedAncestor, err := resolveSync(ancestor)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(ancestor, path)
	if err != nil {
		return "", fmt.Errorf("path reconstruction: %w", err)
	}
	return filepath.Join(resolvedAncestor, rel), nil
}

func ValidateAndAuthorize(step PlanStep, level PermissionLevel, sandbox *Sandbox) (*AuthorizedAction, error) {
	auth := &AuthorizedAction{Step: step}

	switch step.ActionType {
	case "file_read", "file_write":
		if sandbox == nil {
			return nil, fmt.Errorf("sandbox required for %s", step.ActionType)
		}

		// 1. Resolve path traversal (../../)
		resolved, err := sandbox.Resolve(step.Target)
		if err != nil {
			return nil, fmt.Errorf("path resolution: %w", err)
		}

		// 2. Follow symlinks on the deepest existing ancestor
		realPath, err := resolveSync(resolved)
		if err != nil {
			return nil, fmt.Errorf("symlink resolution: %w", err)
		}

		// 3. Re-check against real project root (also resolved)
		realRoot, err := filepath.EvalSymlinks(sandbox.ProjectRoot)
		if err != nil {
			return nil, fmt.Errorf("root symlink resolution: %w", err)
		}

		rel, err := filepath.Rel(realRoot, realPath)
		if err != nil {
			return nil, fmt.Errorf("path outside sandbox (rel error): %w", err)
		}
		if len(rel) > 0 && rel[0] == '.' {
			return nil, fmt.Errorf("path escapes sandbox via symlink: %s", rel)
		}

		auth.ResolvedPath = realPath

		// 4. Permission check for writes
		if step.ActionType == "file_write" && level == ReadOnly {
			return nil, fmt.Errorf("write operations not allowed in read-only mode")
		}

	case "command_exec":
		if level == ReadOnly {
			return nil, fmt.Errorf("command execution not allowed in read-only mode")
		}
		if level == Sandboxed && IsDestructiveCommand(step.Target) {
			auth.RequiresConfirmation = true
		}

	case "browser_action":
		return nil, fmt.Errorf("browser actions not implemented yet")

	case "llm_reason":
		// siempre permitido

	default:
		return nil, fmt.Errorf("unknown action type: %s", step.ActionType)
	}

	return auth, nil
}
