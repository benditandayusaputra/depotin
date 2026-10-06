package auth

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	refreshCookiePath = "/api/v1/auth"
)

type CookieWriter struct {
	secure bool
}

func NewCookieWriter(secure bool) CookieWriter {
	return CookieWriter{secure: secure}
}

func (w CookieWriter) AccessName() string {
	if w.secure {
		return "__Host-dp_at"
	}
	return "dp_at"
}

func (w CookieWriter) RefreshName() string {
	if w.secure {
		return "__Secure-dp_rt"
	}
	return "dp_rt"
}

func (w CookieWriter) Set(c fiber.Ctx, access, refresh string, now time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     w.AccessName(),
		Value:    access,
		Path:     "/",
		MaxAge:   int(AccessTokenTTL.Seconds()),
		Expires:  now.Add(AccessTokenTTL),
		Secure:   w.secure,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
	c.Cookie(&fiber.Cookie{
		Name:     w.RefreshName(),
		Value:    refresh,
		Path:     refreshCookiePath,
		MaxAge:   int(RefreshTokenTTL.Seconds()),
		Expires:  now.Add(RefreshTokenTTL),
		Secure:   w.secure,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}

func (w CookieWriter) Clear(c fiber.Ctx) {
	expired := time.Unix(0, 0)
	c.Cookie(&fiber.Cookie{Name: w.AccessName(), Value: "", Path: "/", MaxAge: -1, Expires: expired, Secure: w.secure, HTTPOnly: true, SameSite: fiber.CookieSameSiteLaxMode})
	c.Cookie(&fiber.Cookie{Name: w.RefreshName(), Value: "", Path: refreshCookiePath, MaxAge: -1, Expires: expired, Secure: w.secure, HTTPOnly: true, SameSite: fiber.CookieSameSiteStrictMode})
}
