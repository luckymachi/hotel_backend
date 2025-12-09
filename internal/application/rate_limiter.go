package application

import (
	"fmt"
	"sync"
	"time"
)

// RateLimitEntry representa una entrada en el rate limiter
type RateLimitEntry struct {
	Count     int
	ResetTime time.Time
}

// RateLimiter implementa un rate limiter simple basado en ventanas de tiempo
type RateLimiter struct {
	limits map[string]*RateLimitEntry
	mu     sync.RWMutex
	window time.Duration
	limit  int
}

// NewRateLimiter crea un nuevo rate limiter
// window: duración de la ventana de tiempo (ej: 1 minuto)
// limit: número máximo de requests permitidos en la ventana
func NewRateLimiter(window time.Duration, limit int) *RateLimiter {
	rl := &RateLimiter{
		limits: make(map[string]*RateLimitEntry),
		window: window,
		limit:  limit,
	}

	// Iniciar limpieza periódica
	go rl.cleanupLoop()

	return rl
}

// Allow verifica si se permite una request para el identificador dado
// identifier puede ser: IP, clientID, conversationID, etc.
func (rl *RateLimiter) Allow(identifier string) (bool, error) {
	if identifier == "" {
		identifier = "anonymous"
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.limits[identifier]

	// Si no existe o la ventana ha expirado, crear nueva entrada
	if !exists || now.After(entry.ResetTime) {
		rl.limits[identifier] = &RateLimitEntry{
			Count:     1,
			ResetTime: now.Add(rl.window),
		}
		return true, nil
	}

	// Si aún está dentro de la ventana
	if entry.Count >= rl.limit {
		timeUntilReset := entry.ResetTime.Sub(now)
		return false, fmt.Errorf("límite de mensajes excedido. Intenta de nuevo en %v", timeUntilReset.Round(time.Second))
	}

	// Incrementar contador
	entry.Count++
	return true, nil
}

// GetRemaining obtiene el número de requests restantes para un identificador
func (rl *RateLimiter) GetRemaining(identifier string) int {
	if identifier == "" {
		identifier = "anonymous"
	}

	rl.mu.RLock()
	defer rl.mu.RUnlock()

	entry, exists := rl.limits[identifier]
	if !exists {
		return rl.limit
	}

	now := time.Now()
	if now.After(entry.ResetTime) {
		return rl.limit
	}

	remaining := rl.limit - entry.Count
	if remaining < 0 {
		return 0
	}

	return remaining
}

// Reset resetea el contador para un identificador específico
func (rl *RateLimiter) Reset(identifier string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.limits, identifier)
}

// cleanupLoop limpia entradas expiradas periódicamente
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup elimina entradas expiradas
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, entry := range rl.limits {
		if now.After(entry.ResetTime) {
			delete(rl.limits, key)
		}
	}
}

// Clear limpia todos los límites
func (rl *RateLimiter) Clear() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.limits = make(map[string]*RateLimitEntry)
}

// Size retorna el número de identificadores rastreados
func (rl *RateLimiter) Size() int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return len(rl.limits)
}
