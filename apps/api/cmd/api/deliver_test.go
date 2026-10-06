package main

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
)

type customerState struct {
	LoanBalance          int32    `json:"loan_balance"`
	StampCount           int32    `json:"stamp_count"`
	IsVerified           bool     `json:"is_verified"`
	PredictionConfidence string   `json:"prediction_confidence"`
	DaysPerGallon        *float64 `json:"days_per_gallon"`
	LastDeliveredQty     *int32   `json:"last_delivered_qty"`
	PredictedEmptyAt     *string  `json:"predicted_empty_at"`
}

func (f orderFixture) customerState() customerState {
	f.h.t.Helper()
	res := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/customers/"+f.customer.ID, nil), http.StatusOK)
	var s customerState
	res.data(f.h.t, &s)
	return s
}

func (f orderFixture) ledgerSum() (int64, int32) {
	f.h.t.Helper()
	var sum int64
	var balance int32
	err := f.h.deps.pool.QueryRow(context.Background(),
		"SELECT coalesce((SELECT sum(delta) FROM gallon_ledger WHERE customer_id = $1), 0), (SELECT loan_balance FROM customers WHERE id = $1)", f.customer.ID,
	).Scan(&sum, &balance)
	if err != nil {
		f.h.t.Fatal(err)
	}
	return sum, balance
}

func (f orderFixture) dispatchedOrder(refillQty int32) orderView {
	f.h.t.Helper()
	v, res := f.create(map[string]any{"customer_id": f.customer.ID, "refill_qty": refillQty})
	f.owner.mustStatus(res, http.StatusCreated)
	f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/orders/"+v.ID+"/assign", map[string]any{"courier_id": f.courierID}), http.StatusOK)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+v.ID+"/dispatch", nil), http.StatusOK)
	return v
}

func TestDeliverUpdatesLedgerLoyaltyAndPrediction(t *testing.T) {
	f := setupOrders(t, "6100")
	f.owner.mustStatus(f.owner.do(http.MethodPatch, "/api/v1/depot", map[string]any{"loyalty_every": 2}), http.StatusOK)

	v := f.dispatchedOrder(2)
	path := "/api/v1/orders/" + v.ID
	f.courier.mustStatus(f.courier.do(http.MethodPost, path+"/deliver", map[string]any{"gallons_returned": 5, "payment_method": "cash", "paid": true}), http.StatusUnprocessableEntity)
	f.courier.mustStatus(f.courier.do(http.MethodPost, path+"/deliver", map[string]any{"gallons_returned": 1, "paid": true}), http.StatusUnprocessableEntity)
	res := f.courier.mustStatus(f.courier.do(http.MethodPost, path+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}, withIdempotency("dl-1")), http.StatusOK)
	var delivered orderView
	res.data(t, &delivered)
	if delivered.Status != order.StatusDelivered || delivered.PaymentStatus != "paid" {
		t.Fatalf("delivered %+v", delivered)
	}

	state := f.customerState()
	if state.LoanBalance != 1 || state.StampCount != 2 || !state.IsVerified || state.PredictionConfidence != "low" || state.LastDeliveredQty == nil || *state.LastDeliveredQty != 2 || state.PredictedEmptyAt == nil {
		t.Fatalf("customer state %+v", state)
	}
	if sum, balance := f.ledgerSum(); sum != 1 || balance != 1 {
		t.Fatalf("ledger sum %d balance %d", sum, balance)
	}

	f.courier.mustStatus(f.courier.do(http.MethodPost, path+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}, withIdempotency("dl-1")), http.StatusOK)
	f.courier.mustStatus(f.courier.do(http.MethodPost, path+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}, withIdempotency("dl-2")), http.StatusConflict)
	if sum, _ := f.ledgerSum(); sum != 1 {
		t.Fatalf("ledger written twice: %d", sum)
	}

	free, _ := f.create(map[string]any{"customer_id": f.customer.ID, "refill_qty": 1})
	if free.FreeQty != 1 || free.Total != 2000 {
		t.Fatalf("loyalty not applied %+v", free)
	}
	blocked, _ := f.create(map[string]any{"customer_id": f.customer.ID, "refill_qty": 1})
	if blocked.FreeQty != 0 {
		t.Fatal("second active free order must not be granted")
	}
	f.h.clock.Advance(4 * 24 * time.Hour)
	f.relogin("6100")
	f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/orders/"+free.ID+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "transfer", "paid": true}), http.StatusOK)
	state = f.customerState()
	if state.StampCount != 0 || state.LoanBalance != 1 || state.PredictionConfidence != "low" || state.DaysPerGallon == nil {
		t.Fatalf("after free delivery %+v", state)
	}
}

