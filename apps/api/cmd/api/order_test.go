package main

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

type orderView struct {
	ID            string  `json:"id"`
	Code          string  `json:"code"`
	Status        string  `json:"status"`
	Source        string  `json:"source"`
	RefillQty     int32   `json:"refill_qty"`
	FreeQty       int32   `json:"free_qty"`
	Subtotal      int64   `json:"subtotal"`
	DeliveryFee   int64   `json:"delivery_fee"`
	Total         int64   `json:"total"`
	PaymentStatus string  `json:"payment_status"`
	CourierID     *string `json:"courier_id"`
	CourierName   *string `json:"courier_name"`
	Items         []struct {
		ProductKind string `json:"product_kind"`
		Qty         int32  `json:"qty"`
		UnitPrice   int64  `json:"unit_price"`
	} `json:"items"`
}

type orderFixture struct {
	h         *harness
	owner     *client
	courier   *client
	courierID string
	customer  customerView
}

func setupOrders(t *testing.T, phoneSeed string) orderFixture {
	t.Helper()
	h := newHarness(t)
	owner := h.client()
	owner.register("Depot Pesanan "+phoneSeed, "0812"+phoneSeed+"0001")
	owner.mustStatus(owner.do(http.MethodPatch, "/api/v1/depot", map[string]any{"delivery_fee": 2000}), http.StatusOK)
	courierID := owner.createCourier("Kurir Andi", "0812"+phoneSeed+"0002")
	courier := h.client()
	courier.mustStatus(courier.login("0812"+phoneSeed+"0002", testPassword), http.StatusOK)
	cust := owner.createCustomer("Bu Rina", "0812"+phoneSeed+"0003")
	return orderFixture{h: h, owner: owner, courier: courier, courierID: courierID, customer: cust}
}

func (f orderFixture) create(body map[string]any, opts ...func(*http.Request)) (orderView, response) {
	f.h.t.Helper()
	if body == nil {
		body = map[string]any{"customer_id": f.customer.ID}
	}
	res := f.owner.do(http.MethodPost, "/api/v1/orders", body, opts...)
	var v orderView
	if res.Status == http.StatusCreated || res.Status == http.StatusOK {
		res.data(f.h.t, &v)
	}
	return v, res
}

func withIdempotency(key string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set(order.HeaderIdempotencyKey, key) }
}

func TestOwnerCreatesOrderWithServerPricing(t *testing.T) {
	f := setupOrders(t, "4100")
	v, res := f.create(nil)
	f.owner.mustStatus(res, http.StatusCreated)
	if v.Status != order.StatusConfirmed || v.Source != order.SourceOwner || v.RefillQty != 2 {
		t.Fatalf("unexpected order %+v", v)
	}
	if !strings.HasPrefix(v.Code, "DP-261006-001") {
		t.Fatalf("code %q", v.Code)
	}
	if v.Subtotal != 12000 || v.DeliveryFee != 2000 || v.Total != 14000 || v.PaymentStatus != "unpaid" {
		t.Fatalf("pricing %+v", v)
	}
	second, _ := f.create(map[string]any{"customer_id": f.customer.ID, "fulfilment": "pickup", "refill_qty": 1})
	if second.Code != "DP-261006-002" || second.Total != 6000 {
		t.Fatalf("second order %+v", second)
	}

	_, tampered := f.create(map[string]any{"customer_id": f.customer.ID, "total": 1})
	f.owner.mustStatus(tampered, http.StatusUnprocessableEntity)
	_, badQty := f.create(map[string]any{"customer_id": f.customer.ID, "refill_qty": 51})
	f.owner.mustStatus(badQty, http.StatusUnprocessableEntity)
	_, badDate := f.create(map[string]any{"customer_id": f.customer.ID, "scheduled_date": "2026-01-01"})
	f.owner.mustStatus(badDate, http.StatusUnprocessableEntity)
	_, tomorrow := f.create(map[string]any{"customer_id": f.customer.ID, "scheduled_date": "2026-10-07"})
	f.owner.mustStatus(tomorrow, http.StatusCreated)
}

func TestOrderKeepsPriceAfterProductChange(t *testing.T) {
	f := setupOrders(t, "4110")
	products := f.owner.listProducts()
	v, _ := f.create(map[string]any{"customer_id": f.customer.ID, "items": []map[string]any{{"product_id": products[0].ID, "qty": 3}}})
	if v.Subtotal != 18000 {
		t.Fatalf("subtotal %d", v.Subtotal)
	}
	f.owner.mustStatus(f.owner.do(http.MethodPatch, "/api/v1/products/"+products[0].ID, map[string]any{"price": 9000}), http.StatusOK)
	res := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/orders/"+v.ID, nil), http.StatusOK)
	var again orderView
	res.data(t, &again)
	if again.Subtotal != 18000 || again.Items[0].UnitPrice != 6000 {
		t.Fatalf("old order changed: %+v", again)
	}
}

func TestOrderIdempotencyKey(t *testing.T) {
	f := setupOrders(t, "4120")
	first, res1 := f.create(nil, withIdempotency("abc-123"))
	f.owner.mustStatus(res1, http.StatusCreated)
	second, res2 := f.create(nil, withIdempotency("abc-123"))
	f.owner.mustStatus(res2, http.StatusOK)
	if first.ID != second.ID {
		t.Fatal("same key must return the same order")
	}
	third, res3 := f.create(nil, withIdempotency("abc-456"))
	f.owner.mustStatus(res3, http.StatusCreated)
	if third.ID == first.ID {
		t.Fatal("different key must create a new order")
	}
	var count int
	if err := f.h.deps.pool.QueryRow(context.Background(), "SELECT count(*) FROM orders WHERE customer_id = $1", f.customer.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 orders, got %d", count)
	}
}

