package cache

import (
	"testing"
	"time"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
)

func TestTTLCache(t *testing.T) {
	clk := clock.NewFake(time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC))
	c := New[string](clk, 30*time.Second)
	if _, ok := c.Get("a"); ok {
		t.Fatal("empty cache should miss")
	}
	c.Set("a", "satu")
	if v, ok := c.Get("a"); !ok || v != "satu" {
		t.Fatalf("got %q %v", v, ok)
	}
	clk.Advance(29 * time.Second)
	if _, ok := c.Get("a"); !ok {
		t.Fatal("should still be cached")
	}
	clk.Advance(2 * time.Second)
	if _, ok := c.Get("a"); ok {
		t.Fatal("should have expired")
	}
	c.Set("b", "dua")
	c.Delete("b")
	if _, ok := c.Get("b"); ok {
		t.Fatal("delete failed")
	}
}
