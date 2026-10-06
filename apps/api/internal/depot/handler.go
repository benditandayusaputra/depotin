package depot

import (
	"fmt"
	"regexp"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/phone"
)

var timePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

type Handler struct {
	q         *sqlc.Queries
	onChanged func(depotID string)
}

func NewHandler(pool *pgxpool.Pool, onChanged func(depotID string)) *Handler {
	if onChanged == nil {
		onChanged = func(string) {}
	}
	return &Handler{q: sqlc.New(pool), onChanged: onChanged}
}

func (h *Handler) Register(r fiber.Router) {
	r.Get("/depot", h.get)
	r.Patch("/depot", h.update)
}

func (h *Handler) get(c fiber.Ctx) error {
	p, _ := httpx.CurrentPrincipal(c)
	d, err := h.q.GetDepot(c.Context(), p.DepotID)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("ambil depot: %w", err))
	}
	return httpx.OK(c, ToView(d))
}

type updateRequest struct {
	Name                 *string  `json:"name" validate:"omitempty,min=3,max=80"`
	Phone                *string  `json:"phone" validate:"omitempty,min=9,max=20"`
	Address              *string  `json:"address" validate:"omitempty,max=300"`
	Lat                  *float64 `json:"lat" validate:"omitempty,latitude"`
	Lng                  *float64 `json:"lng" validate:"omitempty,longitude"`
	OpenTime             *string  `json:"open_time"`
	CloseTime            *string  `json:"close_time"`
	DeliveryFee          *int64   `json:"delivery_fee" validate:"omitempty,gte=0,lte=1000000"`
	IsAcceptingOrders    *bool    `json:"is_accepting_orders"`
	AutoConfirmKnown     *bool    `json:"auto_confirm_known"`
	LoyaltyEvery         *int32   `json:"loyalty_every" validate:"omitempty,gte=0,lte=50"`
	ReminderLeadDays     *int32   `json:"reminder_lead_days" validate:"omitempty,gte=0,lte=7"`
	DefaultDaysPerGallon *float64 `json:"default_days_per_gallon" validate:"omitempty,gte=0.5,lte=30"`
}

func (h *Handler) update(c fiber.Ctx) error {
	var req updateRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	fields := map[string]string{}
	if req.Phone != nil {
		normalized, err := phone.Normalize(*req.Phone)
		if err != nil {
			fields["phone"] = "Nomor HP tidak valid."
		} else {
			req.Phone = &normalized
		}
	}
	for name, value := range map[string]*string{"open_time": req.OpenTime, "close_time": req.CloseTime} {
		if value != nil && !timePattern.MatchString(*value) {
			fields[name] = "Format jam harus HH:MM."
		}
	}
	if req.LoyaltyEvery != nil && *req.LoyaltyEvery == 1 {
		fields["loyalty_every"] = "Minimal 2 atau 0 untuk mematikan."
	}
	if len(fields) > 0 {
		return httpx.Fail(c, httpx.Validation(fields))
	}

	p, _ := httpx.CurrentPrincipal(c)
	params := sqlc.UpdateDepotParams{
		ID: p.DepotID, Name: req.Name, Phone: req.Phone, Address: req.Address, Lat: req.Lat, Lng: req.Lng,
		OpenTime: req.OpenTime, CloseTime: req.CloseTime, DeliveryFee: req.DeliveryFee,
		IsAcceptingOrders: req.IsAcceptingOrders, AutoConfirmKnown: req.AutoConfirmKnown,
		ReminderLeadDays: req.ReminderLeadDays, DefaultDaysPerGallon: req.DefaultDaysPerGallon,
	}
	if req.LoyaltyEvery != nil {
		if *req.LoyaltyEvery == 0 {
			params.ClearLoyalty = true
		} else {
			params.LoyaltyEvery = req.LoyaltyEvery
		}
	}
	d, err := h.q.UpdateDepot(c.Context(), params)
	if err != nil {
		if db.IsNoRows(err) {
			return httpx.Fail(c, httpx.NotFound(""))
		}
		return httpx.Fail(c, fmt.Errorf("ubah depot: %w", err))
	}
	h.onChanged(d.Slug)
	return httpx.OK(c, ToView(d))
}
