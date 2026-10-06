package order

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

func (h *Handler) RegisterCourier(r fiber.Router, courierOnly fiber.Handler) {
	r.Get("/courier/queue", courierOnly, h.queue)
	r.Post("/courier/customers/:id/location", courierOnly, h.saveLocation)
}

func (h *Handler) queue(c fiber.Ctx) error {
	p, _ := httpx.CurrentPrincipal(c)
	depot, err := h.svc.q.GetDepot(c.Context(), p.DepotID)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("ambil depot: %w", err))
	}
	today := DateIn(h.svc.clock.Now(), DepotLocation(depot))
	rows, err := h.svc.q.ListCourierQueue(c.Context(), sqlc.ListCourierQueueParams{CourierID: &p.UserID, DepotID: p.DepotID, UntilDate: today})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("antrean kurir: %w", err))
	}
	orders := make([]sqlc.Order, 0, len(rows))
	byID := make(map[uuid.UUID]sqlc.ListCourierQueueRow, len(rows))
	stops := make([]Stop, 0, len(rows))
	for _, r := range rows {
		orders = append(orders, r.Order)
		byID[r.Order.ID] = r
		stops = append(stops, Stop{ID: r.Order.ID.String(), Status: r.Order.Status, Lat: r.Lat, Lng: r.Lng, Area: r.Area, CreatedAt: r.Order.CreatedAt})
	}
	items, err := h.svc.ItemsFor(c.Context(), orders)
	if err != nil {
		return httpx.Fail(c, err)
	}
	views := make([]CourierView, 0, len(rows))
	for _, s := range SortRoute(stops, depot.Lat, depot.Lng) {
		r := byID[uuid.MustParse(s.ID)]
		views = append(views, ToCourierView(r.Order, items[r.Order.ID], sqlc.Customer{Lat: r.Lat, Lng: r.Lng, Area: r.Area, LoanBalance: r.LoanBalance}))
	}
	return httpx.OK(c, views)
}

type locationRequest struct {
	Lat float64 `json:"lat" validate:"required,latitude"`
	Lng float64 `json:"lng" validate:"required,longitude"`
}

func (h *Handler) saveLocation(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound(""))
	}
	var req locationRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	rows, err := h.svc.q.SetCustomerLocation(c.Context(), sqlc.SetCustomerLocationParams{ID: id, DepotID: p.DepotID, Lat: &req.Lat, Lng: &req.Lng})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("simpan lokasi: %w", err))
	}
	if rows == 0 {
		return httpx.Fail(c, httpx.NotFound(""))
	}
	return httpx.OK(c, fiber.Map{"saved": true})
}

type deliverRequest struct {
	GallonsReturned int32  `json:"gallons_returned" validate:"gte=0,lte=100"`
	PaymentMethod   string `json:"payment_method" validate:"omitempty,oneof=cash transfer qris"`
	Paid            bool   `json:"paid"`
}

func (h *Handler) deliver(c fiber.Ctx) error {
	var req deliverRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	if req.Paid && req.PaymentMethod == "" {
		return httpx.Fail(c, httpx.Validation(map[string]string{"payment_method": "Pilih cara bayar."}))
	}
	idemKey, err := IdempotencyKey(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	return h.respondTransition(c, func(id uuid.UUID, p httpx.Principal) (Detail, error) {
		detail, err := h.svc.Deliver(c.Context(), p.DepotID, id, DeliverInput(req), actorFrom(c), idemKey)
		switch {
		case errors.Is(err, ErrNotAssigned):
			return Detail{}, httpx.NotFound("Pesanan tidak ditemukan.")
		case errors.Is(err, ErrTooManyReturned):
			return Detail{}, httpx.Validation(map[string]string{"gallons_returned": "Galon kosong melebihi yang dipegang pelanggan."})
		}
		return detail, err
	})
}

func (h *Handler) customerOrders(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound(""))
	}
	page, err := httpx.ParsePage(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	params := sqlc.ListOrdersByCustomerParams{CustomerID: id, DepotID: p.DepotID, Limit: int32(page.Limit) + 1}
	if page.HasCursor {
		params.BeforeAt = &page.Cursor.At
		params.BeforeID = &page.Cursor.ID
	}
	orders, err := h.svc.q.ListOrdersByCustomer(c.Context(), params)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("pesanan pelanggan: %w", err))
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
