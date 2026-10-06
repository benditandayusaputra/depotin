package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/crypto"
)

type customerView struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Phone       string  `json:"phone"`
	UsualQty    int32   `json:"usual_qty"`
	IsVerified  bool    `json:"is_verified"`
	HasLink     bool    `json:"has_link"`
	LoanBalance int32   `json:"loan_balance"`
	Snoozed     *string `json:"reminder_snoozed_until"`
}

func (c *client) createCustomer(name, phone string) customerView {
	c.h.t.Helper()
	res := c.mustStatus(c.do(http.MethodPost, "/api/v1/customers", map[string]any{
		"name": name, "phone": phone, "address": "Jl. Mawar 3", "area": "RT 03", "usual_qty": 2,
	}), http.StatusCreated)
	var v customerView
	res.data(c.h.t, &v)
	return v
}

func (c *client) listCustomers(query string) []customerView {
	c.h.t.Helper()
	res := c.mustStatus(c.do(http.MethodGet, "/api/v1/customers"+query, nil), http.StatusOK)
	var out []customerView
	res.data(c.h.t, &out)
	return out
}

func TestCustomersCreateSearchAndIsolation(t *testing.T) {
	h := newHarness(t)
	a := h.client()
	a.register("Depot Pelanggan A", "081211130001")
	b := h.client()
	b.register("Depot Pelanggan B", "081211130002")

	rina := a.createCustomer("Bu Rina", "0812-3333-0001")
	if rina.Phone != "6281233330001" || !rina.IsVerified || rina.UsualQty != 2 || rina.HasLink {
		t.Fatalf("unexpected customer %+v", rina)
	}
	a.createCustomer("Pak Budi", "081233330002")
	b.createCustomer("Bu Rina Depot B", "081233330001")

	a.mustStatus(a.do(http.MethodPost, "/api/v1/customers", map[string]any{"name": "Ganda", "phone": "081233330001"}), http.StatusUnprocessableEntity)

	if got := a.listCustomers("?q=rin"); len(got) != 1 || got[0].ID != rina.ID {
		t.Fatalf("search by name: %+v", got)
	}
	if got := a.listCustomers("?q=33330002"); len(got) != 1 || got[0].Name != "Pak Budi" {
		t.Fatalf("search by phone: %+v", got)
	}
	if got := a.listCustomers("?q=mawar"); len(got) != 2 {
		t.Fatalf("search by address: %+v", got)
	}
	if got := a.listCustomers(""); len(got) != 2 {
		t.Fatalf("list all: %+v", got)
	}
	if got := b.listCustomers(""); len(got) != 1 {
		t.Fatalf("depot B list: %+v", got)
	}

	b.mustStatus(b.do(http.MethodGet, "/api/v1/customers/"+rina.ID, nil), http.StatusNotFound)
	b.mustStatus(b.do(http.MethodPatch, "/api/v1/customers/"+rina.ID, map[string]any{"name": "Dibajak"}), http.StatusNotFound)
	b.mustStatus(b.do(http.MethodPost, "/api/v1/customers/"+rina.ID+"/link", nil), http.StatusNotFound)
	b.mustStatus(b.do(http.MethodPost, "/api/v1/customers/"+rina.ID+"/snooze", map[string]any{"days": 3}), http.StatusNotFound)

	upd := a.mustStatus(a.do(http.MethodPatch, "/api/v1/customers/"+rina.ID, map[string]any{"usual_qty": 3, "phone": "+62 812 3333 0009"}), http.StatusOK)
	var updated customerView
	upd.data(t, &updated)
	if updated.UsualQty != 3 || updated.Phone != "6281233330009" {
		t.Fatalf("update failed: %+v", updated)
	}
}

func TestCustomersPagination(t *testing.T) {
	h := newHarness(t)
	a := h.client()
	a.register("Depot Halaman", "081211130010")
	for i := range 7 {
		a.createCustomer("Pelanggan", "0812444400"+string(rune('0'+i)))
	}
	res := a.mustStatus(a.do(http.MethodGet, "/api/v1/customers?limit=5", nil), http.StatusOK)
	var first []customerView
	res.data(t, &first)
	var meta struct {
		NextCursor *string `json:"next_cursor"`
	}
	if err := json.Unmarshal(res.Body["meta"], &meta); err != nil || meta.NextCursor == nil {
		t.Fatalf("cursor missing: %v %s", err, res.Body["meta"])
	}
	if len(first) != 5 {
		t.Fatalf("first page %d", len(first))
	}
	second := a.listCustomers("?limit=5&cursor=" + url.QueryEscape(*meta.NextCursor))
	if len(second) != 2 {
		t.Fatalf("second page %d", len(second))
	}
	seen := map[string]bool{}
	for _, c := range append(first, second...) {
		if seen[c.ID] {
			t.Fatal("duplicate across pages")
		}
		seen[c.ID] = true
	}
	a.mustStatus(a.do(http.MethodGet, "/api/v1/customers?cursor=rusak", nil), http.StatusUnprocessableEntity)
}