func TestOrderTransitionsOverHTTP(t *testing.T) {
	f := setupOrders(t, "4130")
	v, _ := f.create(nil)
	path := "/api/v1/orders/" + v.ID

	res := f.owner.do(http.MethodPost, path+"/confirm", nil)
	f.owner.mustStatus(res, http.StatusConflict)
	if res.errorCode(t) != httpx.CodeInvalidTransition {
		t.Fatalf("code %s", res.errorCode(t))
	}
	f.courier.mustStatus(f.courier.do(http.MethodGet, path, nil), http.StatusNotFound)
	f.courier.mustStatus(f.courier.do(http.MethodPost, path+"/dispatch", nil), http.StatusNotFound)

	f.owner.mustStatus(f.owner.do(http.MethodPost, path+"/assign", map[string]any{"courier_id": f.customer.ID}), http.StatusUnprocessableEntity)
	assigned := f.owner.mustStatus(f.owner.do(http.MethodPost, path+"/assign", map[string]any{"courier_id": f.courierID}), http.StatusOK)
	var av orderView
	assigned.data(t, &av)
	if av.CourierID == nil || *av.CourierID != f.courierID || av.CourierName == nil || *av.CourierName != "Kurir Andi" {
		t.Fatalf("assign failed %+v", av)
	}

	courierGet := f.courier.mustStatus(f.courier.do(http.MethodGet, path, nil), http.StatusOK)
	if strings.Contains(string(courierGet.Body["data"]), `"subtotal"`) {
		t.Fatal("courier view must not expose subtotal")
	}

	dispatched := f.courier.mustStatus(f.courier.do(http.MethodPost, path+"/dispatch", nil, withIdempotency("d-1")), http.StatusOK)
	var dv orderView
	dispatched.data(t, &dv)
	if dv.Status != order.StatusOnDelivery {
		t.Fatalf("status %s", dv.Status)
	}
	f.courier.mustStatus(f.courier.do(http.MethodPost, path+"/dispatch", nil, withIdempotency("d-1")), http.StatusOK)
	f.courier.mustStatus(f.courier.do(http.MethodPost, path+"/dispatch", nil, withIdempotency("d-2")), http.StatusConflict)
	f.courier.mustStatus(f.courier.do(http.MethodPost, path+"/dispatch", nil), http.StatusConflict)
	var events int
	if err := f.h.deps.pool.QueryRow(context.Background(), "SELECT count(*) FROM order_events WHERE order_id = $1 AND type = 'dispatched'", v.ID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 1 {
		t.Fatalf("dispatched events %d", events)
	}

	f.owner.mustStatus(f.owner.do(http.MethodPost, path+"/mark-paid", map[string]any{"payment_method": "transfer"}), http.StatusOK)
	f.owner.mustStatus(f.owner.do(http.MethodPost, path+"/mark-paid", nil), http.StatusConflict)

	f.owner.mustStatus(f.owner.do(http.MethodPost, path+"/cancel", map[string]any{"reason": "ab"}), http.StatusUnprocessableEntity)
	cancelled := f.owner.mustStatus(f.owner.do(http.MethodPost, path+"/cancel", map[string]any{"reason": "Pelanggan tidak di rumah"}), http.StatusOK)
	var cv orderView
	cancelled.data(t, &cv)
	if cv.Status != order.StatusCancelled {
		t.Fatalf("status %s", cv.Status)
	}
	f.owner.mustStatus(f.owner.do(http.MethodPost, path+"/cancel", map[string]any{"reason": "Lagi"}), http.StatusConflict)
	f.owner.mustStatus(f.owner.do(http.MethodPost, path+"/assign", map[string]any{"courier_id": f.courierID}), http.StatusConflict)

	list := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/orders?status=cancelled", nil), http.StatusOK)
	var listed []orderView
	list.data(t, &listed)
	if len(listed) != 1 || listed[0].ID != v.ID {
		t.Fatalf("list by status %+v", listed)
	}
	f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/orders?status=bogus", nil), http.StatusUnprocessableEntity)
}

func TestOrdersAreIsolatedPerDepot(t *testing.T) {
	f := setupOrders(t, "4140")
	v, _ := f.create(nil)
	other := f.h.client()
	other.register("Depot Lain", "081241400009")
	path := "/api/v1/orders/" + v.ID
	other.mustStatus(other.do(http.MethodGet, path, nil), http.StatusNotFound)
	other.mustStatus(other.do(http.MethodPost, path+"/confirm", nil), http.StatusNotFound)
	other.mustStatus(other.do(http.MethodPost, path+"/cancel", map[string]any{"reason": "Dibajak"}), http.StatusNotFound)
	other.mustStatus(other.do(http.MethodPost, "/api/v1/orders", map[string]any{"customer_id": f.customer.ID}), http.StatusUnprocessableEntity)
	listed := other.mustStatus(other.do(http.MethodGet, "/api/v1/orders", nil), http.StatusOK)
	var out []orderView
	listed.data(t, &out)
	if len(out) != 0 {
		t.Fatal("other depot sees orders")
	}
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders", map[string]any{"customer_id": f.customer.ID}), http.StatusForbidden)
	f.courier.mustStatus(f.courier.do(http.MethodGet, "/api/v1/orders", nil), http.StatusForbidden)
}
