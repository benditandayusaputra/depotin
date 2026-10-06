package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/internal/auth"
	"github.com/benditandayusaputra/depotin/apps/api/internal/config"
	"github.com/benditandayusaputra/depotin/apps/api/internal/customer"
	"github.com/benditandayusaputra/depotin/apps/api/internal/depot"
	"github.com/benditandayusaputra/depotin/apps/api/internal/gallon"
	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/crypto"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/ratelimit"
	"github.com/benditandayusaputra/depotin/apps/api/internal/product"
	"github.com/benditandayusaputra/depotin/apps/api/internal/public"
	"github.com/benditandayusaputra/depotin/apps/api/internal/reminder"
	"github.com/benditandayusaputra/depotin/apps/api/internal/report"
	"github.com/benditandayusaputra/depotin/apps/api/internal/stream"
	"github.com/benditandayusaputra/depotin/apps/api/internal/user"
)

const (
	requestTimeout = 5 * time.Second
	streamPath     = "/api/v1/stream"
	healthPath     = "/healthz"
)

var globalRule = ratelimit.Rule{Limit: 300, Window: time.Minute}

type dependencies struct {
	cfg   config.Config
	log   *slog.Logger
	pool  *pgxpool.Pool
	clock clock.Clock
}

func newApp(d dependencies) (*fiber.App, error) {
	if d.clock == nil {
		d.clock = clock.System{}
	}
	limiter := ratelimit.New(d.clock)
	tokens := auth.NewTokenIssuer(d.cfg.JWTSecret)
	cookies := auth.NewCookieWriter(d.cfg.CookieSecure)
	sealer, err := crypto.NewSealer(d.cfg.LinkEncKey)
	if err != nil {
		return nil, err
	}

	app := fiber.New(fiber.Config{
		AppName:          "depotin-api",
		BodyLimit:        httpx.MaxBodyBytes,
		ReadTimeout:      15 * time.Second,
		IdleTimeout:      60 * time.Second,
		JSONEncoder:      json.Marshal,
		JSONDecoder:      json.Unmarshal,
		ErrorHandler:     httpx.ErrorHandler,
		ProxyHeader:      fiber.HeaderXForwardedFor,
		TrustProxy:       true,
		TrustProxyConfig: fiber.TrustProxyConfig{Private: true, Loopback: true},
		CaseSensitive:    true,
	})

	app.Use(httpx.RequestIDMiddleware())
	app.Use(httpx.RecoverMiddleware(d.log))
	app.Use(httpx.LoggerMiddleware(d.log))
	app.Use(httpx.EdgeKeyMiddleware(d.cfg.EdgeKey, func(c fiber.Ctx) bool {
		path := c.Path()
		return path == healthPath || (path == streamPath && c.Method() == http.MethodGet)
	}))
	app.Use(httpx.RateLimitByIP(limiter, globalRule, "global"))

	app.Get(healthPath, func(c fiber.Ctx) error {
		return httpx.OK(c, fiber.Map{"status": "ok"})
	})

	v1 := app.Group("/api/v1")
	v1.Use(httpx.RequireOrigin(d.cfg.WebOrigin))
	v1.Use(auth.Middleware(tokens, cookies, d.clock))
	v1.Get("/readyz", readyz(d.pool))

	hub := stream.NewHub()
	publisher := stream.NewPublisher(hub)
	links := customer.NewLinks(d.cfg.WebOrigin, sealer)

	authSvc := auth.NewService(d.pool, d.clock, tokens)
	auth.NewHandler(authSvc, cookies, d.clock, limiter).Register(v1.Group("/auth"))

	ownerOnly := httpx.RequireRole(auth.RoleOwner)
	user.NewHandler(d.pool, d.clock).Register(v1, ownerOnly)
	customer.NewHandler(d.pool, d.clock, links).Register(v1, ownerOnly)
	orderSvc := order.NewService(d.pool, d.clock, publisher)
	orderHandler := order.NewHandler(orderSvc)
	orderHandler.Register(v1, ownerOnly, auth.RequireLogin())
	orderHandler.RegisterCourier(v1, httpx.RequireRole(auth.RoleCourier))
	gallon.NewHandler(d.pool, d.clock).Register(v1, ownerOnly)
	publicHandler := public.NewHandler(orderSvc, d.clock, limiter)
	publicHandler.Register(v1)
	depot.NewHandler(d.pool, publicHandler.InvalidateDepot).Register(v1, ownerOnly)
	product.NewHandler(d.pool, publicHandler.InvalidateDepotByID).Register(v1, ownerOnly)
	reminderSvc := reminder.NewService(d.pool, d.clock, links, publisher)
	reminder.NewHandler(reminderSvc).Register(v1, ownerOnly)
	report.NewHandler(d.pool, d.clock, reminderSvc).Register(v1, ownerOnly)
	stream.NewHandler(hub, stream.NewTicketStore(d.clock), limiter, d.cfg.WebOrigin).Register(v1, auth.RequireLogin())
	app.Hooks().OnPreShutdown(func() error {
		hub.CloseAll()
		return nil
	})

	return app, nil
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
