package scheduler

import (
	"testing"
	"time"
)

func TestNextDaily(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	before := time.Date(2026, 10, 6, 5, 30, 0, 0, wib)
	if got := nextDaily(before, 6, wib); !got.Equal(time.Date(2026, 10, 6, 6, 0, 0, 0, wib)) {
		t.Fatalf("before: %v", got)
	}
	exactly := time.Date(2026, 10, 6, 6, 0, 0, 0, wib)
	if got := nextDaily(exactly, 6, wib); !got.Equal(time.Date(2026, 10, 7, 6, 0, 0, 0, wib)) {
		t.Fatalf("exactly: %v", got)
	}
	utc := time.Date(2026, 10, 5, 22, 30, 0, 0, time.UTC)
	if got := nextDaily(utc, 6, wib); !got.Equal(time.Date(2026, 10, 6, 6, 0, 0, 0, wib)) {
		t.Fatalf("utc input: %v", got)
	}
}
