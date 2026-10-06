package ratelimit

import (
	"sync"
	"time"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
)

const sweepEvery = 1000

type Limiter struct {
	mu    sync.Mutex
	clock clock.Clock
	hits  map[string][]time.Time
	calls int
}

func New(clk clock.Clock) *Limiter {
	return &Limiter{clock: clk, hits: make(map[string][]time.Time)}
}

type Rule struct {
	Limit  int
	Window time.Duration
}

func (l *Limiter) Allow(key string, rule Rule) (bool, time.Duration) {
	now := l.clock.Now()
	cutoff := now.Add(-rule.Window)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.calls++
	if l.calls%sweepEvery == 0 {
		l.sweep(cutoff)
	}

	recent := pruneBefore(l.hits[key], cutoff)
	if len(recent) >= rule.Limit {
		l.hits[key] = recent
		return false, recent[0].Add(rule.Window).Sub(now)
	}
	l.hits[key] = append(recent, now)
	return true, 0
}

func (l *Limiter) sweep(cutoff time.Time) {
	for key, times := range l.hits {
		kept := pruneBefore(times, cutoff)
		if len(kept) == 0 {
			delete(l.hits, key)
			continue
		}
		l.hits[key] = kept
	}
}

func pruneBefore(times []time.Time, cutoff time.Time) []time.Time {
	idx := 0
	for idx < len(times) && !times[idx].After(cutoff) {
		idx++
	}
	return times[idx:]
}
