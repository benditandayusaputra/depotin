package stream

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

func TestTicketSingleUseAndExpiry(t *testing.T) {
	clk := clock.NewFake(time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC))
	store := NewTicketStore(clk)
	p := httpx.Principal{UserID: uuid.New(), DepotID: uuid.New(), Role: "owner"}
	raw, err := store.Issue(p)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := store.Redeem(raw)
	if !ok || got.UserID != p.UserID {
		t.Fatal("first redeem should work")
	}
	if _, ok := store.Redeem(raw); ok {
		t.Fatal("ticket must be single use")
	}
	raw2, _ := store.Issue(p)
	clk.Advance(TicketTTL + time.Second)
	if _, ok := store.Redeem(raw2); ok {
		t.Fatal("expired ticket must fail")
	}
	if _, ok := store.Redeem("bukan-tiket"); ok {
		t.Fatal("unknown ticket must fail")
	}
}
