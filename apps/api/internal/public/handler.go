package public

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/customer"
	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/cache"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/crypto"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/phone"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/ratelimit"
	"github.com/benditandayusaputra/depotin/apps/api/internal/product"
)

const (
	depotCacheTTL      = 30 * time.Second
	invalidateTimeout  = 2 * time.Second
	maxPendingPerPhone = 3
	recentOrdersLimit  = 5
	maxTokenLength     = 64
	base62Alphabet     = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

var (
	publicOrderIPRule    = ratelimit.Rule{Limit: 5, Window: 10 * time.Minute}
	publicOrderPhoneRule = ratelimit.Rule{Limit: 3, Window: time.Hour}
	tokenReadRule        = ratelimit.Rule{Limit: 60, Window: time.Minute}
)

type cachedPage struct {
	body []byte
	etag string
}

type Handler struct {
	orders  *order.Service
	q       *sqlc.Queries
	clock   clock.Clock
	limiter *ratelimit.Limiter
	pages   *cache.TTL[cachedPage]
}

func NewHandler(orders *order.Service, clk clock.Clock, limiter *ratelimit.Limiter) *Handler {
	return &Handler{orders: orders, q: orders.Queries(), clock: clk, limiter: limiter, pages: cache.New[cachedPage](clk, depotCacheTTL)}
}

func (h *Handler) InvalidateDepot(slug string) {
	h.pages.Delete(slug)
}

func (h *Handler) InvalidateDepotByID(depotID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), invalidateTimeout)
	defer cancel()
	depot, err := h.q.GetDepot(ctx, depotID)
	if err != nil {
		return
	}
	h.pages.Delete(depot.Slug)
}

func (h *Handler) Register(r fiber.Router) {
	r.Get("/public/depots/:slug", h.depotPage)
	r.Post("/public/depots/:slug/orders", httpx.RateLimitByIP(h.limiter, publicOrderIPRule, "public-order"), h.publicOrder)
	r.Get("/public/track/:token", httpx.RateLimitByIP(h.limiter, tokenReadRule, "token-read"), h.track)
	r.Post("/public/track/:token/cancel", httpx.RateLimitByIP(h.limiter, tokenReadRule, "token-read"), h.cancelTracked)
	r.Get("/public/me/:token", httpx.RateLimitByIP(h.limiter, tokenReadRule, "token-read"), h.me)
	r.Post("/public/me/:token/orders", httpx.RateLimitByIP(h.limiter, publicOrderIPRule, "public-order"), h.reorder)
}

func (h *Handler) depotPage(c fiber.Ctx) error {
	slug := c.Params("slug")
	page, ok := h.pages.Get(slug)
	if !ok {
		depot, err := h.q.GetDepotBySlug(c.Context(), slug)
		if err != nil {
			if db.IsNoRows(err) {
				return httpx.Fail(c, httpx.NotFound("Depot tidak ditemukan."))
			}
			return httpx.Fail(c, fmt.Errorf("ambil depot: %w", err))
		}
		products, err := h.q.ListActiveProducts(c.Context(), depot.ID)
		if err != nil {
			return httpx.Fail(c, fmt.Errorf("daftar produk: %w", err))
		}
		body, err := json.Marshal(map[string]any{"data": DepotPage{Depot: depotView(depot), Products: product.ToViews(products)}})
		if err != nil {
			return httpx.Fail(c, fmt.Errorf("encode halaman depot: %w", err))
		}
		sum := sha256.Sum256(body)
		page = cachedPage{body: body, etag: `"` + hex.EncodeToString(sum[:8]) + `"`}
		h.pages.Set(slug, page)
	}
	c.Set(fiber.HeaderETag, page.etag)
	c.Set(fiber.HeaderCacheControl, "public, max-age=30, stale-while-revalidate=300")
	if strings.Contains(c.Get(fiber.HeaderIfNoneMatch), page.etag) {
		return c.SendStatus(fiber.StatusNotModified)
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
	if err := c.Send(page.body); err != nil {
		return fmt.Errorf("kirim halaman depot: %w", err)
	}
	return nil
}

type publicOrderRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=80"`
	Phone       string `json:"phone" validate:"required,min=9,max=20"`
	Address     string `json:"address" validate:"required,min=5,max=300"`
	AddressNote string `json:"address_note" validate:"max=200"`
	Qty         int32  `json:"qty" validate:"required,gte=1,lte=50"`
	Note        string `json:"note" validate:"max=300"`
	Website     string `json:"website" validate:"max=0"`
}

