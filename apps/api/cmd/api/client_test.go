package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/benditandayusaputra/depotin/apps/api/internal/config"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/testutil"
)

const (
	testWebOrigin = "http://localhost:5173"
	testPassword  = "kata-sandi-aman-123"
)

type harness struct {
	t     *testing.T
	app   *fiber.App
	deps  dependencies
	clock *clock.Fake
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	pool := testutil.Pool(t)
	cfg, err := config.Load(func(k string) (string, bool) {
		switch k {
		case "DATABASE_URL":
			return "postgres://unused", true
		case "EDGE_KEY":
			return testEdgeKey, true
		case "WEB_ORIGIN":
			return testWebOrigin, true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	clk := clock.NewFake(time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC))
	deps := dependencies{cfg: cfg, log: slog.New(slog.NewJSONHandler(io.Discard, nil)), pool: pool, clock: clk}
	return &harness{t: t, app: newApp(deps), deps: deps, clock: clk}
}

type client struct {
	h       *harness
	cookies map[string]string
	ip      string
}

func (h *harness) client() *client {
	return &client{h: h, cookies: map[string]string{}, ip: "203.0.113.10"}
}

type response struct {
	Status int
	Body   map[string]json.RawMessage
	Header http.Header
}

func (r response) errorCode(t *testing.T) string {
	t.Helper()
	var e struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(r.Body["error"], &e); err != nil {
		t.Fatalf("decode error body: %v (%s)", err, r.Body)
	}
	return e.Code
}

func (r response) data(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body["data"], dst); err != nil {
		t.Fatalf("decode data: %v (%s)", err, r.Body["data"])
	}
}

func (c *client) do(method, path string, body any, opts ...func(*http.Request)) response {
	c.h.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.h.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(httpx.HeaderEdgeKey, testEdgeKey)
	req.Header.Set("X-Client-IP", c.ip)
	if body != nil {
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}
	if method != http.MethodGet {
		req.Header.Set(fiber.HeaderOrigin, testWebOrigin)
	}
	for name, value := range c.cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	for _, opt := range opts {
		opt(req)
	}
	res, err := c.h.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		c.h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()
	for _, ck := range res.Cookies() {
		if ck.MaxAge < 0 || ck.Value == "" {
			delete(c.cookies, ck.Name)
			continue
		}
		c.cookies[ck.Name] = ck.Value
	}
	out := response{Status: res.StatusCode, Header: res.Header, Body: map[string]json.RawMessage{}}
	raw, _ := io.ReadAll(res.Body)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out.Body); err != nil {
			c.h.t.Fatalf("%s %s: body bukan JSON: %s", method, path, raw)
		}
	}
	return out
}

func (c *client) mustStatus(res response, want int) response {
	c.h.t.Helper()
	if res.Status != want {
		c.h.t.Fatalf("status %d, want %d: %s", res.Status, want, res.Body)
	}
	return res
}

type sessionData struct {
	User struct {
		ID      string `json:"id"`
		DepotID string `json:"depot_id"`
		Role    string `json:"role"`
		Phone   string `json:"phone"`
	} `json:"user"`
	Depot struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	} `json:"depot"`
}

func (c *client) register(depotName, phone string) sessionData {
	c.h.t.Helper()
	res := c.mustStatus(c.do(http.MethodPost, "/api/v1/auth/register", map[string]any{
		"depot_name": depotName, "name": "Pemilik " + depotName, "phone": phone, "password": testPassword,
	}), http.StatusCreated)
	var s sessionData
	res.data(c.h.t, &s)
	return s
}

func (c *client) login(phone, password string) response {
	c.h.t.Helper()
	return c.do(http.MethodPost, "/api/v1/auth/login", map[string]any{"phone": phone, "password": password})
}

func (c *client) createCourier(name, phone string) string {
	c.h.t.Helper()
	res := c.mustStatus(c.do(http.MethodPost, "/api/v1/users", map[string]any{
		"name": name, "phone": phone, "password": testPassword,
	}), http.StatusCreated)
	var u struct {
		ID string `json:"id"`
	}
	res.data(c.h.t, &u)
	return u.ID
}
