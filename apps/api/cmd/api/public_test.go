package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
)

type createdOrder struct {
	Code       string `json:"code"`
	Status     string `json:"status"`
	TrackToken string `json:"track_token"`
}

type trackView struct {
	Code          string `json:"code"`
	Status        string `json:"status"`
	DeliveryPhone string `json:"delivery_phone"`
	DeliveryName  string `json:"delivery_name"`
	CanCancel     bool   `json:"can_cancel"`
	Depot         struct {
		Name string `json:"name"`
	} `json:"depot"`
}

func (c *client) publicOrder(slug string, body map[string]any, opts ...func(*http.Request)) (createdOrder, response) {
	c.h.t.Helper()
	res := c.do(http.MethodPost, "/api/v1/public/depots/"+slug+"/orders", body, opts...)
	var out createdOrder
	if res.Status == http.StatusCreated || res.Status == http.StatusOK {
		res.data(c.h.t, &out)
	}
	return out, res
}

func TestPublicDepotPageCacheAndETag(t *testing.T) {
	h := newHarness(t)
	owner := h.client()
	s := owner.register("Depot Publik", "081251000001")
	anon := h.client()

	res := anon.mustStatus(anon.do(http.MethodGet, "/api/v1/public/depots/"+s.Depot.Slug, nil), http.StatusOK)
	etag := res.Header.Get(fiber.HeaderETag)
	if etag == "" || !strings.Contains(res.Header.Get(fiber.HeaderCacheControl), "max-age=30") {
		t.Fatalf("headers %v", res.Header)
	}
	var page struct {
		Depot struct {
			Name string `json:"name"`
		} `json:"depot"`
		Products []productView `json:"products"`
	}
	res.data(t, &page)
	if page.Depot.Name != "Depot Publik" || len(page.Products) != 1 || page.Products[0].Price != 6000 {
		t.Fatalf("page %+v", page)
	}
	notModified := anon.do(http.MethodGet, "/api/v1/public/depots/"+s.Depot.Slug, nil, func(r *http.Request) { r.Header.Set(fiber.HeaderIfNoneMatch, etag) })
	if notModified.Status != http.StatusNotModified {
		t.Fatalf("status %d", notModified.Status)
	}

	owner.mustStatus(owner.do(http.MethodPatch, "/api/v1/products/"+page.Products[0].ID, map[string]any{"price": 7000}), http.StatusOK)
	fresh := anon.mustStatus(anon.do(http.MethodGet, "/api/v1/public/depots/"+s.Depot.Slug, nil), http.StatusOK)
	fresh.data(t, &page)
	if page.Products[0].Price != 7000 || fresh.Header.Get(fiber.HeaderETag) == etag {
		t.Fatal("cache not invalidated after price change")
	}
	anon.mustStatus(anon.do(http.MethodGet, "/api/v1/public/depots/tidak-ada", nil), http.StatusNotFound)
}

