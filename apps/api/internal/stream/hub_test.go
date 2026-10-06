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

	o := h.Subscribe(owner, depot, "owner")
	c := h.Subscribe(courier, depot, "courier")
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
		h.Subscribe(owner, depot, "owner")
	}
	if h.Count(depot) != MaxPerUser+1 {
		t.Fatalf("expected %d subscribers, got %d", MaxPerUser+1, h.Count(depot))
	}
	h.Subscribe(owner, depot, "owner")
	if h.Count(depot) != MaxPerUser+1 {
		t.Fatalf("newest connection must replace the oldest, got %d", h.Count(depot))
	}
	if _, open := <-o.Events; open {
		t.Fatal("oldest owner channel should be closed")
	}
	select {
	case _, open := <-c.Events:
		if !open {
			t.Fatal("courier channel must stay open")
		}
	default:
	}
}

func TestSlowClientIsDropped(t *testing.T) {
	h := NewHub()
	depot, user := uuid.New(), uuid.New()
	s := h.Subscribe(user, depot, "owner")
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
