package cache

import (
	"sync"
	"time"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

type TTL[V any] struct {
	mu    sync.Mutex
	clock clock.Clock
	ttl   time.Duration
	items map[string]entry[V]
}

func New[V any](clk clock.Clock, ttl time.Duration) *TTL[V] {
	return &TTL[V]{clock: clk, ttl: ttl, items: make(map[string]entry[V])}
}

func (c *TTL[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || !c.clock.Now().Before(e.expiresAt) {
		delete(c.items, key)
		var zero V
		return zero, false
	}
	return e.value, true
}

func (c *TTL[V]) Set(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = entry[V]{value: value, expiresAt: c.clock.Now().Add(c.ttl)}
}

func (c *TTL[V]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}
