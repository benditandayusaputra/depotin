package httpx

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/crypto"
)

const (
	HeaderRequestID = "X-Request-ID"
	HeaderEdgeKey   = "X-Edge-Key"
)

func RequestIDMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Get(HeaderRequestID)
		if id == "" || len(id) > 64 {
			id = uuid.NewString()
		}
		c.Locals(localRequestID, id)
		c.Set(HeaderRequestID, id)
		return c.Next()
	}
}

func RecoverMiddleware(log *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.ErrorContext(c.Context(), "panik di handler",
					"request_id", RequestID(c),
					"route", routePattern(c),
					"panic", fmt.Sprint(r),
					"stack", string(debug.Stack()),
				)
				err = Fail(c, Internal(fmt.Errorf("panic: %v", r)))
			}
		}()
		return c.Next()
	}
}

func LoggerMiddleware(log *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		attrs := []any{
			"request_id", RequestID(c),
			"method", c.Method(),
			"route", routePattern(c),
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if p, ok := CurrentPrincipal(c); ok {
			attrs = append(attrs, "depot_id", p.DepotID.String(), "user_id", p.UserID.String())
		}
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}
		log.LogAttrs(c.Context(), level, "permintaan", toAttrs(attrs)...)
		return err
	}
}

func toAttrs(kv []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		key, _ := kv[i].(string)
		attrs = append(attrs, slog.Any(key, kv[i+1]))
	}
	return attrs
}

func EdgeKeyMiddleware(edgeKey string, skip func(c fiber.Ctx) bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		if skip(c) {
			return c.Next()
		}
		if !crypto.ConstantTimeEqual(c.Get(HeaderEdgeKey), edgeKey) {
			return Fail(c, Forbidden())
		}
		c.Locals(localEdgeTrusted, true)
		return c.Next()
	}
}