func (h *Handler) publicOrder(c fiber.Ctx) error {
	var req publicOrderRequest
	if err := httpx.DecodeAndValidate(c, &req); err != nil {
		return httpx.Fail(c, err)
	}
	normalized, err := phone.Normalize(req.Phone)
	if err != nil {
		return httpx.Fail(c, httpx.Validation(map[string]string{"phone": "Nomor WhatsApp tidak valid."}))
	}
	depot, err := h.q.GetDepotBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		if db.IsNoRows(err) {
			return httpx.Fail(c, httpx.NotFound("Depot tidak ditemukan."))
		}
		return httpx.Fail(c, fmt.Errorf("ambil depot: %w", err))
	}
	if !depot.IsAcceptingOrders {
		return httpx.Fail(c, httpx.Conflict("Depot sedang tidak menerima pesanan."))
	}
	if err := httpx.CheckLimit(c, h.limiter, publicOrderPhoneRule, "public-order-phone:"+depot.ID.String()+":"+normalized); err != nil {
		return httpx.Fail(c, err)
	}
	pending, err := h.q.CountPendingOrdersByPhone(c.Context(), sqlc.CountPendingOrdersByPhoneParams{DepotID: depot.ID, DeliveryPhone: normalized})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("hitung pesanan menunggu: %w", err))
	}
	if pending >= maxPendingPerPhone {
		return httpx.Fail(c, httpx.Conflict("Masih ada pesanan yang menunggu konfirmasi untuk nomor ini."))
	}
	idemKey, err := order.IdempotencyKey(c)
	if err != nil {
		return httpx.Fail(c, err)
	}

	cust, err := h.findOrCreateCustomer(c, depot.ID, req, normalized)
	if err != nil {
		return httpx.Fail(c, err)
	}
	detail, created, err := h.orders.Create(c.Context(), order.CreateInput{
		DepotID: depot.ID, CustomerID: cust.ID, RefillQty: req.Qty, Fulfilment: order.FulfilmentDelivery, Note: req.Note,
		Source: order.SourcePublic, Status: order.StatusPending, Actor: order.Actor{Type: order.ActorCustomer, IP: httpx.ClientIP(c)},
		IdempotencyKey: idemKey, Delivery: &order.DeliveryCopy{Name: req.Name, Phone: normalized, Address: req.Address, Note: req.AddressNote},
	})
	if err != nil {
		return httpx.Fail(c, mapPublicCreateError(err))
	}
	return h.respondCreated(c, detail, created)
}

func (h *Handler) findOrCreateCustomer(c fiber.Ctx, depotID uuid.UUID, req publicOrderRequest, normalized string) (sqlc.Customer, error) {
	cust, err := h.q.GetCustomerByPhone(c.Context(), sqlc.GetCustomerByPhoneParams{DepotID: depotID, Phone: normalized})
	if err == nil {
		return cust, nil
	}
	if !db.IsNoRows(err) {
		return sqlc.Customer{}, fmt.Errorf("cari pelanggan: %w", err)
	}
	cust, err = h.q.CreateCustomer(c.Context(), sqlc.CreateCustomerParams{
		ID: idgen.NewID(), DepotID: depotID, Name: req.Name, Phone: normalized, Address: req.Address, AddressNote: req.AddressNote,
		Source: customer.SourcePublic, IsVerified: false, UsualQty: req.Qty,
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			existing, ferr := h.q.GetCustomerByPhone(c.Context(), sqlc.GetCustomerByPhoneParams{DepotID: depotID, Phone: normalized})
			if ferr != nil {
				return sqlc.Customer{}, fmt.Errorf("cari pelanggan: %w", ferr)
			}
			return existing, nil
		}
		return sqlc.Customer{}, fmt.Errorf("buat pelanggan: %w", err)
	}
	return cust, nil
}

