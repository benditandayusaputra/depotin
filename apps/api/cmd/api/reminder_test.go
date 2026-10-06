package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
)

type reminderView struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	DueDate  string `json:"due_date"`
	Customer struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"customer"`
}

func (c *client) reminders() []reminderView {
	c.h.t.Helper()
	res := c.mustStatus(c.do(http.MethodGet, "/api/v1/reminders", nil), http.StatusOK)
	var out []reminderView
	res.data(c.h.t, &out)
	return out
}

func TestReminderQueueSendSkipAndConversion(t *testing.T) {
	f := setupOrders(t, "7100")
	f.owner.mustStatus(f.owner.do(http.MethodPatch, "/api/v1/depot", map[string]any{"default_days_per_gallon": 0.5}), http.StatusOK)
	first := f.dispatchedOrder(1)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+first.ID+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}), http.StatusOK)

	busy := f.owner.createCustomer("Pak Sibuk", "081271000020")
	busyOrder := f.dispatchedOrder(1)
	_ = busyOrder
	f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/orders", map[string]any{"customer_id": busy.ID}), http.StatusCreated)

	queue := f.owner.reminders()
	if len(queue) != 0 {
		t.Fatalf("customer with active order must not be queued: %+v", queue)
	}
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+busyOrder.ID+"/deliver", map[string]any{"gallons_returned": 0, "payment_method": "cash", "paid": true}), http.StatusOK)

	queue = f.owner.reminders()
	if len(queue) != 1 || queue[0].Status != "queued" || queue[0].Customer.ID != f.customer.ID || queue[0].DueDate != "2026-10-06" {
		t.Fatalf("queue %+v", queue)
	}
	if again := f.owner.reminders(); len(again) != 1 || again[0].ID != queue[0].ID {
		t.Fatalf("refresh must be idempotent: %+v", again)
	}

	other := f.h.client()
	other.register("Depot Lain", "081271000099")
	if got := other.reminders(); len(got) != 0 {
		t.Fatal("other depot sees reminders")
	}
	other.mustStatus(other.do(http.MethodPost, "/api/v1/reminders/"+queue[0].ID+"/send", nil), http.StatusNotFound)

	res := f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/reminders/"+queue[0].ID+"/send", nil), http.StatusOK)
	var sent struct {
		Reminder reminderView `json:"reminder"`
		Link     string       `json:"link"`
		WaURL    string       `json:"wa_url"`
	}
	res.data(t, &sent)
	if sent.Reminder.Status != "sent" || !strings.Contains(sent.Link, "/p/") || !strings.Contains(sent.Link, "?r="+queue[0].ID) {
		t.Fatalf("sent %+v", sent)
	}
	if !strings.HasPrefix(sent.WaURL, "https://wa.me/6281271000003?text=Halo+Bu+Rina") {
		t.Fatalf("wa url %s", sent.WaURL)
	}
	f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/reminders/"+queue[0].ID+"/send", nil), http.StatusOK)
	if listed := f.owner.reminders(); len(listed) != 1 || listed[0].Status != "sent" {
		t.Fatalf("sent reminder should stay listed: %+v", listed)
	}

	token := strings.TrimSuffix(strings.TrimPrefix(sent.Link, testWebOrigin+"/p/"), "?r="+queue[0].ID)
	anon := f.h.client()
	anon.mustStatus(anon.do(http.MethodPost, "/api/v1/public/me/"+token+"/orders", map[string]any{"r": queue[0].ID}), http.StatusCreated)
	orders := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/orders", nil), http.StatusOK)
	var list []orderView
	orders.data(t, &list)
	if list[0].Source != order.SourceReminder {
		t.Fatalf("source %s", list[0].Source)
	}
	if listed := f.owner.reminders(); len(listed) != 0 {
		t.Fatalf("ordered reminder should leave the queue: %+v", listed)
	}
	f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/reminders/"+queue[0].ID+"/skip", nil), http.StatusConflict)
}

func TestReminderSnoozeAndSkip(t *testing.T) {
	f := setupOrders(t, "7110")
	f.owner.mustStatus(f.owner.do(http.MethodPatch, "/api/v1/depot", map[string]any{"default_days_per_gallon": 0.5}), http.StatusOK)
	v := f.dispatchedOrder(1)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+v.ID+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}), http.StatusOK)
	f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/customers/"+f.customer.ID+"/snooze", map[string]any{"days": 2}), http.StatusOK)
	if q := f.owner.reminders(); len(q) != 0 {
		t.Fatalf("snoozed customer queued: %+v", q)
	}
	f.h.clock.Advance(49 * time.Hour)
	f.relogin("7110")
	q := f.owner.reminders()
	if len(q) != 1 {
		t.Fatalf("after snooze expiry expected 1, got %+v", q)
	}
	skipped := f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/reminders/"+q[0].ID+"/skip", nil), http.StatusOK)
	var rv reminderView
	skipped.data(t, &rv)
	if rv.Status != "skipped" {
		t.Fatalf("status %s", rv.Status)
	}
	if again := f.owner.reminders(); len(again) != 0 {
		t.Fatalf("skipped reminder must not reappear same day: %+v", again)
	}
	f.h.clock.Advance(24 * time.Hour)
	f.relogin("7110")
	next := f.owner.reminders()
	if len(next) != 1 || next[0].ID == q[0].ID || next[0].Status != "queued" {
		t.Fatalf("skip only suppresses the same day: %+v", next)
	}
	f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/reminders/"+next[0].ID+"/send", nil), http.StatusOK)
	f.h.clock.Advance(24 * time.Hour)
	f.relogin("7110")
	if later := f.owner.reminders(); len(later) != 1 || later[0].ID != next[0].ID {
		t.Fatalf("a sent reminder blocks new ones for 3 days: %+v", later)
	}
}
