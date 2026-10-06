package user

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/audit"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/crypto"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/phone"
)

const roleCourier = "courier"

type Handler struct {
	pool  *pgxpool.Pool
	q     *sqlc.Queries
	clock clock.Clock
}

func NewHandler(pool *pgxpool.Pool, clk clock.Clock) *Handler {
	return &Handler{pool: pool, q: sqlc.New(pool), clock: clk}
}

func (h *Handler) Register(r fiber.Router) {
	r.Get("/users", h.list)
	r.Post("/users", h.create)
	r.Patch("/users/:id", h.update)
	r.Post("/users/:id/reset-password", h.resetPassword)
}

func (h *Handler) list(c fiber.Ctx) error {
	p, _ := httpx.CurrentPrincipal(c)
	users, err := h.q.ListUsersByDepot(c.Context(), p.DepotID)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("daftar pengguna: %w", err))
	}
	return httpx.OK(c, ToViews(users))
}

type createRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=80"`
	Phone    string `json:"phone" validate:"required,min=9,max=20"`
	Password string `json:"password" validate:"required,min=10,max=128"`
}

func (h *Handler) create(c fiber.Ctx) error {
	var req createRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	normalized, err := phone.Normalize(req.Phone)
	if err != nil {
		return httpx.Fail(c, httpx.Validation(map[string]string{"phone": "Nomor HP tidak valid."}))
	}
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	var created sqlc.User
	err = db.WithTx(c.Context(), h.pool, func(tx pgx.Tx) error {
		q := h.q.WithTx(tx)
		created, err = q.CreateUser(c.Context(), sqlc.CreateUserParams{
			ID: idgen.NewID(), DepotID: p.DepotID, Role: roleCourier, Name: req.Name, Phone: normalized, PasswordHash: hash,
		})
		if err != nil {
			return fmt.Errorf("buat kurir: %w", err)
		}
		return audit.Record(c.Context(), q, audit.Entry{
			DepotID: p.DepotID, UserID: &p.UserID, Action: audit.ActionUserCreated, EntityType: "user", EntityID: &created.ID, IP: httpx.ClientIP(c),
		})
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return httpx.Fail(c, httpx.Validation(map[string]string{"phone": "Nomor HP sudah terdaftar."}))
		}
		return httpx.Fail(c, err)
	}
	return httpx.Created(c, ToView(created))
}

type updateRequest struct {
	Name     *string `json:"name" validate:"omitempty,min=2,max=80"`
	IsActive *bool   `json:"is_active"`
}

func (h *Handler) update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound(""))
	}
	var req updateRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	if id == p.UserID && req.IsActive != nil && !*req.IsActive {
		return httpx.Fail(c, httpx.Validation(map[string]string{"is_active": "Akun sendiri tidak bisa dinonaktifkan."}))
	}
	var updated sqlc.User
	err = db.WithTx(c.Context(), h.pool, func(tx pgx.Tx) error {
		q := h.q.WithTx(tx)
		updated, err = q.UpdateUserProfile(c.Context(), sqlc.UpdateUserProfileParams{ID: id, DepotID: p.DepotID, Name: req.Name, IsActive: req.IsActive})
		if err != nil {
			return fmt.Errorf("ubah pengguna: %w", err)
		}
		if req.IsActive != nil && !*req.IsActive {
			now := h.clock.Now()
			if err := q.RevokeUserSessions(c.Context(), sqlc.RevokeUserSessionsParams{UserID: id, RevokedAt: &now}); err != nil {
				return fmt.Errorf("cabut sesi: %w", err)
			}
		}
		return audit.Record(c.Context(), q, audit.Entry{
			DepotID: p.DepotID, UserID: &p.UserID, Action: audit.ActionUserUpdated, EntityType: "user", EntityID: &id, IP: httpx.ClientIP(c),
		})
	})
	if err != nil {
		if db.IsNoRows(err) {
			return httpx.Fail(c, httpx.NotFound(""))
		}
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, ToView(updated))
}

type resetPasswordRequest struct {
	Password string `json:"password" validate:"required,min=10,max=128"`
}

func (h *Handler) resetPassword(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound(""))
	}
	var req resetPasswordRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	target, err := h.q.GetUserInDepot(c.Context(), sqlc.GetUserInDepotParams{ID: id, DepotID: p.DepotID})
	if err != nil {
		if db.IsNoRows(err) {
			return httpx.Fail(c, httpx.NotFound(""))
		}
		return httpx.Fail(c, fmt.Errorf("cari pengguna: %w", err))
	}
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return httpx.Fail(c, err)
	}
	now := h.clock.Now()
	err = db.WithTx(c.Context(), h.pool, func(tx pgx.Tx) error {
		q := h.q.WithTx(tx)
		if err := q.UpdateUserPassword(c.Context(), sqlc.UpdateUserPasswordParams{ID: target.ID, PasswordHash: hash}); err != nil {
			return fmt.Errorf("setel ulang kata sandi: %w", err)
		}
		if err := q.RevokeUserSessions(c.Context(), sqlc.RevokeUserSessionsParams{UserID: target.ID, RevokedAt: &now}); err != nil {
			return fmt.Errorf("cabut sesi: %w", err)
		}
		return audit.Record(c.Context(), q, audit.Entry{
			DepotID: p.DepotID, UserID: &p.UserID, Action: audit.ActionUserPasswordReset, EntityType: "user", EntityID: &target.ID, IP: httpx.ClientIP(c),
		})
	})
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, fiber.Map{"reset": true})
}
