package application

import (
	"strings"
	"sync"
	"time"

	"github.com/Maxito7/hotel_backend/internal/tavily"
)

// WebCacheEntry representa una entrada en el caché
type WebCacheEntry struct {
	Response  *tavily.SearchResponse
	Timestamp time.Time
}

// WebCache implementa un caché simple en memoria para búsquedas web
type WebCache struct {
	cache map[string]*WebCacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewWebCache crea un nuevo caché de búsquedas web
func NewWebCache(ttl time.Duration) *WebCache {
	cache := &WebCache{
		cache: make(map[string]*WebCacheEntry),
		ttl:   ttl,
	}

	// Iniciar limpieza periódica
	go cache.cleanupLoop()

	return cache
}

// Get obtiene una respuesta del caché si existe y no ha expirado
func (wc *WebCache) Get(query string) (*tavily.SearchResponse, bool) {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	key := wc.normalizeQuery(query)
	entry, exists := wc.cache[key]

	if !exists {
		return nil, false
	}

	// Verificar si ha expirado
	if time.Since(entry.Timestamp) > wc.ttl {
		return nil, false
	}

	return entry.Response, true
}

// Set guarda una respuesta en el caché
func (wc *WebCache) Set(query string, response *tavily.SearchResponse) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	key := wc.normalizeQuery(query)
	wc.cache[key] = &WebCacheEntry{
		Response:  response,
		Timestamp: time.Now(),
	}
}

// normalizeQuery normaliza la query para usar como clave
func (wc *WebCache) normalizeQuery(query string) string {
	// Convertir a minúsculas y eliminar espacios extras
	normalized := strings.ToLower(strings.TrimSpace(query))
	// Reemplazar múltiples espacios por uno solo
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

// cleanupLoop limpia entradas expiradas periódicamente
func (wc *WebCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		wc.cleanup()
	}
}

// cleanup elimina entradas expiradas
func (wc *WebCache) cleanup() {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	now := time.Now()
	for key, entry := range wc.cache {
		if now.Sub(entry.Timestamp) > wc.ttl {
			delete(wc.cache, key)
		}
	}
}

// Clear limpia todo el caché
func (wc *WebCache) Clear() {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	wc.cache = make(map[string]*WebCacheEntry)
}

// Size retorna el número de entradas en el caché
func (wc *WebCache) Size() int {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	return len(wc.cache)
}