func TestLedgerNeverNegativeAndAdjustments(t *testing.T) {
	f := setupOrders(t, "6110")
	v := f.dispatchedOrder(2)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+v.ID+"/deliver", map[string]any{"gallons_returned": 0, "payment_method": "cash", "paid": false}), http.StatusOK)
	if sum, balance := f.ledgerSum(); sum != 2 || balance != 2 {
		t.Fatalf("after delivery sum %d balance %d", sum, balance)
	}
	adjPath := "/api/v1/customers/" + f.customer.ID + "/ledger-adjustments"
	f.owner.mustStatus(f.owner.do(http.MethodPost, adjPath, map[string]any{"kind": "lost", "delta": -3, "note": "Hilang"}), http.StatusUnprocessableEntity)
	f.owner.mustStatus(f.owner.do(http.MethodPost, adjPath, map[string]any{"kind": "lost", "delta": -1, "note": ""}), http.StatusUnprocessableEntity)
	f.owner.mustStatus(f.owner.do(http.MethodPost, adjPath, map[string]any{"kind": "lost", "delta": -1, "note": "Galon pecah"}), http.StatusCreated)
	f.owner.mustStatus(f.owner.do(http.MethodPost, adjPath, map[string]any{"kind": "adjustment", "delta": -1, "note": "Dikembalikan di depot"}), http.StatusCreated)
	if sum, balance := f.ledgerSum(); sum != 0 || balance != 0 {
		t.Fatalf("after adjustments sum %d balance %d", sum, balance)
	}
	f.owner.mustStatus(f.owner.do(http.MethodPost, adjPath, map[string]any{"kind": "adjustment", "delta": -1, "note": "Terlalu banyak"}), http.StatusUnprocessableEntity)

	second := f.dispatchedOrder(1)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+second.ID+"/deliver", map[string]any{"gallons_returned": 2, "payment_method": "cash", "paid": true}), http.StatusUnprocessableEntity)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+second.ID+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}), http.StatusOK)
	if sum, balance := f.ledgerSum(); sum != 0 || balance != 0 {
		t.Fatalf("balanced delivery changed ledger: sum %d balance %d", sum, balance)
	}

	ledger := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/customers/"+f.customer.ID+"/ledger", nil), http.StatusOK)
	var entries []map[string]any
	ledger.data(t, &entries)
	if len(entries) != 3 {
		t.Fatalf("ledger entries %d", len(entries))
	}
	other := f.h.client()
	other.register("Depot Lain", "081261100099")
	other.mustStatus(other.do(http.MethodGet, "/api/v1/customers/"+f.customer.ID+"/ledger", nil), http.StatusNotFound)
	other.mustStatus(other.do(http.MethodPost, adjPath, map[string]any{"kind": "lost", "delta": 1, "note": "Dibajak"}), http.StatusNotFound)
}

func TestConcurrentDeliveriesOnlyOneSucceeds(t *testing.T) {
	f := setupOrders(t, "6120")
	v := f.dispatchedOrder(2)
	path := "/api/v1/orders/" + v.ID + "/deliver"
	const attempts = 6
	results := make([]int, attempts)
	var wg sync.WaitGroup
	for i := range attempts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := f.h.client()
			c.cookies = map[string]string{}
			for k, val := range f.courier.cookies {
				c.cookies[k] = val
			}
			res := c.do(http.MethodPost, path, map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}, withIdempotency("race-"+string(rune('a'+i))))
			results[i] = res.Status
		}(i)
	}
	wg.Wait()
	ok, conflict := 0, 0
	for _, s := range results {
		switch s {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict++
		default:
			t.Fatalf("unexpected status %d", s)
		}
	}
	if ok != 1 || conflict != attempts-1 {
		t.Fatalf("ok=%d conflict=%d", ok, conflict)
	}
	if sum, balance := f.ledgerSum(); sum != 1 || balance != 1 {
		t.Fatalf("ledger sum %d balance %d", sum, balance)
	}
}

