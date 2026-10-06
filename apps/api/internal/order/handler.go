package order

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

const (
	HeaderIdempotencyKey = "Idempotency-Key"
	maxIdempotencyKey    = 100
	roleCourier          = "courier"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r fiber.Router, ownerOnly, loggedIn fiber.Handler) {
	r.Get("/orders", ownerOnly, h.list)
	r.Post("/orders", ownerOnly, h.create)
	r.Get("/orders/:id", loggedIn, h.get)
	r.Post("/orders/:id/confirm", ownerOnly, h.confirm)
	r.Post("/orders/:id/assign", ownerOnly, h.assign)
	r.Post("/orders/:id/dispatch", loggedIn, h.dispatch)
	r.Post("/orders/:id/cancel", ownerOnly, h.cancel)
	r.Post("/orders/:id/mark-paid", ownerOnly, h.markPaid)
}

func actorFrom(c fiber.Ctx) Actor {
	p, _ := httpx.CurrentPrincipal(c)
	id := p.UserID
	return Actor{Type: ActorUser, UserID: &id, Role: p.Role, IP: httpx.ClientIP(c)}
}

func IdempotencyKey(c fiber.Ctx) (string, error) {
	key := c.Get(HeaderIdempotencyKey)
	if len(key) > maxIdempotencyKey {
		return "", httpx.ValidationMessage("Idempotency-Key terlalu panjang.")
	}
	return key, nil
}

func (h *Handler) list(c fiber.Ctx) error {
	p, _ := httpx.CurrentPrincipal(c)
	page, err := httpx.ParsePage(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	params := sqlc.ListOrdersParams{DepotID: p.DepotID, Limit: int32(page.Limit) + 1}
	if status := c.Query("status"); status != "" {
		if !IsActive(status) && !IsFinal(status) {
			return httpx.Fail(c, httpx.Validation(map[string]string{"status": "Pilihan tidak tersedia."}))
		}
		params.Status = &status
	}
	if raw := c.Query("date"); raw != "" {
		date, err := time.Parse(time.DateOnly, raw)
		if err != nil {
			return httpx.Fail(c, httpx.Validation(map[string]string{"date": "Format tanggal harus YYYY-MM-DD."}))
		}
		params.ScheduledDate = &date
	}
	if raw := c.Query("courier_id"); raw != "" {
		courierID, err := uuid.Parse(raw)
		if err != nil {
			return httpx.Fail(c, httpx.Validation(map[string]string{"courier_id": "Format ID tidak valid."}))
		}
		params.CourierID = &courierID
	}
	if page.HasCursor {
		params.BeforeAt = &page.Cursor.At
		params.BeforeID = &page.Cursor.ID
	}
	orders, err := h.svc.q.ListOrders(c.Context(), params)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("daftar pesanan: %w", err))
	}
	next := ""
	if len(orders) > page.Limit {
		orders = orders[:page.Limit]
		last := orders[len(orders)-1]
		next = httpx.EncodeCursor(last.CreatedAt, last.ID)
	}
	views, err := h.views(c, orders)
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.List(c, views, next)
}

func (h *Handler) views(c fiber.Ctx, orders []sqlc.Order) ([]View, error) {
	p, _ := httpx.CurrentPrincipal(c)
	items, err := h.svc.ItemsFor(c.Context(), orders)
	if err != nil {
		return nil, err
	}
	names, err := h.svc.CourierNames(c.Context(), p.DepotID)
	if err != nil {
		return nil, err
	}
	views := make([]View, 0, len(orders))
	for _, o := range orders {
		views = append(views, ToView(o, items[o.ID], courierName(names, o.CourierID)))
	}
	return views, nil
}

func courierName(names map[uuid.UUID]string, id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	name, ok := names[*id]
	if !ok {
		return nil
	}
	return &name
}

type itemRequest struct {
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Qty       int32     `json:"qty" validate:"required,gte=1,lte=50"`
}

type createRequest struct {
	CustomerID    uuid.UUID     `json:"customer_id" validate:"required"`
	Items         []itemRequest `json:"items" validate:"omitempty,max=10,dive"`
	RefillQty     int32         `json:"refill_qty" validate:"omitempty,gte=1,lte=50"`
	Fulfilment    string        `json:"fulfilment" validate:"omitempty,oneof=delivery pickup"`
	ScheduledDate string        `json:"scheduled_date" validate:"omitempty,datetime=2006-01-02"`
	Note          string        `json:"note" validate:"max=300"`
}

