package auth

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/benditandayusaputra/depotin/apps/api/internal/depot"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/phone"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/ratelimit"
	"github.com/benditandayusaputra/depotin/apps/api/internal/user"
)

var (
	loginRule    = ratelimit.Rule{Limit: 5, Window: time.Minute}
	registerRule = ratelimit.Rule{Limit: 3, Window: time.Hour}
)

type Handler struct {
	svc     *Service
	cookies CookieWriter
	clock   clock.Clock
	limiter *ratelimit.Limiter
}

func NewHandler(svc *Service, cookies CookieWriter, clk clock.Clock, limiter *ratelimit.Limiter) *Handler {
	return &Handler{svc: svc, cookies: cookies, clock: clk, limiter: limiter}
}

func (h *Handler) Register(r fiber.Router) {
	r.Post("/register", httpx.RateLimitByIP(h.limiter, registerRule, "register"), h.register)
	r.Post("/login", h.login)
	r.Post("/refresh", h.refresh)
	r.Post("/logout", h.logout)
	r.Post("/logout-all", RequireLogin(), h.logoutAll)
	r.Get("/me", RequireLogin(), h.me)
	r.Post("/password", RequireLogin(), h.changePassword)
}

type registerRequest struct {
	DepotName string `json:"depot_name" validate:"required,min=3,max=80"`
	Name      string `json:"name" validate:"required,min=2,max=80"`
	Phone     string `json:"phone" validate:"required,min=9,max=20"`
	Password  string `json:"password" validate:"required,min=10,max=128"`
}

type sessionResponse struct {
	User  user.View  `json:"user"`
	Depot depot.View `json:"depot"`
}

func (h *Handler) register(c fiber.Ctx) error {
	var req registerRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	normalized, err := phone.Normalize(req.Phone)
	if err != nil {
		return httpx.Fail(c, httpx.Validation(map[string]string{"phone": "Nomor HP tidak valid."}))
	}
	u, d, tokens, err := h.svc.Register(c.Context(), RegisterInput{
		DepotName: req.DepotName, OwnerName: req.Name, Phone: normalized, Password: req.Password, Meta: h.meta(c),
	})
	if err != nil {
		if errors.Is(err, ErrPhoneTaken) {
			return httpx.Fail(c, httpx.Validation(map[string]string{"phone": "Nomor HP sudah terdaftar."}))
		}
		return httpx.Fail(c, err)
	}
	h.cookies.Set(c, tokens.Access, tokens.Refresh, h.clock.Now())
	return httpx.Created(c, sessionResponse{User: user.ToView(u), Depot: depot.ToView(d)})
}

type loginRequest struct {
	Phone    string `json:"phone" validate:"required,min=9,max=20"`
	Password string `json:"password" validate:"required,max=128"`
}

func (h *Handler) login(c fiber.Ctx) error {
	var req loginRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	normalized, err := phone.Normalize(req.Phone)
	if err != nil {
		return httpx.Fail(c, httpx.Unauthenticated().WithMessage(ErrInvalidCredentials))
	}
	if err := httpx.CheckLimit(c, h.limiter, loginRule, "login:"+httpx.ClientIP(c)+":"+normalized); err != nil {
		return httpx.Fail(c, err)
	}
	u, tokens, err := h.svc.Login(c.Context(), normalized, req.Password, h.meta(c))
	if err != nil {
		return httpx.Fail(c, loginError(err))
	}
	d, err := h.svc.q.GetDepot(c.Context(), u.DepotID)
	if err != nil {
		return httpx.Fail(c, err)
	}
	h.cookies.Set(c, tokens.Access, tokens.Refresh, h.clock.Now())
	return httpx.OK(c, sessionResponse{User: user.ToView(u), Depot: depot.ToView(d)})
}

func loginError(err error) error {
	switch {
	case errors.Is(err, ErrAccountLocked):
		return httpx.Conflict("Akun terkunci sementara. Coba lagi dalam 15 menit.")
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrUserInactive):
		return httpx.Unauthenticated().WithMessage(ErrInvalidCredentials)
	}
	return err
}

func (h *Handler) refresh(c fiber.Ctx) error {
	raw := c.Cookies(h.cookies.RefreshName())
	if raw == "" {
		return httpx.Fail(c, httpx.Unauthenticated())
	}
	tokens, err := h.svc.Refresh(c.Context(), raw, h.meta(c))
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			h.cookies.Clear(c)
			return httpx.Fail(c, httpx.Unauthenticated())
		}
		return httpx.Fail(c, err)
	}
	h.cookies.Set(c, tokens.Access, tokens.Refresh, h.clock.Now())
	return httpx.OK(c, fiber.Map{"refreshed": true})
}

func (h *Handler) logout(c fiber.Ctx) error {
	if err := h.svc.Logout(c.Context(), c.Cookies(h.cookies.RefreshName())); err != nil {
		return httpx.Fail(c, err)
	}
	h.cookies.Clear(c)
	return httpx.OK(c, fiber.Map{"logged_out": true})
}

func (h *Handler) logoutAll(c fiber.Ctx) error {
	p, _ := httpx.CurrentPrincipal(c)
	if err := h.svc.LogoutAll(c.Context(), p.UserID); err != nil {
		return httpx.Fail(c, err)
	}
	h.cookies.Clear(c)
	return httpx.OK(c, fiber.Map{"logged_out": true})
}

func (h *Handler) me(c fiber.Ctx) error {
	p, _ := httpx.CurrentPrincipal(c)
	u, d, err := h.svc.Me(c.Context(), p.UserID)
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, sessionResponse{User: user.ToView(u), Depot: depot.ToView(d)})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,max=128"`
	NewPassword     string `json:"new_password" validate:"required,min=10,max=128"`
}

func (h *Handler) changePassword(c fiber.Ctx) error {
	var req changePasswordRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	if err := h.svc.ChangePassword(c.Context(), p.UserID, p.SessionFamily, req.CurrentPassword, req.NewPassword, httpx.ClientIP(c)); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return httpx.Fail(c, httpx.Validation(map[string]string{"current_password": "Kata sandi saat ini salah."}))
		}
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, fiber.Map{"changed": true})
}

func (h *Handler) meta(c fiber.Ctx) RequestMeta {
	return RequestMeta{UserAgent: c.Get(fiber.HeaderUserAgent), IP: httpx.ClientIP(c)}
}