func TestPublicOrderFlow(t *testing.T) {
	h := newHarness(t)
	owner := h.client()
	s := owner.register("Depot Alur B", "081251000010")
	existing := owner.createCustomer("Bu Rina Asli", "081251000011")
	anon := h.client()

	body := map[string]any{"name": "Rina Palsu", "phone": "0812-5100-0011", "address": "Jl. Melati 5 RT 02", "qty": 2}
	created, res := anon.publicOrder(s.Depot.Slug, body, withIdempotency("pub-1"))
	anon.mustStatus(res, http.StatusCreated)
	if created.Status != order.StatusPending || created.TrackToken == "" || !strings.HasPrefix(created.Code, "DP-") {
		t.Fatalf("created %+v", created)
	}
	raw := string(res.Body["data"])
	if strings.Contains(raw, "Bu Rina Asli") || strings.Contains(raw, "Mawar") || strings.Contains(raw, existing.ID) {
		t.Fatalf("response leaks existing customer data: %s", raw)
	}
	again, res2 := anon.publicOrder(s.Depot.Slug, body, withIdempotency("pub-1"))
	anon.mustStatus(res2, http.StatusOK)
	if again.Code != created.Code {
		t.Fatal("idempotent public order must return same code")
	}

	detail := owner.mustStatus(owner.do(http.MethodGet, "/api/v1/customers/"+existing.ID, nil), http.StatusOK)
	var cust customerView
	detail.data(t, &cust)
	if cust.Name != "Bu Rina Asli" {
		t.Fatal("existing customer must not be overwritten by public form")
	}

	_, bot := anon.publicOrder(s.Depot.Slug, map[string]any{"name": "Bot", "phone": "081251000012", "address": "Jl. Robot 1", "qty": 1, "website": "http://spam"})
	anon.mustStatus(bot, http.StatusUnprocessableEntity)

	newcomer := map[string]any{"name": "Pak Baru", "phone": "081251000013", "address": "Jl. Baru 7", "qty": 1}
	repeat := h.client()
	repeat.ip = "198.51.100.70"
	for range 3 {
		_, r := repeat.publicOrder(s.Depot.Slug, newcomer)
		repeat.mustStatus(r, http.StatusCreated)
	}
	_, fourth := repeat.publicOrder(s.Depot.Slug, newcomer)
	repeat.mustStatus(fourth, http.StatusTooManyRequests)
	otherIP := h.client()
	otherIP.ip = "198.51.100.77"
	_, tooMany := otherIP.publicOrder(s.Depot.Slug, newcomer)
	otherIP.mustStatus(tooMany, http.StatusTooManyRequests)

	list := owner.mustStatus(owner.do(http.MethodGet, "/api/v1/customers?q=Baru", nil), http.StatusOK)
	var found []customerView
	list.data(t, &found)
	if len(found) != 1 || found[0].IsVerified {
		t.Fatalf("public customer should exist unverified: %+v", found)
	}

	owner.mustStatus(owner.do(http.MethodPatch, "/api/v1/depot", map[string]any{"is_accepting_orders": false}), http.StatusOK)
	_, closed := otherIP.publicOrder(s.Depot.Slug, map[string]any{"name": "X", "phone": "081251000019", "address": "Jl. Tutup 1", "qty": 1})
	otherIP.mustStatus(closed, http.StatusConflict)
}

func TestPublicTrackAndCancel(t *testing.T) {
	h := newHarness(t)
	owner := h.client()
	s := owner.register("Depot Lacak", "081251000020")
	anon := h.client()
	created, _ := anon.publicOrder(s.Depot.Slug, map[string]any{"name": "Dewi", "phone": "081251000021", "address": "Jl. Lacak 2", "qty": 1})

	res := anon.mustStatus(anon.do(http.MethodGet, "/api/v1/public/track/"+created.TrackToken, nil), http.StatusOK)
	if res.Header.Get(fiber.HeaderCacheControl) != "no-store" {
		t.Fatalf("cache-control %q", res.Header.Get(fiber.HeaderCacheControl))
	}
	var tv trackView
	res.data(t, &tv)
	if tv.Code != created.Code || tv.DeliveryPhone != "•••••••••0021" || !tv.CanCancel || tv.Depot.Name != "Depot Lacak" {
		t.Fatalf("track %+v", tv)
	}
	anon.mustStatus(anon.do(http.MethodGet, "/api/v1/public/track/bukan-token", nil), http.StatusNotFound)
	anon.mustStatus(anon.do(http.MethodGet, "/api/v1/public/track/"+strings.Repeat("a", 22), nil), http.StatusNotFound)

	cancelled := anon.mustStatus(anon.do(http.MethodPost, "/api/v1/public/track/"+created.TrackToken+"/cancel", nil), http.StatusOK)
	cancelled.data(t, &tv)
	if tv.Status != order.StatusCancelled || tv.CanCancel {
		t.Fatalf("cancel %+v", tv)
	}
	anon.mustStatus(anon.do(http.MethodPost, "/api/v1/public/track/"+created.TrackToken+"/cancel", nil), http.StatusConflict)

	second, _ := anon.publicOrder(s.Depot.Slug, map[string]any{"name": "Dewi", "phone": "081251000021", "address": "Jl. Lacak 2", "qty": 1})
	ordersRes := owner.mustStatus(owner.do(http.MethodGet, "/api/v1/orders?status=pending", nil), http.StatusOK)
	var pending []orderView
	ordersRes.data(t, &pending)
	if len(pending) != 1 || pending[0].Code != second.Code {
		t.Fatalf("pending list %+v", pending)
	}
	owner.mustStatus(owner.do(http.MethodPost, "/api/v1/orders/"+pending[0].ID+"/confirm", nil), http.StatusOK)
	anon.mustStatus(anon.do(http.MethodPost, "/api/v1/public/track/"+second.TrackToken+"/cancel", nil), http.StatusConflict)
}