func (h *Handler) create(c fiber.Ctx) error {
	var req createRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	idemKey, err := IdempotencyKey(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	in := CreateInput{
		DepotID: p.DepotID, CustomerID: req.CustomerID, RefillQty: req.RefillQty, Fulfilment: req.Fulfilment, Note: req.Note,
		Source: SourceOwner, Status: StatusConfirmed, Actor: actorFrom(c), IdempotencyKey: idemKey,
	}
	if in.Fulfilment == "" {
		in.Fulfilment = FulfilmentDelivery
	}
	for _, it := range req.Items {
		in.Items = append(in.Items, ItemInput(it))
	}
	if req.ScheduledDate != "" {
		date, _ := time.Parse(time.DateOnly, req.ScheduledDate)
		in.ScheduledDate = &date
	}
	detail, created, err := h.svc.Create(c.Context(), in)
	if err != nil {
		return httpx.Fail(c, mapCreateError(err))
	}
	view := ToView(detail.Order, detail.Items, nil)
	if created {
		return httpx.Created(c, view)
	}
	return httpx.OK(c, view)
}

func mapCreateError(err error) error {
	switch {
	case errors.Is(err, ErrCustomerNotFound):
		return httpx.Validation(map[string]string{"customer_id": "Pelanggan tidak ditemukan."})
	case errors.Is(err, ErrProductNotFound):
		return httpx.Validation(map[string]string{"items": "Produk tidak ditemukan atau tidak aktif."})
	case errors.Is(err, ErrNoRefillProduct):
		return httpx.Validation(map[string]string{"items": "Depot belum punya produk isi ulang aktif."})
	case errors.Is(err, ErrScheduleRange):
		return httpx.Validation(map[string]string{"scheduled_date": "Tanggal antar harus hari ini sampai 7 hari ke depan."})
	case errors.Is(err, ErrEmptyOrder), errors.Is(err, ErrQtyOutOfRange):
		return httpx.Validation(map[string]string{"items": "Jumlah harus 1 sampai 50."})
	}
	return err
}

func (h *Handler) get(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound("Pesanan tidak ditemukan."))
	}
	p, _ := httpx.CurrentPrincipal(c)
	detail, err := h.svc.Load(c.Context(), p.DepotID, id)
	if err != nil {
		return httpx.Fail(c, err)
	}
	if p.Role == roleCourier {
		if detail.Order.CourierID == nil || *detail.Order.CourierID != p.UserID {
			return httpx.Fail(c, httpx.NotFound("Pesanan tidak ditemukan."))
		}
		return httpx.OK(c, ToCourierView(detail.Order, detail.Items, detail.Customer))
	}
	names, err := h.svc.CourierNames(c.Context(), p.DepotID)
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, ToView(detail.Order, detail.Items, courierName(names, detail.Order.CourierID)))
}

func (h *Handler) confirm(c fiber.Ctx) error {
	return h.respondTransition(c, func(id uuid.UUID, p httpx.Principal) (Detail, error) {
		return h.svc.Confirm(c.Context(), p.DepotID, id, actorFrom(c))
	})
}

type assignRequest struct {
	CourierID uuid.UUID `json:"courier_id" validate:"required"`
}

func (h *Handler) assign(c fiber.Ctx) error {
	var req assignRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	return h.respondTransition(c, func(id uuid.UUID, p httpx.Principal) (Detail, error) {
		detail, err := h.svc.Assign(c.Context(), p.DepotID, id, req.CourierID, actorFrom(c))
		if errors.Is(err, ErrCourierNotFound) {
			return Detail{}, httpx.Validation(map[string]string{"courier_id": "Kurir tidak ditemukan atau tidak aktif."})
		}
		return detail, err
	})
}

func (h *Handler) dispatch(c fiber.Ctx) error {
	idemKey, err := IdempotencyKey(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	return h.respondTransition(c, func(id uuid.UUID, p httpx.Principal) (Detail, error) {
		detail, err := h.svc.Dispatch(c.Context(), p.DepotID, id, actorFrom(c), idemKey)
		if errors.Is(err, ErrNotAssigned) {
			return Detail{}, httpx.NotFound("Pesanan tidak ditemukan.")
		}
		return detail, err
	})
}

type cancelRequest struct {
	Reason string `json:"reason" validate:"required,min=3,max=200"`
}

func (h *Handler) cancel(c fiber.Ctx) error {
	var req cancelRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	return h.respondTransition(c, func(id uuid.UUID, p httpx.Principal) (Detail, error) {
		return h.svc.Cancel(c.Context(), p.DepotID, id, req.Reason, actorFrom(c))
	})
}

type markPaidRequest struct {
	PaymentMethod *string `json:"payment_method" validate:"omitempty,oneof=cash transfer qris"`
}

func (h *Handler) markPaid(c fiber.Ctx) error {
	var req markPaidRequest
	if len(c.Body()) > 0 {
		if err := httpx.DecodeAndValidate(c, &req); err != nil {
			return httpx.Fail(c, err)
		}
	}
	return h.respondTransition(c, func(id uuid.UUID, p httpx.Principal) (Detail, error) {
		detail, err := h.svc.MarkPaid(c.Context(), p.DepotID, id, req.PaymentMethod, actorFrom(c))
		var apiErr *httpx.Error
		if errors.As(err, &apiErr) && apiErr.Code == httpx.CodeInvalidTransition {
			return Detail{}, httpx.InvalidTransition("Pesanan sudah lunas.")
		}
		return detail, err
	})
}

func (h *Handler) respondTransition(c fiber.Ctx, apply func(id uuid.UUID, p httpx.Principal) (Detail, error)) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound("Pesanan tidak ditemukan."))
	}
	p, _ := httpx.CurrentPrincipal(c)
	detail, err := apply(id, p)
	if err != nil {
		return httpx.Fail(c, err)
	}
	if p.Role == roleCourier {
		return httpx.OK(c, ToCourierView(detail.Order, detail.Items, detail.Customer))
	}
	names, err := h.svc.CourierNames(c.Context(), p.DepotID)
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, ToView(detail.Order, detail.Items, courierName(names, detail.Order.CourierID)))
}
