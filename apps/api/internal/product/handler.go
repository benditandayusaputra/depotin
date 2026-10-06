package product

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/audit"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
)

type View struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Price     int64     `json:"price"`
	IsActive  bool      `json:"is_active"`
	SortOrder int32     `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

func ToView(p sqlc.Product) View {
	return View{ID: p.ID, Name: p.Name, Kind: p.Kind, Price: p.Price, IsActive: p.IsActive, SortOrder: p.SortOrder, CreatedAt: p.CreatedAt}
}

func ToViews(products []sqlc.Product) []View {
	out := make([]View, 0, len(products))
	for _, p := range products {
		out = append(out, ToView(p))
	}
	return out
}

type Handler struct {
	pool      *pgxpool.Pool
	q         *sqlc.Queries
	onChanged func(depotID uuid.UUID)
}

func NewHandler(pool *pgxpool.Pool, onChanged func(depotID uuid.UUID)) *Handler {
	if onChanged == nil {
		onChanged = func(uuid.UUID) {}
	}
	return &Handler{pool: pool, q: sqlc.New(pool), onChanged: onChanged}
}

func (h *Handler) Register(r fiber.Router, guard fiber.Handler) {
	r.Get("/products", guard, h.list)
	r.Post("/products", guard, h.create)
	r.Patch("/products/:id", guard, h.update)
}

func (h *Handler) list(c fiber.Ctx) error {
	p, _ := httpx.CurrentPrincipal(c)
	products, err := h.q.ListProducts(c.Context(), p.DepotID)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("daftar produk: %w", err))
	}
	return httpx.OK(c, ToViews(products))
}

type createRequest struct {
	Name      string `json:"name" validate:"required,min=2,max=60"`
	Kind      string `json:"kind" validate:"required,oneof=refill new_gallon other"`
	Price     int64  `json:"price" validate:"gte=0,lte=100000000"`
	SortOrder int32  `json:"sort_order" validate:"gte=0,lte=1000"`
}

func (h *Handler) create(c fiber.Ctx) error {
	var req createRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	created, err := h.q.CreateProduct(c.Context(), sqlc.CreateProductParams{
		ID: idgen.NewID(), DepotID: p.DepotID, Name: req.Name, Kind: req.Kind, Price: req.Price, SortOrder: req.SortOrder,
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return httpx.Fail(c, httpx.Validation(map[string]string{"name": "Nama produk sudah dipakai."}))
		}
		return httpx.Fail(c, fmt.Errorf("buat produk: %w", err))
	}
	h.onChanged(p.DepotID)
	return httpx.Created(c, ToView(created))
}

type updateRequest struct {
	Name      *string `json:"name" validate:"omitempty,min=2,max=60"`
	Kind      *string `json:"kind" validate:"omitempty,oneof=refill new_gallon other"`
	Price     *int64  `json:"price" validate:"omitempty,gte=0,lte=100000000"`
	IsActive  *bool   `json:"is_active"`
	SortOrder *int32  `json:"sort_order" validate:"omitempty,gte=0,lte=1000"`
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
	before, err := h.q.GetProduct(c.Context(), sqlc.GetProductParams{ID: id, DepotID: p.DepotID})
	if err != nil {
		if db.IsNoRows(err) {
			return httpx.Fail(c, httpx.NotFound(""))
		}
		return httpx.Fail(c, fmt.Errorf("cari produk: %w", err))
	}
	var updated sqlc.Product
	err = db.WithTx(c.Context(), h.pool, func(tx pgx.Tx) error {
		q := h.q.WithTx(tx)
		updated, err = q.UpdateProduct(c.Context(), sqlc.UpdateProductParams{
			ID: id, DepotID: p.DepotID, Name: req.Name, Price: req.Price, Kind: req.Kind, IsActive: req.IsActive, SortOrder: req.SortOrder,
		})
		if err != nil {
			return fmt.Errorf("ubah produk: %w", err)
		}
		if req.Price != nil && *req.Price != before.Price {
			return audit.Record(c.Context(), q, audit.Entry{
				DepotID: p.DepotID, UserID: &p.UserID, Action: audit.ActionPriceChanged, EntityType: "product", EntityID: &id,
				Meta: map[string]any{"from": before.Price, "to": *req.Price}, IP: httpx.ClientIP(c),
			})
		}
		return nil
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return httpx.Fail(c, httpx.Validation(map[string]string{"name": "Nama produk sudah dipakai."}))
		}
		return httpx.Fail(c, err)
	}
	h.onChanged(p.DepotID)
	return httpx.OK(c, ToView(updated))
}
