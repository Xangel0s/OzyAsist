package memory

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// KVStore define el contrato unificado de almacenamiento clave-valor en memoria
type KVStore interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key string, val string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Flush(ctx context.Context) error
	Close() error
}

var (
	defaultCache   KVStore
	defaultCacheMu sync.RWMutex
)

// DefaultCache retorna la instancia global del caché (RAM por defecto, o Redis si fue configurado)
func DefaultCache() KVStore {
	defaultCacheMu.RLock()
	c := defaultCache
	defaultCacheMu.RUnlock()
	if c != nil {
		return c
	}

	defaultCacheMu.Lock()
	defer defaultCacheMu.Unlock()
	if defaultCache == nil {
		defaultCache = InitCache("")
	}
	return defaultCache
}

// SetDefaultCache permite reemplazar la instancia global (útil para pruebas)
func SetDefaultCache(store KVStore) {
	defaultCacheMu.Lock()
	defer defaultCacheMu.Unlock()
	defaultCache = store
}

// InitCache inicializa el caché detectando si existe REDIS_URL / REDIS_ADDR
// Si Redis no está disponible o falla, opera en modo RAM puro (Zero-Docker)
func InitCache(customRedisURL string) KVStore {
	targetURL := customRedisURL
	if targetURL == "" {
		targetURL = os.Getenv("REDIS_URL")
		if targetURL == "" {
			targetURL = os.Getenv("REDIS_ADDR")
		}
	}

	memStore := NewMemoryKVStore()

	if targetURL != "" {
		redisStore, err := NewRedisKVStore(targetURL)
		if err == nil {
			log.Printf("[CACHE] Conectado exitosamente a Redis: %s", sanitizeRedisURL(targetURL))
			return NewHybridStore(redisStore, memStore)
		}
		log.Printf("[CACHE] Redis no accesible (%v). Fallback a motor en RAM puro (Zero-Docker)", err)
	} else {
		log.Printf("[CACHE] Operando en motor RAM puro (Zero-Docker)")
	}

	return memStore
}

// -----------------------------------------------------------------------
// MemoryKVStore: Implementación en memoria RAM pura (Thread-Safe con TTL)
// -----------------------------------------------------------------------

type memoryItem struct {
	value     string
	expiresAt time.Time
}

type MemoryKVStore struct {
	mu        sync.RWMutex
	items     map[string]memoryItem
	stopClean chan struct{}
}

// NewMemoryKVStore crea un nuevo almacén en memoria RAM con recolección periódica
func NewMemoryKVStore() *MemoryKVStore {
	store := &MemoryKVStore{
		items:     make(map[string]memoryItem),
		stopClean: make(chan struct{}),
	}
	go store.cleanupLoop()
	return store
}

func (m *MemoryKVStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.evictExpired()
		case <-m.stopClean:
			return
		}
	}
}

func (m *MemoryKVStore) evictExpired() {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	for k, item := range m.items {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(m.items, k)
		}
	}
}

func (m *MemoryKVStore) Get(_ context.Context, key string) (string, bool, error) {
	m.mu.RLock()
	item, found := m.items[key]
	m.mu.RUnlock()

	if !found {
		return "", false, nil
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return "", false, nil
	}

	return item.value, true, nil
}

func (m *MemoryKVStore) Set(_ context.Context, key string, val string, ttl time.Duration) error {
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.mu.Lock()
	m.items[key] = memoryItem{
		value:     val,
		expiresAt: expiresAt,
	}
	m.mu.Unlock()
	return nil
}

func (m *MemoryKVStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()
	return nil
}

func (m *MemoryKVStore) Flush(_ context.Context) error {
	m.mu.Lock()
	m.items = make(map[string]memoryItem)
	m.mu.Unlock()
	return nil
}

func (m *MemoryKVStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	select {
	case <-m.stopClean:
	default:
		close(m.stopClean)
	}
	return nil
}

// -----------------------------------------------------------------------
// HybridStore: Intenta Redis; si falla o cae, delega silenciosamente en RAM
// -----------------------------------------------------------------------

type HybridStore struct {
	redis KVStore
	mem   KVStore
}

func NewHybridStore(redis, mem KVStore) *HybridStore {
	return &HybridStore{redis: redis, mem: mem}
}

func (h *HybridStore) Get(ctx context.Context, key string) (string, bool, error) {
	val, found, err := h.redis.Get(ctx, key)
	if err == nil && found {
		return val, true, nil
	}
	// Fallback a RAM
	return h.mem.Get(ctx, key)
}

func (h *HybridStore) Set(ctx context.Context, key string, val string, ttl time.Duration) error {
	// Guardar en RAM siempre para lectura ultrarrápida
	_ = h.mem.Set(ctx, key, val, ttl)
	// E intentar guardar en Redis
	return h.redis.Set(ctx, key, val, ttl)
}

func (h *HybridStore) Delete(ctx context.Context, key string) error {
	_ = h.mem.Delete(ctx, key)
	return h.redis.Delete(ctx, key)
}

func (h *HybridStore) Flush(ctx context.Context) error {
	_ = h.mem.Flush(ctx)
	return h.redis.Flush(ctx)
}

