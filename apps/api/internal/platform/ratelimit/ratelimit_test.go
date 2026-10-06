package ratelimit

import (
	"testing"
	"time"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
)

func TestAllowSlidingWindow(t *testing.T) {
	clk := clock.NewFake(time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC))
	l := New(clk)
	rule := Rule{Limit: 3, Window: time.Minute}

	for i := range 3 {
		if ok, _ := l.Allow("ip", rule); !ok {
			t.Fatalf("hit %d should pass", i)
		}
	}
	ok, retry := l.Allow("ip", rule)
	if ok {
		t.Fatal("4th hit should be blocked")
	}
	if retry != time.Minute {
		t.Fatalf("retry after %v", retry)
	}
	if ok, _ := l.Allow("other", rule); !ok {
		t.Fatal("other key should be independent")
	}

	clk.Advance(30 * time.Second)
	if ok, retry := l.Allow("ip", rule); ok || retry != 30*time.Second {
		t.Fatalf("expected still blocked with 30s retry, got ok=%v retry=%v", ok, retry)
	}

	clk.Advance(31 * time.Second)
	if ok, _ := l.Allow("ip", rule); !ok {
		t.Fatal("window should have slid")
	}
}

func TestSweepDropsStaleKeys(t *testing.T) {
	clk := clock.NewFake(time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC))
	l := New(clk)
	rule := Rule{Limit: 1, Window: time.Second}
	l.Allow("stale", rule)
	clk.Advance(time.Hour)
	for range sweepEvery {
		l.Allow("fresh", rule)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.hits["stale"]; exists {
		t.Fatal("stale key should have been swept")
	}
}
