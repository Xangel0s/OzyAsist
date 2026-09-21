package agent

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FocusedEntity almacena el último archivo o directorio sobre el cual el usuario o el agente operaron.
type FocusedEntity struct {
	FilePath string
	FileName string
	DirPath  string
	Topic    string
}

var (
	focusedMu       sync.RWMutex
	sessionFocused  = make(map[string]FocusedEntity)
	lastGlobalFocus FocusedEntity
)

// SetFocusedFile registra el archivo actualmente en foco en la conversación.
func SetFocusedFile(filePath string, topic ...string) {
	SetFocusedFileFor("", filePath, topic...)
}

// SetFocusedFileFor registra el archivo en foco para una sesión o chatID específico.
func SetFocusedFileFor(sessionID, filePath string, topic ...string) {
	if filePath == "" {
		return
	}
	clean := filepath.Clean(filePath)
	top := ""
	if len(topic) > 0 && topic[0] != "" {
		top = strings.ToLower(strings.TrimSpace(topic[0]))
	}

	entity := FocusedEntity{
		FilePath: clean,
		FileName: filepath.Base(clean),
		DirPath:  filepath.Dir(clean),
		Topic:    top,
	}

	focusedMu.Lock()
	defer focusedMu.Unlock()

	lastGlobalFocus = entity
	if sessionID != "" {
		sessionFocused[sessionID] = entity
	}
}

// GetFocusedFile retorna la ruta del archivo en foco si aún existe en disco.
func GetFocusedFile() (string, bool) {
	return GetFocusedFileFor("")
}

// GetFocusedFileFor retorna la ruta del archivo en foco para la sesión dada (o el foco global).
func GetFocusedFileFor(sessionID string) (string, bool) {
	focusedMu.RLock()
	defer focusedMu.RUnlock()

	var entity FocusedEntity
	if sessionID != "" {
		if e, ok := sessionFocused[sessionID]; ok && e.FilePath != "" {
			entity = e
		}
	}
	if entity.FilePath == "" {
		entity = lastGlobalFocus
	}

	if entity.FilePath == "" {
		return "", false
	}
	if _, err := os.Stat(entity.FilePath); err == nil {
		return entity.FilePath, true
	}
	return "", false
}

// GetFocusedEntity retorna la entidad en foco actual para una sesión.
func GetFocusedEntity(sessionID ...string) FocusedEntity {
	focusedMu.RLock()
	defer focusedMu.RUnlock()

	if len(sessionID) > 0 && sessionID[0] != "" {
		if e, ok := sessionFocused[sessionID[0]]; ok {
			return e
		}
	}
	return lastGlobalFocus
}

// ClearFocusedEntity limpia la entidad en foco.
func ClearFocusedEntity(sessionID ...string) {
	focusedMu.Lock()
	defer focusedMu.Unlock()

	if len(sessionID) > 0 && sessionID[0] != "" {
		delete(sessionFocused, sessionID[0])
	}
	lastGlobalFocus = FocusedEntity{}
}
