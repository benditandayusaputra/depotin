package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"

	minSecretBytes = 32
)

type Config struct {
	AppEnv           string
	HTTPAddr         string
	DatabaseURL      string
	WebOrigin        string
	EdgeKey          string
	JWTSecret        []byte
	LinkEncKey       []byte
	CookieSecure     bool
	DBKeepaliveUntil time.Time
	DemoResetHour    int
	LogLevel         slog.Level
}

type Lookup func(key string) (string, bool)

func Load(lookup Lookup) (Config, error) {
	get := func(key, fallback string) string {
		if v, ok := lookup(key); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		return fallback
	}

	cfg := Config{
		AppEnv:      get("APP_ENV", EnvDevelopment),
		HTTPAddr:    get("HTTP_ADDR", ":8080"),
		DatabaseURL: get("DATABASE_URL", ""),
		WebOrigin:   strings.TrimRight(get("WEB_ORIGIN", "http://localhost:5173"), "/"),
		EdgeKey:     get("EDGE_KEY", ""),
	}

	if cfg.AppEnv != EnvDevelopment && cfg.AppEnv != EnvProduction {
		return Config{}, fmt.Errorf("APP_ENV %q tidak dikenal", cfg.AppEnv)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL wajib diisi")
	}

	isProd := cfg.AppEnv == EnvProduction

	var err error
	if cfg.EdgeKey == "" {
		if isProd {
			return Config{}, errors.New("EDGE_KEY wajib diisi")
		}
		cfg.EdgeKey = "dev-edge-key-dev-edge-key-dev-edge-key"
	}
	if isProd && len(cfg.EdgeKey) < minSecretBytes {
		return Config{}, errors.New("EDGE_KEY terlalu pendek")
	}

	cfg.JWTSecret, err = secretBytes(get("JWT_SECRET", ""), "JWT_SECRET", isProd, []byte("dev-jwt-secret-dev-jwt-secret-dev-jwt-secret"))
	if err != nil {
		return Config{}, err
	}

	cfg.LinkEncKey, err = secretBytes(get("LINK_ENC_KEY", ""), "LINK_ENC_KEY", isProd, []byte("dev-link-enc-key-dev-link-enc-ke"))
	if err != nil {
		return Config{}, err
	}
	if len(cfg.LinkEncKey) != minSecretBytes {
		return Config{}, fmt.Errorf("LINK_ENC_KEY harus tepat %d byte setelah dekode base64", minSecretBytes)
	}

	cfg.CookieSecure, err = strconv.ParseBool(get("COOKIE_SECURE", strconv.FormatBool(isProd)))
	if err != nil {
		return Config{}, fmt.Errorf("COOKIE_SECURE tidak valid: %w", err)
	}
	if isProd && !cfg.CookieSecure {
		return Config{}, errors.New("COOKIE_SECURE harus true di produksi")
	}

	if raw := get("DB_KEEPALIVE_UNTIL", ""); raw != "" {
		cfg.DBKeepaliveUntil, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return Config{}, fmt.Errorf("DB_KEEPALIVE_UNTIL tidak valid: %w", err)
		}
	}

	cfg.DemoResetHour = -1
	if raw := get("DEMO_RESET_HOUR", ""); raw != "" {
		cfg.DemoResetHour, err = strconv.Atoi(raw)
		if err != nil || cfg.DemoResetHour < 0 || cfg.DemoResetHour > 23 {
			return Config{}, fmt.Errorf("DEMO_RESET_HOUR harus 0 sampai 23, dapat %q", raw)
		}
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(get("LOG_LEVEL", "info"))); err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL tidak valid: %w", err)
	}

	return cfg, nil
}

func (c Config) IsProduction() bool {
	return c.AppEnv == EnvProduction
}

func secretBytes(raw, name string, isProd bool, devFallback []byte) ([]byte, error) {
	if raw == "" {
		if isProd {
			return nil, fmt.Errorf("%s wajib diisi", name)
		}
		return devFallback, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		decoded = []byte(raw)
	}
	if len(decoded) < minSecretBytes {
		return nil, fmt.Errorf("%s terlalu pendek, minimal %d byte", name, minSecretBytes)
	}
	return decoded, nil
}
