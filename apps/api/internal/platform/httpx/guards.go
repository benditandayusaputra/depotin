package httpx

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/ratelimit"
)

func RequireRole(roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		p, ok := CurrentPrincipal(c)
		if !ok {
			return Fail(c, Unauthenticated())
		}
		for _, r := range roles {
			if p.Role == r {
				return c.Next()
			}
		}
		return Fail(c, Forbidden())
	}
}

func RequireOrigin(webOrigin string) fiber.Handler {
	return func(c fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}
		if c.Get(fiber.HeaderOrigin) != webOrigin {
			return Fail(c, Forbidden())
		}
		if len(c.Body()) > 0 && !strings.HasPrefix(c.Get(fiber.HeaderContentType), fiber.MIMEApplicationJSON) {
			return Fail(c, ValidationMessage("Content-Type harus application/json."))
		}
		return c.Next()
	}
}

type KeyFunc func(c fiber.Ctx) string

func RateLimit(limiter *ratelimit.Limiter, rule ratelimit.Rule, key KeyFunc) fiber.Handler {
	return func(c fiber.Ctx) error {
		k := key(c)
		if k == "" {
			return c.Next()
		}
		if ok, retry := limiter.Allow(k, rule); !ok {
			return Fail(c, RateLimited(retryAfterSeconds(retry)))
		}
		return c.Next()
	}
}

func RateLimitByIP(limiter *ratelimit.Limiter, rule ratelimit.Rule, scope string) fiber.Handler {
	return RateLimit(limiter, rule, func(c fiber.Ctx) string {
		return scope + ":" + ClientIP(c)
	})
}

func CheckLimit(c fiber.Ctx, limiter *ratelimit.Limiter, rule ratelimit.Rule, key string) error {
	if ok, retry := limiter.Allow(key, rule); !ok {
		return RateLimited(retryAfterSeconds(retry))
	}
	return nil
}

func retryAfterSeconds(d time.Duration) int {
	secs := int(d.Round(time.Second).Seconds())
	if secs < 1 {
		return 1
	}
	return secs
}
