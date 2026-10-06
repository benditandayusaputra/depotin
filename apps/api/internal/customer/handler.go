package customer

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/audit"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/phone"
)

const (
	SourceOwner  = "owner"
	SourcePublic = "public"

	searchLimit   = 20
	idleLoanDays  = 30
	maxSnoozeDays = 30
)

type Handler struct {
	pool  *pgxpool.Pool
	q     *sqlc.Queries
	clock clock.Clock
	links *Links
}

func NewHandler(pool *pgxpool.Pool, clk clock.Clock, links *Links) *Handler {
	return &Handler{pool: pool, q: sqlc.New(pool), clock: clk, links: links}
}

func (h *Handler) Register(r fiber.Router, guard fiber.Handler) {
	r.Get("/customers", guard, h.list)
	r.Post("/customers", guard, h.create)
	r.Get("/customers/:id", guard, h.get)
	r.Patch("/customers/:id", guard, h.update)
	r.Post("/customers/:id/link", guard, h.link)
	r.Post("/customers/:id/link/rotate", guard, h.rotateLink)
	r.Post("/customers/:id/snooze", guard, h.snooze)
}

func (h *Handler) list(c fiber.Ctx) error {
	p, _ := httpx.CurrentPrincipal(c)
	ctx := c.Context()
	if q := c.Query("q"); q != "" {
		if len(q) > 60 {
			return httpx.Fail(c, httpx.Validation(map[string]string{"q": "Maksimal 60 karakter."}))
		}
		rows, err := h.q.SearchCustomers(ctx, sqlc.SearchCustomersParams{DepotID: p.DepotID, Limit: searchLimit, Q: q})
		if err != nil {
			return httpx.Fail(c, fmt.Errorf("cari pelanggan: %w", err))
		}
		return httpx.List(c, ToViews(rows), "")
	}
	page, err := httpx.ParsePage(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	now := h.clock.Now()
	limit := int32(page.Limit)
	var rows []sqlc.Customer
	switch filter := c.Query("filter", "all"); filter {
	case "all":
		params := sqlc.ListCustomersParams{DepotID: p.DepotID, Limit: limit + 1}
		if page.HasCursor {
			params.BeforeAt = &page.Cursor.At
			params.BeforeID = &page.Cursor.ID
		}
		rows, err = h.q.ListCustomers(ctx, params)
		if err == nil && len(rows) > page.Limit {
			last := rows[page.Limit-1]
			return httpx.List(c, ToViews(rows[:page.Limit]), httpx.EncodeCursor(last.CreatedAt, last.ID))
		}
	case "due":
		depot, derr := h.q.GetDepot(ctx, p.DepotID)
		if derr != nil {
			return httpx.Fail(c, fmt.Errorf("ambil depot: %w", derr))
		}
		dueBefore := now.Add(time.Duration(depot.ReminderLeadDays) * 24 * time.Hour)
		rows, err = h.q.ListDueCustomers(ctx, sqlc.ListDueCustomersParams{DepotID: p.DepotID, Limit: httpx.MaxLimit, DueBefore: dueBefore})
	case "at_risk":
		rows, err = h.q.ListAtRiskCustomers(ctx, sqlc.ListAtRiskCustomersParams{DepotID: p.DepotID, Limit: httpx.MaxLimit, Now: now})
	case "loan":
		rows, err = h.q.ListLoanCustomers(ctx, sqlc.ListLoanCustomersParams{DepotID: p.DepotID, Limit: httpx.MaxLimit})
	default:
		return httpx.Fail(c, httpx.Validation(map[string]string{"filter": "Pilihan tidak tersedia."}))
	}
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("daftar pelanggan: %w", err))
	}
	return httpx.List(c, ToViews(rows), "")
}

type createRequest struct {
	Name        string   `json:"name" validate:"required,min=1,max=80"`
	Phone       string   `json:"phone" validate:"required,min=9,max=20"`
	Address     string   `json:"address" validate:"max=300"`
	AddressNote string   `json:"address_note" validate:"max=200"`
	Area        string   `json:"area" validate:"max=60"`
	Lat         *float64 `json:"lat" validate:"omitempty,latitude"`
	Lng         *float64 `json:"lng" validate:"omitempty,longitude"`
	UsualQty    int32    `json:"usual_qty" validate:"omitempty,gte=1,lte=50"`
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
	if req.UsualQty == 0 {
		req.UsualQty = 1
	}
	p, _ := httpx.CurrentPrincipal(c)
	created, err := h.q.CreateCustomer(c.Context(), sqlc.CreateCustomerParams{
		ID: idgen.NewID(), DepotID: p.DepotID, Name: req.Name, Phone: normalized, Address: req.Address,
		AddressNote: req.AddressNote, Area: req.Area, Lat: req.Lat, Lng: req.Lng, Source: SourceOwner, IsVerified: true, UsualQty: req.UsualQty,
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return httpx.Fail(c, httpx.Validation(map[string]string{"phone": "Nomor ini sudah terdaftar di depot."}))
		}
		return httpx.Fail(c, fmt.Errorf("buat pelanggan: %w", err))
	}
	return httpx.Created(c, ToView(created))
}

