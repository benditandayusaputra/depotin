package stream

import (
	"sync"
	"time"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/crypto"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
)

const (
	TicketTTL   = 30 * time.Second
	ticketBytes = 32
)

type ticket struct {
	principal httpx.Principal
	expiresAt time.Time
}

type TicketStore struct {
	mu      sync.Mutex
	clock   clock.Clock
	tickets map[string]ticket
}

func NewTicketStore(clk clock.Clock) *TicketStore {
	return &TicketStore{clock: clk, tickets: make(map[string]ticket)}
}

func (s *TicketStore) Issue(p httpx.Principal) (string, error) {
	raw, err := idgen.RandomToken(ticketBytes)
	if err != nil {
		return "", err
	}
	now := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, t := range s.tickets {
		if !now.Before(t.expiresAt) {
			delete(s.tickets, key)
		}
	}
	s.tickets[string(crypto.HashToken(raw))] = ticket{principal: p, expiresAt: now.Add(TicketTTL)}
	return raw, nil
}

func (s *TicketStore) Redeem(raw string) (httpx.Principal, bool) {
	key := string(crypto.HashToken(raw))
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tickets[key]
	if !ok {
		return httpx.Principal{}, false
	}
	delete(s.tickets, key)
	if !s.clock.Now().Before(t.expiresAt) {
		return httpx.Principal{}, false
	}
	return t.principal, true
}
