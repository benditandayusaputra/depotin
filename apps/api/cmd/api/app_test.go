package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/benditandayusaputra/depotin/apps/api/internal/config"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/testutil"
)

const testEdgeKey = "test-edge-key-test-edge-key-test"

func testApp(t *testing.T) *dependencies {
	t.Helper()
	pool := testutil.Pool(t)
	cfg, err := config.Load(func(k string) (string, bool) {
		switch k {
		case "DATABASE_URL":
			return "postgres://unused", true
		case "EDGE_KEY":
			return testEdgeKey, true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return &dependencies{cfg: cfg, log: log, pool: pool}
}

func decodeError(t *testing.T, res *http.Response) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.Error.Code
}

func TestHealthzWithoutEdgeKey(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL kosong")
	}
	app := newApp(*testApp(t))
	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if res.Header.Get(httpx.HeaderRequestID) == "" {
		t.Fatal("request id header missing")
	}
}

func TestReadyzRejectsWithoutEdgeKey(t *testing.T) {
	app := newApp(*testApp(t))
	cases := map[string]string{"missing": "", "wrong": "bukan-kunci"}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/readyz", nil)
			if key != "" {
				req.Header.Set(httpx.HeaderEdgeKey, key)
			}
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusForbidden {
				t.Fatalf("status %d", res.StatusCode)
			}
			if code := decodeError(t, res); code != httpx.CodeForbidden {
				t.Fatalf("code %q", code)
			}
		})
	}
}

func TestReadyzWithEdgeKey(t *testing.T) {
	deps := testApp(t)
	app := newApp(*deps)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/readyz", nil)
	req.Header.Set(httpx.HeaderEdgeKey, testEdgeKey)
	req.Header.Set(httpx.HeaderRequestID, "req-123")
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if got := res.Header.Get(httpx.HeaderRequestID); got != "req-123" {
		t.Fatalf("request id %q", got)
	}
	var body struct {
		Data struct {
			Status   string `json:"status"`
			Database string `json:"database"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Status != "ok" || body.Data.Database != "ok" {
		t.Fatalf("body %+v", body)
	}
	if err := deps.pool.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownRouteReturnsStandardError(t *testing.T) {
	app := newApp(*testApp(t))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tidak-ada", nil)
	req.Header.Set(httpx.HeaderEdgeKey, testEdgeKey)
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", res.StatusCode)
	}
	if code := decodeError(t, res); code != httpx.CodeNotFound {
		t.Fatalf("code %q", code)
	}
}