func TestCourierQueueAndLocation(t *testing.T) {
	f := setupOrders(t, "6130")
	f.owner.mustStatus(f.owner.do(http.MethodPatch, "/api/v1/depot", map[string]any{"lat": -6.2, "lng": 106.85}), http.StatusOK)
	farCustomer := f.owner.createCustomer("Pak Jauh", "081261300010")
	f.owner.mustStatus(f.owner.do(http.MethodPatch, "/api/v1/customers/"+farCustomer.ID, map[string]any{"lat": -6.3, "lng": 106.85}), http.StatusOK)
	nearCustomer := f.owner.createCustomer("Bu Dekat", "081261300011")
	f.owner.mustStatus(f.owner.do(http.MethodPatch, "/api/v1/customers/"+nearCustomer.ID, map[string]any{"lat": -6.21, "lng": 106.85}), http.StatusOK)

	far, _ := f.create(map[string]any{"customer_id": farCustomer.ID})
	near, _ := f.create(map[string]any{"customer_id": nearCustomer.ID})
	noLoc, _ := f.create(map[string]any{"customer_id": f.customer.ID})
	unassigned, _ := f.create(map[string]any{"customer_id": f.customer.ID})
	for _, id := range []string{far.ID, near.ID, noLoc.ID} {
		f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/orders/"+id+"/assign", map[string]any{"courier_id": f.courierID}), http.StatusOK)
	}
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+far.ID+"/dispatch", nil), http.StatusOK)

	res := f.courier.mustStatus(f.courier.do(http.MethodGet, "/api/v1/courier/queue", nil), http.StatusOK)
	var queue []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	res.data(t, &queue)
	if len(queue) != 3 || queue[0].ID != far.ID || queue[1].ID != near.ID || queue[2].ID != noLoc.ID {
		t.Fatalf("queue order %+v (far=%s near=%s noLoc=%s unassigned=%s)", queue, far.ID, near.ID, noLoc.ID, unassigned.ID)
	}
	f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/courier/queue", nil), http.StatusForbidden)

	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/courier/customers/"+f.customer.ID+"/location", map[string]any{"lat": -6.25, "lng": 106.9}), http.StatusOK)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/courier/customers/"+f.customer.ID+"/location", map[string]any{"lat": 95, "lng": 106.9}), http.StatusUnprocessableEntity)
	state := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/customers/"+f.customer.ID, nil), http.StatusOK)
	var cust struct {
		Lat *float64 `json:"lat"`
	}
	state.data(t, &cust)
	if cust.Lat == nil || *cust.Lat != -6.25 {
		t.Fatalf("location not saved %+v", cust)
	}
	other := f.h.client()
	other.register("Depot Lain", "081261300099")
	other.createCourier("Kurir Lain", "081261300098")
	otherCourier := f.h.client()
	otherCourier.mustStatus(otherCourier.login("081261300098", testPassword), http.StatusOK)
	otherCourier.mustStatus(otherCourier.do(http.MethodPost, "/api/v1/courier/customers/"+f.customer.ID+"/location", map[string]any{"lat": -6.0, "lng": 106.0}), http.StatusNotFound)
}

func TestGallonSummaryAndCustomerOrders(t *testing.T) {
	f := setupOrders(t, "6140")
	v := f.dispatchedOrder(3)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+v.ID+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}), http.StatusOK)

	summary := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/gallons/summary", nil), http.StatusOK)
	var s struct {
		TotalOnLoan int64            `json:"total_on_loan"`
		Idle        []map[string]any `json:"idle"`
	}
	summary.data(t, &s)
	if s.TotalOnLoan != 2 || len(s.Idle) != 0 {
		t.Fatalf("summary %+v", s)
	}
	f.h.clock.Advance(31 * 24 * time.Hour)
	f.relogin("6140")
	summary = f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/gallons/summary", nil), http.StatusOK)
	summary.data(t, &s)
	if len(s.Idle) != 1 {
		t.Fatalf("idle should list the customer: %+v", s)
	}

	orders := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/customers/"+f.customer.ID+"/orders", nil), http.StatusOK)
	var list []orderView
	orders.data(t, &list)
	if len(list) != 1 || list[0].ID != v.ID {
		t.Fatalf("customer orders %+v", list)
	}
}

func (f orderFixture) relogin(seed string) {
	f.h.t.Helper()
	f.owner.cookies = map[string]string{}
	f.owner.mustStatus(f.owner.login("0812"+seed+"0001", testPassword), http.StatusOK)
	f.courier.cookies = map[string]string{}
	f.courier.mustStatus(f.courier.login("0812"+seed+"0002", testPassword), http.StatusOK)
}
