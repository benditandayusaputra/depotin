package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func mapLookup(m map[string]string) Lookup {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func TestLoadDevelopmentUsesFallbackSecrets(t *testing.T) {
	cfg, err := Load(mapLookup(map[string]string{"DATABASE_URL": "postgres://x"}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.EdgeKey == "" || len(cfg.JWTSecret) < 32 || len(cfg.LinkEncKey) != 32 {
		t.Fatalf("fallback secrets missing: %+v", cfg)
	}
	if cfg.CookieSecure {
		t.Fatal("development should default to insecure cookies")
	}
	if cfg.DemoResetHour != -1 {
		t.Fatalf("demo reset should be off, got %d", cfg.DemoResetHour)
	}
}

func TestLoadProductionRejectsMissingOrShortSecrets(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	cases := map[string]map[string]string{
		"missing edge key": {"APP_ENV": "production", "DATABASE_URL": "postgres://x", "JWT_SECRET": key, "LINK_ENC_KEY": key},
		"short edge key":   {"APP_ENV": "production", "DATABASE_URL": "postgres://x", "EDGE_KEY": "short", "JWT_SECRET": key, "LINK_ENC_KEY": key},
		"short jwt":        {"APP_ENV": "production", "DATABASE_URL": "postgres://x", "EDGE_KEY": strings.Repeat("e", 32), "JWT_SECRET": "abc", "LINK_ENC_KEY": key},
		"insecure cookie":  {"APP_ENV": "production", "DATABASE_URL": "postgres://x", "EDGE_KEY": strings.Repeat("e", 32), "JWT_SECRET": key, "LINK_ENC_KEY": key, "COOKIE_SECURE": "false"},
		"missing db":       {"APP_ENV": "production", "EDGE_KEY": strings.Repeat("e", 32), "JWT_SECRET": key, "LINK_ENC_KEY": key},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(mapLookup(env)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestLoadProductionAcceptsValidConfig(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	cfg, err := Load(mapLookup(map[string]string{
		"APP_ENV":            "production",
		"DATABASE_URL":       "postgres://x",
		"EDGE_KEY":           strings.Repeat("e", 40),
		"JWT_SECRET":         key,
		"LINK_ENC_KEY":       key,
		"WEB_ORIGIN":         "https://depotin.vercel.app/",
		"DEMO_RESET_HOUR":    "3",
		"LOG_LEVEL":          "warn",
		"DB_KEEPALIVE_UNTIL": "2026-11-14T12:00:00Z",
	}))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.WebOrigin != "https://depotin.vercel.app" {
		t.Fatalf("web origin not trimmed: %q", cfg.WebOrigin)
	}
	if !cfg.CookieSecure || cfg.DemoResetHour != 3 || cfg.DBKeepaliveUntil.IsZero() {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
