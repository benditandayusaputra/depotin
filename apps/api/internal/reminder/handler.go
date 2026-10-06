package reminder

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/customer"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

type View struct {
	ID               uuid.UUID     `json:"id"`
	DueDate          string        `json:"due_date"`
	PredictedEmptyAt time.Time     `json:"predicted_empty_at"`
	Status           string        `json:"status"`
	SentAt           *time.Time    `json:"sent_at"`
	OrderID          *uuid.UUID    `json:"order_id"`
	Customer         customer.View `json:"customer"`
}

func toView(r sqlc.Reminder, c sqlc.Customer) View {
	return View{
		ID: r.ID, DueDate: r.DueDate.Format(time.DateOnly), PredictedEmptyAt: r.PredictedEmptyAt, Status: r.Status,
		SentAt: r.SentAt, OrderID: r.OrderID, Customer: customer.ToView(c),
	}
}

type SentView struct {
	Reminder View   `json:"reminder"`
	Link     string `json:"link"`
	WaURL    string `json:"wa_url"`
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r fiber.Router, ownerOnly fiber.Handler) {
	r.Get("/reminders", ownerOnly, h.list)
	r.Post("/reminders/:id/send", ownerOnly, h.send)
	r.Post("/reminders/:id/skip", ownerOnly, h.skip)
}

func (h *Handler) depot(c fiber.Ctx) (sqlc.Depot, error) {
	p, _ := httpx.CurrentPrincipal(c)
	depot, err := h.svc.q.GetDepot(c.Context(), p.DepotID)
	if err != nil {
		return sqlc.Depot{}, fmt.Errorf("ambil depot: %w", err)
	}
	return depot, nil
}

func (h *Handler) list(c fiber.Ctx) error {
	depot, err := h.depot(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	if _, err := h.svc.Refresh(c.Context(), depot); err != nil {
		return httpx.Fail(c, err)
	}
	items, err := h.svc.List(c.Context(), depot)
	if err != nil {
		return httpx.Fail(c, err)
	}
	views := make([]View, 0, len(items))
	for _, it := range items {
		views = append(views, toView(it.Reminder, it.Customer))
	}
	return httpx.OK(c, views)
}

func (h *Handler) send(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound("Pengingat tidak ditemukan."))
	}
	depot, err := h.depot(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	p, _ := httpx.CurrentPrincipal(c)
	sent, err := h.svc.Send(c.Context(), depot, id, p.UserID)
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, SentView{Reminder: toView(sent.Reminder, sent.Customer), Link: sent.Link, WaURL: sent.WaURL})
}

func (h *Handler) skip(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.Fail(c, httpx.NotFound("Pengingat tidak ditemukan."))
	}
	p, _ := httpx.CurrentPrincipal(c)
	r, err := h.svc.Skip(c.Context(), p.DepotID, id)
	if err != nil {
		return httpx.Fail(c, err)
	}
	cust, err := h.svc.q.GetCustomer(c.Context(), sqlc.GetCustomerParams{ID: r.CustomerID, DepotID: p.DepotID})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("ambil pelanggan: %w", err))
	}
	return httpx.OK(c, toView(r, cust))
}