func mapPublicCreateError(err error) error {
	switch {
	case errors.Is(err, order.ErrNoRefillProduct):
		return httpx.Conflict("Depot belum menyiapkan produk isi ulang.")
	case errors.Is(err, order.ErrQtyOutOfRange), errors.Is(err, order.ErrEmptyOrder):
		return httpx.Validation(map[string]string{"qty": "Jumlah harus 1 sampai 50."})
	case errors.Is(err, order.ErrScheduleRange):
		return httpx.Validation(map[string]string{"scheduled_date": "Pilih hari ini atau besok."})
	}
	return err
}

func (h *Handler) respondCreated(c fiber.Ctx, detail order.Detail, created bool) error {
	body := CreatedOrder{Code: detail.Order.Code, Status: detail.Order.Status, TrackToken: detail.Token}
	if created {
		return httpx.Created(c, body)
	}
	return httpx.OK(c, body)
}

func parseToken(c fiber.Ctx) (string, bool) {
	token := c.Params("token")
	if token == "" || len(token) > maxTokenLength {
		return "", false
	}
	if strings.Trim(token, base62Alphabet) != "" {
		return "", false
	}
	return token, true
}

func (h *Handler) loadTracked(c fiber.Ctx) (sqlc.Order, sqlc.Depot, error) {
	token, ok := parseToken(c)
	if !ok {
		return sqlc.Order{}, sqlc.Depot{}, httpx.NotFound("Link tidak dikenal.")
	}
	o, err := h.q.GetOrderByTrackTokenHash(c.Context(), crypto.HashToken(token))
	if err != nil {
		if db.IsNoRows(err) {
			return sqlc.Order{}, sqlc.Depot{}, httpx.NotFound("Link tidak dikenal.")
		}
		return sqlc.Order{}, sqlc.Depot{}, fmt.Errorf("cari pesanan: %w", err)
	}
	depot, err := h.q.GetDepot(c.Context(), o.DepotID)
	if err != nil {
		return sqlc.Order{}, sqlc.Depot{}, fmt.Errorf("ambil depot: %w", err)
	}
	return o, depot, nil
}

func noStore(c fiber.Ctx) {
	c.Set(fiber.HeaderCacheControl, "no-store")
}

func (h *Handler) track(c fiber.Ctx) error {
	noStore(c)
	o, depot, err := h.loadTracked(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	items, err := h.q.ListOrderItems(c.Context(), []uuid.UUID{o.ID})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("butir pesanan: %w", err))
	}
	return httpx.OK(c, trackView(o, items, depot))
}

