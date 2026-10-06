package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestStreamTicketIsSingleUse(t *testing.T) {
	h := newHarness(t)
	owner := h.client()
	owner.register("Depot Stream", "081281000001")
	anon := h.client()
	anon.mustStatus(anon.do(http.MethodPost, "/api/v1/stream/tickets", nil), http.StatusUnauthorized)

	res := owner.mustStatus(owner.do(http.MethodPost, "/api/v1/stream/tickets", nil), http.StatusOK)
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	res.data(t, &ticket)
	if ticket.Ticket == "" {
		t.Fatal("ticket missing")
	}

	open := func() *http.Response {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stream?ticket="+ticket.Ticket, nil)
		req.Header.Set(fiber.HeaderOrigin, testWebOrigin)
		resp, err := h.app.Test(req, fiber.TestConfig{Timeout: 300 * time.Millisecond, FailOnTimeout: false})
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	first := open()
	if first.StatusCode != http.StatusOK || !strings.HasPrefix(first.Header.Get(fiber.HeaderContentType), "text/event-stream") {
		t.Fatalf("status %d type %q", first.StatusCode, first.Header.Get(fiber.HeaderContentType))
	}
	if first.Header.Get(fiber.HeaderAccessControlAllowOrigin) != testWebOrigin {
		t.Fatal("cors origin missing")
	}
	body, _ := io.ReadAll(first.Body)
	if !strings.Contains(string(body), "event: ready") {
		t.Fatalf("body %q", body)
	}
	second := open()
	if second.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ticket reused, status %d", second.StatusCode)
	}
	bogus := httptest.NewRequest(http.MethodGet, "/api/v1/stream?ticket=salah", nil)
	resp, err := h.app.Test(bogus)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bogus ticket status %d", resp.StatusCode)
	}
}