func personalToken(t *testing.T, owner *client, customerID string) string {
	t.Helper()
	res := owner.mustStatus(owner.do(http.MethodPost, "/api/v1/customers/"+customerID+"/link", nil), http.StatusOK)
	var link struct {
		Link string `json:"link"`
	}
	res.data(t, &link)
	return strings.TrimPrefix(link.Link, testWebOrigin+"/p/")
}

func TestPersonalPageAndReorder(t *testing.T) {
	h := newHarness(t)
	owner := h.client()
	owner.register("Depot Pribadi", "081251000030")
	rina := owner.createCustomer("Bu Rina", "081251000031")
	token := personalToken(t, owner, rina.ID)
	anon := h.client()

	res := anon.mustStatus(anon.do(http.MethodGet, "/api/v1/public/me/"+token, nil), http.StatusOK)
	var me struct {
		Customer struct {
			Name        string `json:"name"`
			PhoneMasked string `json:"phone_masked"`
			UsualQty    int32  `json:"usual_qty"`
		} `json:"customer"`
		Depot struct {
			RefillPrice int64 `json:"refill_price"`
		} `json:"depot"`
		ActiveOrders []trackView `json:"active_orders"`
	}
	res.data(t, &me)
	if me.Customer.Name != "Bu Rina" || me.Customer.PhoneMasked != "•••••••••0031" || me.Customer.UsualQty != 2 || me.Depot.RefillPrice != 6000 {
		t.Fatalf("me %+v", me)
	}
	if strings.Contains(string(res.Body["data"]), "6281251000031") {
		t.Fatal("personal page leaks full phone")
	}

	created := anon.do(http.MethodPost, "/api/v1/public/me/"+token+"/orders", nil)
	anon.mustStatus(created, http.StatusCreated)
	var co createdOrder
	created.data(t, &co)
	if co.Status != order.StatusConfirmed {
		t.Fatalf("auto confirm expected, got %s", co.Status)
	}
	res = anon.mustStatus(anon.do(http.MethodGet, "/api/v1/public/me/"+token, nil), http.StatusOK)
	res.data(t, &me)
	if len(me.ActiveOrders) != 1 || me.ActiveOrders[0].Code != co.Code {
		t.Fatalf("active orders %+v", me.ActiveOrders)
	}
	ordersRes := owner.mustStatus(owner.do(http.MethodGet, "/api/v1/orders", nil), http.StatusOK)
	var list []orderView
	ordersRes.data(t, &list)
	if len(list) != 1 || list[0].Source != order.SourceLink || list[0].RefillQty != 2 {
		t.Fatalf("owner list %+v", list)
	}

	bogus := anon.do(http.MethodPost, "/api/v1/public/me/"+token+"/orders", map[string]any{"qty": 1, "r": uuid.NewString()})
	anon.mustStatus(bogus, http.StatusCreated)
	bogus.data(t, &co)
	ordersRes = owner.mustStatus(owner.do(http.MethodGet, "/api/v1/orders", nil), http.StatusOK)
	ordersRes.data(t, &list)
	if list[0].Source != order.SourceLink {
		t.Fatalf("invalid reminder must fall back to link, got %s", list[0].Source)
	}
	anon.mustStatus(anon.do(http.MethodPost, "/api/v1/public/me/"+token+"/orders", map[string]any{"scheduled_date": "2026-10-09"}), http.StatusUnprocessableEntity)

	owner.mustStatus(owner.do(http.MethodPost, "/api/v1/customers/"+rina.ID+"/link/rotate", nil), http.StatusOK)
	anon.mustStatus(anon.do(http.MethodGet, "/api/v1/public/me/"+token, nil), http.StatusNotFound)
}

