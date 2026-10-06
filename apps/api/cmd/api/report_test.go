package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

type todayView struct {
	OrdersByStatus  map[string]int64 `json:"orders_by_status"`
	Revenue         int64            `json:"revenue"`
	GallonsSold     int64            `json:"gallons_sold"`
	DeliveredOrders int64            `json:"delivered_orders"`
	ExpectedDemand  int64            `json:"expected_demand"`
	RemindersQueued int64            `json:"reminders_queued"`
	GallonsOnLoan   int64            `json:"gallons_on_loan"`
}

type summaryView struct {
	Revenue            int64            `json:"revenue"`
	GallonsSold        int64            `json:"gallons_sold"`
	DeliveredOrders    int64            `json:"delivered_orders"`
	OrdersBySource     map[string]int64 `json:"orders_by_source"`
	RemindersSent      int64            `json:"reminders_sent"`
	RemindersConverted int64            `json:"reminders_converted"`
	ConversionRate     float64          `json:"conversion_rate"`
	NewCustomers       int64            `json:"new_customers"`
	ActiveCustomers    int64            `json:"active_customers"`
	Daily              []struct {
		Day     string `json:"day"`
		Revenue int64  `json:"revenue"`
	} `json:"daily"`
}

func TestDashboardAndReportMatchManualNumbers(t *testing.T) {
	f := setupOrders(t, "9100")
	budi := f.owner.createCustomer("Pak Budi", "081291000010")

	delivered1 := f.dispatchedOrder(2)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+delivered1.ID+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}), http.StatusOK)
	budiOrder, _ := f.create(map[string]any{"customer_id": budi.ID, "refill_qty": 3, "fulfilment": "pickup"})
	f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/orders/"+budiOrder.ID+"/deliver", map[string]any{"gallons_returned": 3, "payment_method": "transfer", "paid": true}), http.StatusOK)
	pendingOrder, _ := f.create(map[string]any{"customer_id": f.customer.ID})
	cancelled, _ := f.create(map[string]any{"customer_id": budi.ID})
	f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/orders/"+cancelled.ID+"/cancel", map[string]any{"reason": "Salah pesan"}), http.StatusOK)
	_ = pendingOrder

	res := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/dashboard/today", nil), http.StatusOK)
	var today todayView
	res.data(t, &today)
	if today.Revenue != 14000+18000 || today.GallonsSold != 5 || today.DeliveredOrders != 2 {
		t.Fatalf("today totals %+v", today)
	}
	if today.OrdersByStatus["delivered"] != 2 || today.OrdersByStatus["confirmed"] != 1 || today.OrdersByStatus["cancelled"] != 1 || today.OrdersByStatus["pending"] != 0 {
		t.Fatalf("by status %+v", today.OrdersByStatus)
	}
	if today.GallonsOnLoan != 1 {
		t.Fatalf("on loan %d", today.GallonsOnLoan)
	}

	summary := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/reports/summary?from=2026-10-01&to=2026-10-06", nil), http.StatusOK)
	var s summaryView
	summary.data(t, &s)
	if s.Revenue != 32000 || s.GallonsSold != 5 || s.DeliveredOrders != 2 || s.ActiveCustomers != 2 || s.NewCustomers != 2 {
		t.Fatalf("summary %+v", s)
	}
	if s.OrdersBySource["owner"] != 4 || s.OrdersBySource["public"] != 0 {
		t.Fatalf("by source %+v", s.OrdersBySource)
	}
	if len(s.Daily) != 1 || s.Daily[0].Day != "2026-10-06" || s.Daily[0].Revenue != 32000 {
		t.Fatalf("daily %+v", s.Daily)
	}
	f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/reports/summary?from=2026-10-06&to=2026-10-01", nil), http.StatusUnprocessableEntity)
	f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/reports/summary?from=2026-01-01&to=2026-10-06", nil), http.StatusUnprocessableEntity)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/export.csv?from=2026-10-06&to=2026-10-06", nil)
	req.Header.Set(httpx.HeaderEdgeKey, testEdgeKey)
	for name, value := range f.owner.cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	csvRes, err := f.h.app.Test(req)
	if err != nil || csvRes.StatusCode != http.StatusOK {
		t.Fatalf("csv status %v %v", csvRes, err)
	}
	auditRes := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/audit", nil), http.StatusOK)
	var logs []struct {
		Action string `json:"action"`
	}
	auditRes.data(t, &logs)
	if len(logs) == 0 || logs[0].Action != "report_exported" {
		t.Fatalf("audit %+v", logs)
	}
	f.courier.mustStatus(f.courier.do(http.MethodGet, "/api/v1/dashboard/today", nil), http.StatusForbidden)
}

func TestReportConversionRate(t *testing.T) {
	f := setupOrders(t, "9110")
	f.owner.mustStatus(f.owner.do(http.MethodPatch, "/api/v1/depot", map[string]any{"default_days_per_gallon": 0.5}), http.StatusOK)
	v := f.dispatchedOrder(1)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+v.ID+"/deliver", map[string]any{"gallons_returned": 1, "payment_method": "cash", "paid": true}), http.StatusOK)
	queue := f.owner.reminders()
	if len(queue) != 1 {
		t.Fatalf("queue %+v", queue)
	}
	sentRes := f.owner.mustStatus(f.owner.do(http.MethodPost, "/api/v1/reminders/"+queue[0].ID+"/send", nil), http.StatusOK)
	var sent struct {
		Link string `json:"link"`
	}
	sentRes.data(t, &sent)
	token := strings.TrimSuffix(strings.TrimPrefix(sent.Link, testWebOrigin+"/p/"), "?r="+queue[0].ID)
	anon := f.h.client()
	anon.mustStatus(anon.do(http.MethodPost, "/api/v1/public/me/"+token+"/orders", map[string]any{"r": queue[0].ID}), http.StatusCreated)

	summary := f.owner.mustStatus(f.owner.do(http.MethodGet, "/api/v1/reports/summary?from=2026-10-06&to=2026-10-06", nil), http.StatusOK)
	var s summaryView
	summary.data(t, &s)
	if s.RemindersSent != 1 || s.RemindersConverted != 1 || s.ConversionRate != 1 || s.OrdersBySource["reminder"] != 1 {
		t.Fatalf("conversion %+v", s)
	}
}
