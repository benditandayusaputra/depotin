package stream

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/sse"
	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/ratelimit"
)

const (
	heartbeat       = 20 * time.Second
	maxTicketLength = 64
)

var (
	ticketRule = ratelimit.Rule{Limit: 20, Window: time.Minute}
	streamRule = ratelimit.Rule{Limit: 30, Window: time.Minute}
)

type Handler struct {
	hub       *Hub
	tickets   *TicketStore
	limiter   *ratelimit.Limiter
	webOrigin string
}

func NewHandler(hub *Hub, tickets *TicketStore, limiter *ratelimit.Limiter, webOrigin string) *Handler {
	return &Handler{hub: hub, tickets: tickets, limiter: limiter, webOrigin: webOrigin}
}

func (h *Handler) Register(r fiber.Router, loggedIn fiber.Handler) {
	r.Post("/stream/tickets", loggedIn, h.issueTicket)
	r.Get("/stream", httpx.RateLimitByIP(h.limiter, streamRule, "stream"), h.authorize, sse.New(sse.Config{
		Handler:           h.serve,
		HeartbeatInterval: heartbeat,
		Retry:             5 * time.Second,
	}))
}

func (h *Handler) issueTicket(c fiber.Ctx) error {
	p, _ := httpx.CurrentPrincipal(c)
	if err := httpx.CheckLimit(c, h.limiter, ticketRule, "ticket:"+p.UserID.String()); err != nil {
		return httpx.Fail(c, err)
	}
	raw, err := h.tickets.Issue(p)
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, fiber.Map{"ticket": raw, "expires_in": int(TicketTTL.Seconds())})
}

func (h *Handler) authorize(c fiber.Ctx) error {
	c.Set(fiber.HeaderAccessControlAllowOrigin, h.webOrigin)
	c.Set(fiber.HeaderVary, fiber.HeaderOrigin)
	c.Set(fiber.HeaderCacheControl, "no-store")
	raw := c.Query("ticket")
	if raw == "" || len(raw) > maxTicketLength {
		return httpx.Fail(c, httpx.Unauthenticated())
	}
	p, ok := h.tickets.Redeem(raw)
	if !ok {
		return httpx.Fail(c, httpx.Unauthenticated())
	}
	httpx.SetPrincipal(c, p)
	return c.Next()
}

func (h *Handler) serve(c fiber.Ctx, stream *sse.Stream) error {
	p, _ := httpx.CurrentPrincipal(c)
	sub, ok := h.hub.Subscribe(p.UserID, p.DepotID, p.Role)
	if !ok {
		if err := stream.Event(sse.Event{Name: "error", Data: map[string]string{"code": "too_many_connections"}}); err != nil {
			return fmt.Errorf("kirim galat stream: %w", err)
		}
		return nil
	}
	defer h.hub.Unsubscribe(sub)
	if err := stream.Event(sse.Event{Name: "ready", Data: map[string]string{"role": p.Role}}); err != nil {
		return nil
	}
	for {
		select {
		case <-stream.Done():
			return nil
		case e, open := <-sub.Events:
			if !open {
				return nil
			}
			if err := stream.Event(sse.Event{Name: e.Name, Data: e.Data}); err != nil {
				return nil
			}
		}
	}
}

type Publisher struct {
	hub *Hub
}

func NewPublisher(hub *Hub) Publisher {
	return Publisher{hub: hub}
}

type orderSummary struct {
	ID        uuid.UUID  `json:"id"`
	Code      string     `json:"code"`
	Status    string     `json:"status"`
	Source    string     `json:"source"`
	Name      string     `json:"delivery_name"`
	Total     int64      `json:"total"`
	CourierID *uuid.UUID `json:"courier_id"`
}

func summarize(v order.View) orderSummary {
	return orderSummary{ID: v.ID, Code: v.Code, Status: v.Status, Source: v.Source, Name: v.DeliveryName, Total: v.Total, CourierID: v.CourierID}
}

func (p Publisher) OrderCreated(depotID uuid.UUID, v order.View) {
	p.hub.Publish(depotID, Event{Name: EventOrderNew, Data: summarize(v)}, func(s *Subscriber) bool { return s.Role == "owner" })
}

func (p Publisher) OrderUpdated(depotID uuid.UUID, v order.View) {
	data := summarize(v)
	p.hub.Publish(depotID, Event{Name: EventOrderUpd, Data: data}, func(s *Subscriber) bool {
		return s.Role == "owner" || (v.CourierID != nil && *v.CourierID == s.UserID)
	})
}

func (p Publisher) ReminderQueued(depotID uuid.UUID, count int64) {
	p.hub.Publish(depotID, Event{Name: EventReminders, Data: map[string]int64{"count": count}}, func(s *Subscriber) bool { return s.Role == "owner" })
}