func (h *Handler) get(c fiber.Ctx) error {
	cust, err := h.load(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, ToView(cust))
}

type updateRequest struct {
	Name        *string  `json:"name" validate:"omitempty,min=1,max=80"`
	Phone       *string  `json:"phone" validate:"omitempty,min=9,max=20"`
	Address     *string  `json:"address" validate:"omitempty,max=300"`
	AddressNote *string  `json:"address_note" validate:"omitempty,max=200"`
	Area        *string  `json:"area" validate:"omitempty,max=60"`
	Lat         *float64 `json:"lat" validate:"omitempty,latitude"`
	Lng         *float64 `json:"lng" validate:"omitempty,longitude"`
	UsualQty    *int32   `json:"usual_qty" validate:"omitempty,gte=1,lte=50"`
	IsActive    *bool    `json:"is_active"`
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
	if req.Phone != nil {
		normalized, err := phone.Normalize(*req.Phone)
		if err != nil {
			return httpx.Fail(c, httpx.Validation(map[string]string{"phone": "Nomor HP tidak valid."}))
		}
		req.Phone = &normalized
	}
	p, _ := httpx.CurrentPrincipal(c)
	updated, err := h.q.UpdateCustomer(c.Context(), sqlc.UpdateCustomerParams{
		ID: id, DepotID: p.DepotID, Name: req.Name, Phone: req.Phone, Address: req.Address, AddressNote: req.AddressNote,
		Area: req.Area, Lat: req.Lat, Lng: req.Lng, UsualQty: req.UsualQty, IsActive: req.IsActive,
	})
	if err != nil {
		switch {
		case db.IsNoRows(err):
			return httpx.Fail(c, httpx.NotFound(""))
		case db.IsUniqueViolation(err):
			return httpx.Fail(c, httpx.Validation(map[string]string{"phone": "Nomor ini sudah terdaftar di depot."}))
		}
		return httpx.Fail(c, fmt.Errorf("ubah pelanggan: %w", err))
	}
	return httpx.OK(c, ToView(updated))
}

type linkResponse struct {
	Link  string `json:"link"`
	WaURL string `json:"wa_url"`
}

func (h *Handler) link(c fiber.Ctx) error {
	cust, err := h.load(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	depot, err := h.q.GetDepot(c.Context(), cust.DepotID)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("ambil depot: %w", err))
	}
	var token string
	err = db.WithTx(c.Context(), h.pool, func(tx pgx.Tx) error {
		token, err = h.links.EnsureToken(c.Context(), h.q.WithTx(tx), cust, h.clock.Now())
		return err
	})
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, h.linkResponse(cust, depot, token))
}

func (h *Handler) rotateLink(c fiber.Ctx) error {
	cust, err := h.load(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	depot, err := h.q.GetDepot(c.Context(), cust.DepotID)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("ambil depot: %w", err))
	}
	p, _ := httpx.CurrentPrincipal(c)
	var token string
	err = db.WithTx(c.Context(), h.pool, func(tx pgx.Tx) error {
		q := h.q.WithTx(tx)
		token, err = h.links.Issue(c.Context(), q, cust.DepotID, cust.ID, h.clock.Now())
		if err != nil {
			return err
		}
		return audit.Record(c.Context(), q, audit.Entry{
			DepotID: p.DepotID, UserID: &p.UserID, Action: audit.ActionLinkRotated, EntityType: "customer", EntityID: &cust.ID, IP: httpx.ClientIP(c),
		})
	})
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, h.linkResponse(cust, depot, token))
}

func (h *Handler) linkResponse(cust sqlc.Customer, depot sqlc.Depot, token string) linkResponse {
	link := h.links.PersonalURL(token)
	return linkResponse{Link: link, WaURL: WhatsAppURL(cust.Phone, ShareMessage(cust.Name, depot.Name, link))}
}

type snoozeRequest struct {
	Days int `json:"days" validate:"required,gte=1,lte=30"`
}

func (h *Handler) snooze(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound(""))
	}
	var req snoozeRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	until := h.clock.Now().AddDate(0, 0, req.Days)
	rows, err := h.q.SnoozeCustomer(c.Context(), sqlc.SnoozeCustomerParams{ID: id, DepotID: p.DepotID, ReminderSnoozedUntil: &until})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("tunda pengingat: %w", err))
	}
	if rows == 0 {
		return httpx.Fail(c, httpx.NotFound(""))
	}
	return httpx.OK(c, fiber.Map{"snoozed_until": until.Format(time.DateOnly)})
}

func (h *Handler) load(c fiber.Ctx) (sqlc.Customer, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return sqlc.Customer{}, httpx.NotFound("")
	}
	p, _ := httpx.CurrentPrincipal(c)
	cust, err := h.q.GetCustomer(c.Context(), sqlc.GetCustomerParams{ID: id, DepotID: p.DepotID})
	if err != nil {
		if db.IsNoRows(err) {
			return sqlc.Customer{}, httpx.NotFound("")
		}
		return sqlc.Customer{}, fmt.Errorf("cari pelanggan: %w", err)
	}
	return cust, nil
}