func (h *HybridStore) Close() error {
	_ = h.mem.Close()
	return h.redis.Close()
}

// -----------------------------------------------------------------------
// RedisKVStore: Cliente RESP nativo en Go puro (Sin dependencias externas)
// -----------------------------------------------------------------------

type RedisKVStore struct {
	addr     string
	password string
	db       int
	mu       sync.Mutex
	conn     net.Conn
	reader   *bufio.Reader
}

// NewRedisKVStore conecta a un servidor Redis usando protocolo RESP nativo
func NewRedisKVStore(rawURL string) (*RedisKVStore, error) {
	addr, password, db, err := parseRedisTarget(rawURL)
	if err != nil {
		return nil, err
	}

	store := &RedisKVStore{
		addr:     addr,
		password: password,
		db:       db,
	}

	if err := store.reconnect(); err != nil {
		return nil, err
	}

	// Verificar con PING
	resp, err := store.execCommand("PING")
	if err != nil || resp != "PONG" {
		_ = store.Close()
		return nil, fmt.Errorf("redis ping falló: %v", err)
	}

	return store, nil
}

func parseRedisTarget(raw string) (addr string, password string, db int, err error) {
	if !strings.Contains(raw, "://") {
		// Asumir "host:port"
		return raw, "", 0, nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", "", 0, err
	}

	addr = u.Host
	if !strings.Contains(addr, ":") {
		addr = addr + ":6379"
	}

	if u.User != nil {
		password, _ = u.User.Password()
	}

	db = 0
	trimmedPath := strings.TrimPrefix(u.Path, "/")
	if trimmedPath != "" {
		if d, e := strconv.Atoi(trimmedPath); e == nil {
			db = d
		}
	}

	return addr, password, db, nil
}

func sanitizeRedisURL(raw string) string {
	if strings.Contains(raw, "@") {
		parts := strings.Split(raw, "@")
		return "redis://***@" + parts[len(parts)-1]
	}
	return raw
}

func (r *RedisKVStore) reconnect() error {
	if r.conn != nil {
		_ = r.conn.Close()
		r.conn = nil
	}

	d := net.Dialer{Timeout: 1 * time.Second}
	conn, err := d.Dial("tcp", r.addr)
	if err != nil {
		return err
	}

	r.conn = conn
	r.reader = bufio.NewReader(conn)

	// Autenticación opcional
	if r.password != "" {
		if _, err := r.execCommand("AUTH", r.password); err != nil {
			_ = r.conn.Close()
			return fmt.Errorf("autenticación redis falló: %w", err)
		}
	}

	// Selección de DB opcional
	if r.db > 0 {
		if _, err := r.execCommand("SELECT", strconv.Itoa(r.db)); err != nil {
			_ = r.conn.Close()
			return fmt.Errorf("selección de db redis falló: %w", err)
		}
	}

	return nil
}

func (r *RedisKVStore) execCommand(args ...string) (any, error) {
	if r.conn == nil {
		if err := r.reconnect(); err != nil {
			return nil, err
		}
	}

	// Serialización RESP: *<count>\r\n$<len>\r\n<arg>\r\n...
	var buf strings.Builder
	fmt.Fprintf(&buf, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(&buf, "$%d\r\n%s\r\n", len(a), a)
	}

	_ = r.conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := r.conn.Write([]byte(buf.String())); err != nil {
		_ = r.reconnect()
		return nil, err
	}

	return r.parseRESP()
}

func (r *RedisKVStore) parseRESP() (any, error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return nil, errors.New("respuesta vacía de redis")
	}

	switch line[0] {
	case '+': // Simple string
		return line[1:], nil
	case '-': // Error
		return nil, errors.New(line[1:])
	case ':': // Integer
		return strconv.ParseInt(line[1:], 10, 64)
	case '$': // Bulk string
		length, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		if length == -1 {
			return nil, nil // Nil bulk string (key not found)
		}
		data := make([]byte, length+2) // + \r\n
		_, err = io.ReadFull(r.reader, data)
		if err != nil {
			return nil, err
		}
		return string(data[:length]), nil
	default:
		return nil, fmt.Errorf("tipo de respuesta RESP desconocido: %c", line[0])
	}
}

func (r *RedisKVStore) Get(_ context.Context, key string) (string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	resp, err := r.execCommand("GET", key)
	if err != nil {
		return "", false, err
	}
	if resp == nil {
		return "", false, nil
	}
	if s, ok := resp.(string); ok {
		return s, true, nil
	}
	return fmt.Sprintf("%v", resp), true, nil
}

func (r *RedisKVStore) Set(_ context.Context, key string, val string, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var args []string
	if ttl > 0 {
		seconds := int(ttl.Seconds())
		if seconds < 1 {
			seconds = 1
		}
		args = []string{"SET", key, val, "EX", strconv.Itoa(seconds)}
	} else {
		args = []string{"SET", key, val}
	}

	_, err := r.execCommand(args...)
	return err
}

func (r *RedisKVStore) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.execCommand("DEL", key)
	return err
}

func (r *RedisKVStore) Flush(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.execCommand("FLUSHDB")
	return err
}

func (r *RedisKVStore) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn != nil {
		err := r.conn.Close()
		r.conn = nil
		return err
	}
	return nil
}
