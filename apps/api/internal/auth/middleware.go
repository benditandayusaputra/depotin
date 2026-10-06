package auth

import (
	"github.com/gofiber/fiber/v3"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

func Middleware(tokens *TokenIssuer, cookies CookieWriter, clk clock.Clock) fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := c.Cookies(cookies.AccessName())
		if raw == "" {
			return c.Next()
		}
		claims, err := tokens.Parse(raw, clk.Now())
		if err != nil {
			return c.Next()
		}
		httpx.SetPrincipal(c, httpx.Principal{
			UserID:        claims.UserID(),
			DepotID:       claims.DepotID,
			Role:          claims.Role,
			SessionFamily: claims.FamilyID,
		})
		return c.Next()
	}
}

func RequireLogin() fiber.Handler {
	return func(c fiber.Ctx) error {
		if _, ok := httpx.CurrentPrincipal(c); !ok {
			return httpx.Fail(c, httpx.Unauthenticated())
		}
		return c.Next()
	}
}