func TestCustomerLinkAndRotation(t *testing.T) {
	h := newHarness(t)
	a := h.client()
	a.register("Depot Link", "081211130020")
	rina := a.createCustomer("Bu Rina", "081233330001")

	res := a.mustStatus(a.do(http.MethodPost, "/api/v1/customers/"+rina.ID+"/link", nil), http.StatusOK)
	var link struct {
		Link  string `json:"link"`
		WaURL string `json:"wa_url"`
	}
	res.data(t, &link)
	if !strings.HasPrefix(link.Link, testWebOrigin+"/p/") {
		t.Fatalf("link %q", link.Link)
	}
	if !strings.HasPrefix(link.WaURL, "https://wa.me/6281233330001?text=") || !strings.Contains(link.WaURL, url.QueryEscape(link.Link)) {
		t.Fatalf("wa url %q", link.WaURL)
	}
	again := a.mustStatus(a.do(http.MethodPost, "/api/v1/customers/"+rina.ID+"/link", nil), http.StatusOK)
	var same struct {
		Link string `json:"link"`
	}
	again.data(t, &same)
	if same.Link != link.Link {
		t.Fatal("link must be stable without rotation")
	}

	rotated := a.mustStatus(a.do(http.MethodPost, "/api/v1/customers/"+rina.ID+"/link/rotate", nil), http.StatusOK)
	var fresh struct {
		Link string `json:"link"`
	}
	rotated.data(t, &fresh)
	if fresh.Link == link.Link {
		t.Fatal("rotation must change link")
	}
	oldToken := strings.TrimPrefix(link.Link, testWebOrigin+"/p/")
	newToken := strings.TrimPrefix(fresh.Link, testWebOrigin+"/p/")
	var oldCount, newCount int
	if err := h.deps.pool.QueryRow(context.Background(), "SELECT count(*) FROM customers WHERE token_hash = $1", crypto.HashToken(oldToken)).Scan(&oldCount); err != nil {
		t.Fatal(err)
	}
	if err := h.deps.pool.QueryRow(context.Background(), "SELECT count(*) FROM customers WHERE token_hash = $1", crypto.HashToken(newToken)).Scan(&newCount); err != nil {
		t.Fatal(err)
	}
	if oldCount != 0 || newCount != 1 {
		t.Fatalf("old token still valid (%d) or new missing (%d)", oldCount, newCount)
	}
	var auditCount int
	if err := h.deps.pool.QueryRow(context.Background(), "SELECT count(*) FROM audit_logs WHERE action = 'link_rotated' AND entity_id = $1", rina.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("audit count %d", auditCount)
	}
	detail := a.mustStatus(a.do(http.MethodGet, "/api/v1/customers/"+rina.ID, nil), http.StatusOK)
	var view customerView
	detail.data(t, &view)
	if !view.HasLink {
		t.Fatal("has_link should be true")
	}
}

func TestCustomerSnooze(t *testing.T) {
	h := newHarness(t)
	a := h.client()
	a.register("Depot Tunda", "081211130030")
	rina := a.createCustomer("Bu Rina", "081233330001")
	a.mustStatus(a.do(http.MethodPost, "/api/v1/customers/"+rina.ID+"/snooze", map[string]any{"days": 0}), http.StatusUnprocessableEntity)
	a.mustStatus(a.do(http.MethodPost, "/api/v1/customers/"+rina.ID+"/snooze", map[string]any{"days": 3}), http.StatusOK)
	detail := a.mustStatus(a.do(http.MethodGet, "/api/v1/customers/"+rina.ID, nil), http.StatusOK)
	var view customerView
	detail.data(t, &view)
	if view.Snoozed == nil || *view.Snoozed != "2026-10-09" {
		t.Fatalf("snoozed until %v", view.Snoozed)
	}
}

func TestCustomerFiltersDueAtRiskLoan(t *testing.T) {
	f := setupOrders(t, "3200")
	f.owner.mustStatus(f.owner.do(http.MethodPatch, "/api/v1/depot", map[string]any{"default_days_per_gallon": 0.5}), http.StatusOK)
	v := f.dispatchedOrder(2)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+v.ID+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}), http.StatusOK)

	if due := f.owner.listCustomers("?filter=due"); len(due) != 1 || due[0].ID != f.customer.ID {
		t.Fatalf("due %+v", due)
	}
	if loan := f.owner.listCustomers("?filter=loan"); len(loan) != 1 || loan[0].LoanBalance != 1 {
		t.Fatalf("loan %+v", loan)
	}
	if risk := f.owner.listCustomers("?filter=at_risk"); len(risk) != 0 {
		t.Fatalf("not yet at risk %+v", risk)
	}
	f.h.clock.Advance(3 * 24 * time.Hour)
	f.relogin("3200")
	if risk := f.owner.listCustomers("?filter=at_risk"); len(risk) != 1 || risk[0].ID != f.customer.ID {
		t.Fatalf("at risk %+v", risk)
	}
	f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/customers?filter=bogus", nil), http.StatusUnprocessableEntity)
}
