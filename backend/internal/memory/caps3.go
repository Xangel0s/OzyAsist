package memory

import (
	"sync"
	"time"
)

// Caps3 almacena memoria de trabajo en RAM durante la sesión activa.
// Es volátil — se pierde al reiniciar el servidor.
// Contiene el estado actual de la conversación y contexto de la sesión.

type Caps3Entry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	ExpiresAt int64  `json:"expiresAt,omitempty"` // Unix timestamp, 0 = no expiry
}

type Caps3Store struct {
	mu  sync.RWMutex
	buf map[string]Caps3Entry
}

var caps3 *Caps3Store

func init() {
	caps3 = &Caps3Store{
		buf: make(map[string]Caps3Entry),
	}
}

func GetCaps3() *Caps3Store {
	return caps3
}

func (c *Caps3Store) Set(key, value string, ttlSeconds int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := Caps3Entry{Key: key, Value: value}
	if ttlSeconds > 0 {
		entry.ExpiresAt = time.Now().Unix() + int64(ttlSeconds)
	}
	c.buf[key] = entry
}

func (c *Caps3Store) Get(key string) (string, bool) {
	c.mu.RLock()
	entry, ok := c.buf[key]
	if !ok {
		c.mu.RUnlock()
		return "", false
	}
	if entry.ExpiresAt > 0 && time.Now().Unix() > entry.ExpiresAt {
		c.mu.RUnlock()
		c.mu.Lock()
		delete(c.buf, key)
		c.mu.Unlock()
		return "", false
	}
	c.mu.RUnlock()
	return entry.Value, true
}

func (c *Caps3Store) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.buf, key)
}

// ContextByChatID devuelve todo el buffer de trabajo como un string plano (para inyectar en prompt).
func (c *Caps3Store) ContextByChatID(chatID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result string
	for k, v := range c.buf {
		if v.ExpiresAt > 0 && time.Now().Unix() > v.ExpiresAt {
			continue
		}
		result += k + ": " + v.Value + "\n"
	}
	return result
}
