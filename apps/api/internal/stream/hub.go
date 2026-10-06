package stream

import (
	"sync"

	"github.com/google/uuid"
)

const (
	clientBuffer   = 16
	MaxPerUser     = 3
	EventOrderNew  = "order.created"
	EventOrderUpd  = "order.updated"
	EventReminders = "reminder.queued"
)

type Event struct {
	Name string
	Data any
}

type Subscriber struct {
	UserID  uuid.UUID
	DepotID uuid.UUID
	Role    string
	Events  chan Event
}

type Hub struct {
	mu      sync.Mutex
	byDepot map[uuid.UUID]map[*Subscriber]struct{}
	byUser  map[uuid.UUID]int
}

func NewHub() *Hub {
	return &Hub{byDepot: make(map[uuid.UUID]map[*Subscriber]struct{}), byUser: make(map[uuid.UUID]int)}
}

func (h *Hub) Subscribe(userID, depotID uuid.UUID, role string) (*Subscriber, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.byUser[userID] >= MaxPerUser {
		return nil, false
	}
	s := &Subscriber{UserID: userID, DepotID: depotID, Role: role, Events: make(chan Event, clientBuffer)}
	if h.byDepot[depotID] == nil {
		h.byDepot[depotID] = make(map[*Subscriber]struct{})
	}
	h.byDepot[depotID][s] = struct{}{}
	h.byUser[userID]++
	return s, true
}

func (h *Hub) Unsubscribe(s *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.remove(s)
}

func (h *Hub) remove(s *Subscriber) {
	subs := h.byDepot[s.DepotID]
	if _, ok := subs[s]; !ok {
		return
	}
	delete(subs, s)
	if len(subs) == 0 {
		delete(h.byDepot, s.DepotID)
	}
	h.byUser[s.UserID]--
	if h.byUser[s.UserID] <= 0 {
		delete(h.byUser, s.UserID)
	}
	close(s.Events)
}

func (h *Hub) Publish(depotID uuid.UUID, e Event, accept func(s *Subscriber) bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.byDepot[depotID] {
		if accept != nil && !accept(s) {
			continue
		}
		select {
		case s.Events <- e:
		default:
			h.remove(s)
		}
	}
}

func (h *Hub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, subs := range h.byDepot {
		for s := range subs {
			h.remove(s)
		}
	}
}

func (h *Hub) Count(depotID uuid.UUID) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.byDepot[depotID])
}
