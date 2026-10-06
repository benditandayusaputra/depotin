package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/internal/config"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

const (
	requestTimeout = 5 * time.Second
	streamPath     = "/api/v1/stream"
	healthPath     = "/healthz"
)

type dependencies struct {
	cfg  config.Config
	log  *slog.Logger
	pool *pgxpool.Pool
}

func newApp(d dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:          "depotin-api",
		BodyLimit:        httpx.MaxBodyBytes,
		ReadTimeout:      15 * time.Second,
		WriteTimeout:     0,
		IdleTimeout:      60 * time.Second,
		JSONEncoder:      json.Marshal,
		JSONDecoder:      json.Unmarshal,
		ErrorHandler:     httpx.ErrorHandler,
		ProxyHeader:      fiber.HeaderXForwardedFor,
		TrustProxy:       true,
		TrustProxyConfig: fiber.TrustProxyConfig{Private: true, Loopback: true},
		StrictRouting:    false,
		CaseSensitive:    true,
	})

	app.Use(httpx.RequestIDMiddleware())
	app.Use(httpx.RecoverMiddleware(d.log))
	app.Use(httpx.LoggerMiddleware(d.log))
	app.Use(httpx.EdgeKeyMiddleware(d.cfg.EdgeKey, func(c fiber.Ctx) bool {
		path := c.Path()
		return path == healthPath || (path == streamPath && c.Method() == http.MethodGet)
	}))

	app.Get(healthPath, func(c fiber.Ctx) error {
		return httpx.OK(c, fiber.Map{"status": "ok"})
	})

	v1 := app.Group("/api/v1")
	v1.Get("/readyz", readyz(d.pool))

	return app
}

func readyz(pool *pgxpool.Pool) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), requestTimeout)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return httpx.Fail(c, httpx.Unavailable(err))
		}
		return httpx.OK(c, fiber.Map{"status": "ok", "database": "ok"})
	}
}
