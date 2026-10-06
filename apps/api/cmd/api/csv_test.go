package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

func TestExportCSVContent(t *testing.T) {
	f := setupOrders(t, "9120")
	v := f.dispatchedOrder(2)
	f.courier.mustStatus(f.courier.do(http.MethodPost, "/api/v1/orders/"+v.ID+"/deliver", map[string]any{"gallons_returned": 2, "payment_method": "qris", "paid": true}), http.StatusOK)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/export.csv?from=2026-10-06&to=2026-10-06", nil)
	req.Header.Set(httpx.HeaderEdgeKey, testEdgeKey)
	for name, value := range f.owner.cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	res, err := f.h.app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(res.Header.Get(fiber.HeaderContentType), "text/csv") {
		t.Fatalf("status %d type %s", res.StatusCode, res.Header.Get(fiber.HeaderContentType))
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "kode,tanggal,nama") {
		t.Fatalf("csv %q", body)
	}
	if !strings.Contains(lines[1], v.Code) || !strings.Contains(lines[1], "Bu Rina") || !strings.Contains(lines[1], ",2,0,14000,qris,paid,owner,Kurir Andi") {
		t.Fatalf("row %q", lines[1])
	}
}