func (h *Handler) cancelTracked(c fiber.Ctx) error {
	noStore(c)
	o, depot, err := h.loadTracked(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	detail, err := h.orders.CancelByCustomer(c.Context(), o.DepotID, o.ID, httpx.ClientIP(c))
	if err != nil {
		return httpx.Fail(c, err)
	}
	return httpx.OK(c, trackView(detail.Order, detail.Items, depot))
}

func (h *Handler) loadMe(c fiber.Ctx) (sqlc.Customer, sqlc.Depot, error) {
	token, ok := parseToken(c)
	if !ok {
		return sqlc.Customer{}, sqlc.Depot{}, httpx.NotFound("Link tidak dikenal.")
	}
	cust, err := h.q.GetCustomerByTokenHash(c.Context(), crypto.HashToken(token))
	if err != nil {
		if db.IsNoRows(err) {
			return sqlc.Customer{}, sqlc.Depot{}, httpx.NotFound("Link tidak dikenal.")
		}
		return sqlc.Customer{}, sqlc.Depot{}, fmt.Errorf("cari pelanggan: %w", err)
	}
	depot, err := h.q.GetDepot(c.Context(), cust.DepotID)
	if err != nil {
		return sqlc.Customer{}, sqlc.Depot{}, fmt.Errorf("ambil depot: %w", err)
	}
	return cust, depot, nil
}

func (h *Handler) me(c fiber.Ctx) error {
	noStore(c)
	cust, depot, err := h.loadMe(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	refillPrice := int64(0)
	if refill, err := h.q.GetRefillProduct(c.Context(), depot.ID); err == nil {
		refillPrice = refill.Price
	} else if !db.IsNoRows(err) {
		return httpx.Fail(c, fmt.Errorf("produk isi ulang: %w", err))
	}
	active, err := h.q.ListActiveOrdersByCustomer(c.Context(), cust.ID)
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("pesanan aktif: %w", err))
	}
	recent, err := h.q.ListOrdersByCustomer(c.Context(), sqlc.ListOrdersByCustomerParams{CustomerID: cust.ID, DepotID: depot.ID, Limit: recentOrdersLimit + int32(len(active))})
	if err != nil {
		return httpx.Fail(c, fmt.Errorf("riwayat pesanan: %w", err))
	}
	items, err := h.orders.ItemsFor(c.Context(), append(append([]sqlc.Order{}, active...), recent...))
	if err != nil {
		return httpx.Fail(c, err)
	}
	page := MePage{
		Customer:     MeCustomer{Name: cust.Name, PhoneMasked: phone.Mask(cust.Phone), UsualQty: cust.UsualQty, LoanBalance: cust.LoanBalance, StampCount: cust.StampCount, Area: cust.Area},
		Depot:        MeDepot{DepotView: depotView(depot), RefillPrice: refillPrice, LoyaltyEvery: depot.LoyaltyEvery},
		ActiveOrders: make([]TrackView, 0, len(active)),
		RecentOrders: make([]TrackView, 0, recentOrdersLimit),
	}
	for _, o := range active {
		page.ActiveOrders = append(page.ActiveOrders, trackView(o, items[o.ID], depot))
	}
	for _, o := range recent {
		if order.IsFinal(o.Status) && len(page.RecentOrders) < recentOrdersLimit {
			page.RecentOrders = append(page.RecentOrders, trackView(o, items[o.ID], depot))
		}
	}
	return httpx.OK(c, page)
}

type reorderRequest struct {
	Qty           int32  `json:"qty" validate:"omitempty,gte=1,lte=50"`
	ScheduledDate string `json:"scheduled_date" validate:"omitempty,datetime=2006-01-02"`
	Note          string `json:"note" validate:"max=300"`
	ReminderID    string `json:"r" validate:"omitempty,uuid"`
}

func (h *Handler) reorder(c fiber.Ctx) error {
	noStore(c)
	cust, depot, err := h.loadMe(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	var req reorderRequest
	if len(c.Body()) > 0 {
		if err := httpx.DecodeAndValidate(c, &req); err != nil {
			return httpx.Fail(c, err)
		}
	}
	if !depot.IsAcceptingOrders {
		return httpx.Fail(c, httpx.Conflict("Depot sedang tidak menerima pesanan."))
	}
	idemKey, err := order.IdempotencyKey(c)
	if err != nil {
		return httpx.Fail(c, err)
	}
	status := order.StatusPending
	if depot.AutoConfirmKnown && cust.IsVerified {
		status = order.StatusConfirmed
	}
	in := order.CreateInput{
		DepotID: depot.ID, CustomerID: cust.ID, RefillQty: req.Qty, Fulfilment: order.FulfilmentDelivery, Note: req.Note,
		Source: order.SourceLink, Status: status, Actor: order.Actor{Type: order.ActorCustomer, IP: httpx.ClientIP(c)}, IdempotencyKey: idemKey,
	}
	if req.ScheduledDate != "" {
		date, _ := time.Parse(time.DateOnly, req.ScheduledDate)
		today := order.DateIn(h.clock.Now(), order.DepotLocation(depot))
		if date.Before(today) || date.After(today.AddDate(0, 0, 1)) {
			return httpx.Fail(c, httpx.Validation(map[string]string{"scheduled_date": "Pilih hari ini atau besok."}))
		}
		in.ScheduledDate = &date
	}
	if req.ReminderID != "" {
		id := uuid.MustParse(req.ReminderID)
		in.ReminderID = &id
	}
	detail, created, err := h.orders.Create(c.Context(), in)
	if err != nil {
		return httpx.Fail(c, mapPublicCreateError(err))
	}
	return h.respondCreated(c, detail, created)
}
