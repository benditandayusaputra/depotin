package idgen

import (
	"strings"
	"testing"
	"time"
)

func TestNewIDIsVersion7AndMonotonic(t *testing.T) {
	a := NewID()
	b := NewID()
	if a.Version() != 7 || b.Version() != 7 {
		t.Fatalf("expected v7, got %d and %d", a.Version(), b.Version())
	}
	if a.String() >= b.String() {
		t.Fatalf("expected %s < %s", a, b)
	}
}

func TestRandomTokenIsBase62AndUnique(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		tok, err := RandomToken(16)
		if err != nil {
			t.Fatal(err)
		}
		if len(tok) < 20 || len(tok) > 22 {
			t.Fatalf("unexpected length %d for %q", len(tok), tok)
		}
		for _, r := range tok {
			if !strings.ContainsRune(base62Alphabet, r) {
				t.Fatalf("non base62 char %q in %q", r, tok)
			}
		}
		if seen[tok] {
			t.Fatalf("duplicate token %q", tok)
		}
		seen[tok] = true
	}
}

func TestEncodeBase62KeepsLeadingZeroBytes(t *testing.T) {
	if got := encodeBase62([]byte{0, 0, 1}); got != "001" {
		t.Fatalf("got %q", got)
	}
}

func TestOrderCode(t *testing.T) {
	day := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	if got := OrderCode(day, 14); got != "DP-261020-014" {
		t.Fatalf("got %q", got)
	}
	if got := OrderCode(day, 1234); got != "DP-261020-1234" {
		t.Fatalf("got %q", got)
	}
}
