package main

import (
	"net/http"
	"testing"
)

type productView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Price    int64  `json:"price"`
	IsActive bool   `json:"is_active"`
}

func (c *client) listProducts() []productView {
	c.h.t.Helper()
	res := c.mustStatus(c.do(http.MethodGet, "/api/v1/products", nil), http.StatusOK)
	var out []productView
	res.data(c.h.t, &out)
	return out
}

func TestProductsCrudAndIsolation(t *testing.T) {
	h := newHarness(t)
	a := h.client()
	a.register("Depot Produk A", "081211120001")
	b := h.client()
	b.register("Depot Produk B", "081211120002")

	initial := a.listProducts()
	if len(initial) != 1 || initial[0].Kind != "refill" || initial[0].Name != "Isi ulang galon" {
		t.Fatalf("default product missing: %+v", initial)
	}
	res := a.mustStatus(a.do(http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Galon baru + isi", "kind": "new_gallon", "price": 45000, "sort_order": 2,
	}), http.StatusCreated)
	var created productView
	res.data(t, &created)

	a.mustStatus(a.do(http.MethodPost, "/api/v1/products", map[string]any{"name": "Galon baru + isi", "kind": "other", "price": 1}), http.StatusUnprocessableEntity)
	a.mustStatus(a.do(http.MethodPost, "/api/v1/products", map[string]any{"name": "Salah", "kind": "bogus", "price": 1}), http.StatusUnprocessableEntity)

	upd := a.mustStatus(a.do(http.MethodPatch, "/api/v1/products/"+created.ID, map[string]any{"price": 47000, "is_active": false}), http.StatusOK)
	var updated productView
	upd.data(t, &updated)
	if updated.Price != 47000 || updated.IsActive {
		t.Fatalf("update not applied: %+v", updated)
	}

	b.mustStatus(b.do(http.MethodPatch, "/api/v1/products/"+created.ID, map[string]any{"price": 1}), http.StatusNotFound)
	if got := b.listProducts(); len(got) != 1 {
		t.Fatalf("depot B sees %d products", len(got))
	}

	courierPhone := "081211120003"
	a.createCourier("Kurir", courierPhone)
	courier := h.client()
	courier.mustStatus(courier.login(courierPhone, testPassword), http.StatusOK)
	courier.mustStatus(courier.do(http.MethodGet, "/api/v1/products", nil), http.StatusForbidden)
}
