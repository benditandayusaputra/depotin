package stream

import (
	"testing"

	"github.com/google/uuid"
)

func TestHubFanOutAndLimits(t *testing.T) {
	h := NewHub()
	depot := uuid.New()
	owner := uuid.New()
	courier := uuid.New()

	o, ok := h.Subscribe(owner, depot, "owner")
	if !ok {
		t.Fatal("owner subscribe")
	}
	c, ok := h.Subscribe(courier, depot, "courier")
	if !ok {
		t.Fatal("courier subscribe")
	}
	h.Publish(depot, Event{Name: EventOrderNew}, func(s *Subscriber) bool { return s.Role == "owner" })
	select {
	case e := <-o.Events:
		if e.Name != EventOrderNew {
			t.Fatalf("event %s", e.Name)
		}
	default:
		t.Fatal("owner should receive")
	}
	select {
	case <-c.Events:
		t.Fatal("courier must not receive owner-only event")
	default:
	}
	h.Publish(uuid.New(), Event{Name: EventOrderNew}, nil)
	select {
	case <-o.Events:
		t.Fatal("other depot event leaked")
	default:
	}

	for i := 1; i < MaxPerUser; i++ {
		if _, ok := h.Subscribe(owner, depot, "owner"); !ok {
			t.Fatalf("subscription %d should pass", i+1)
		}
	}
	if _, ok := h.Subscribe(owner, depot, "owner"); ok {
		t.Fatal("4th subscription must be refused")
	}
	h.Unsubscribe(o)
	if _, ok := h.Subscribe(owner, depot, "owner"); !ok {
		t.Fatal("slot should free after unsubscribe")
	}
	if _, open := <-o.Events; open {
		t.Fatal("channel should be closed")
	}
}

func TestSlowClientIsDropped(t *testing.T) {
	h := NewHub()
	depot, user := uuid.New(), uuid.New()
	s, _ := h.Subscribe(user, depot, "owner")
	for range clientBuffer + 1 {
		h.Publish(depot, Event{Name: EventOrderUpd}, nil)
	}
	if h.Count(depot) != 0 {
		t.Fatal("slow client should be removed")
	}
	received := 0
	for range s.Events {
		received++
	}
	if received != clientBuffer {
		t.Fatalf("buffered %d", received)
	}
}