func TestReminderAttribution(t *testing.T) {
	h := newHarness(t)
	owner := h.client()
	s := owner.register("Depot Atribusi", "081251000040")
	rina := owner.createCustomer("Bu Rina", "081251000041")
	token := personalToken(t, owner, rina.ID)
	anon := h.client()
	now := h.clock.Now()

	insert := func(sentAt time.Time, due string) string {
		id := uuid.NewString()
		_, err := h.deps.pool.Exec(context.Background(),
			"INSERT INTO reminders (id, depot_id, customer_id, due_date, predicted_empty_at, status, sent_at) VALUES ($1, $2, $3, $4, $5, 'sent', $6)",
			id, s.Depot.ID, rina.ID, due, now, sentAt)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	stale := insert(now.Add(-49*time.Hour), "2026-10-04")
	fresh := insert(now.Add(-2*time.Hour), "2026-10-06")

	res := anon.mustStatus(anon.do(http.MethodPost, "/api/v1/public/me/"+token+"/orders", map[string]any{"r": stale}), http.StatusCreated)
	var co createdOrder
	res.data(t, &co)
	ordersRes := owner.mustStatus(owner.do(http.MethodGet, "/api/v1/orders", nil), http.StatusOK)
	var list []orderView
	ordersRes.data(t, &list)
	if list[0].Source != order.SourceLink {
		t.Fatalf("stale reminder should not be attributed: %s", list[0].Source)
	}
	owner.mustStatus(owner.do(http.MethodPost, "/api/v1/orders/"+list[0].ID+"/cancel", map[string]any{"reason": "Uji coba"}), http.StatusOK)

	anon.mustStatus(anon.do(http.MethodPost, "/api/v1/public/me/"+token+"/orders", map[string]any{"r": fresh}), http.StatusCreated)
	ordersRes = owner.mustStatus(owner.do(http.MethodGet, "/api/v1/orders", nil), http.StatusOK)
	ordersRes.data(t, &list)
	if list[0].Source != order.SourceReminder {
		t.Fatalf("fresh reminder should be attributed: %s", list[0].Source)
	}
	var status string
	var orderID *string
	if err := h.deps.pool.QueryRow(context.Background(), "SELECT status, order_id::text FROM reminders WHERE id = $1", fresh).Scan(&status, &orderID); err != nil {
		t.Fatal(err)
	}
	if status != "ordered" || orderID == nil || *orderID != list[0].ID {
		t.Fatalf("reminder status %s order %v", status, orderID)
	}
	owner.mustStatus(owner.do(http.MethodPost, "/api/v1/orders/"+list[0].ID+"/cancel", map[string]any{"reason": "Batal lagi"}), http.StatusOK)
	if err := h.deps.pool.QueryRow(context.Background(), "SELECT status, order_id::text FROM reminders WHERE id = $1", fresh).Scan(&status, &orderID); err != nil {
		t.Fatal(err)
	}
	if status != "sent" || orderID != nil {
		t.Fatalf("reminder should reopen: %s %v", status, orderID)
	}
}
