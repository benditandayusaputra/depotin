package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
)

const (
	trackTokenBytes = 16
	maxScheduleDays = 7

	ActorUser     = "user"
	ActorCustomer = "customer"
	ActorSystem   = "system"
)

var (
	ErrCustomerNotFound = errors.New("pelanggan tidak ditemukan")
	ErrProductNotFound  = errors.New("produk tidak ditemukan")
	ErrNoRefillProduct  = errors.New("depot belum punya produk isi ulang")
	ErrCourierNotFound  = errors.New("kurir tidak ditemukan")
	ErrNotAssigned      = errors.New("pesanan bukan milik kurir ini")
	ErrScheduleRange    = errors.New("tanggal antar di luar rentang")
)

type Publisher interface {
	OrderCreated(depotID uuid.UUID, view View)
	OrderUpdated(depotID uuid.UUID, view View)
}

type NopPublisher struct{}

func (NopPublisher) OrderCreated(uuid.UUID, View) {}
func (NopPublisher) OrderUpdated(uuid.UUID, View) {}

type Actor struct {
	Type   string
	UserID *uuid.UUID
	Role   string
	IP     string
}

type ItemInput struct {
	ProductID uuid.UUID
	Qty       int32
}

type DeliveryCopy struct {
	Name    string
	Phone   string
	Address string
	Note    string
}

type CreateInput struct {
	DepotID        uuid.UUID
	CustomerID     uuid.UUID
	Items          []ItemInput
	RefillQty      int32
	Fulfilment     string
	ScheduledDate  *time.Time
	Note           string
	Source         string
	Status         string
	Actor          Actor
	ReminderID     *uuid.UUID
	IdempotencyKey string
	Delivery       *DeliveryCopy
}

type Detail struct {
	Order    sqlc.Order
	Items    []sqlc.OrderItem
	Customer sqlc.Customer
	Token    string
}

type Service struct {
	pool      *pgxpool.Pool
	q         *sqlc.Queries
	clock     clock.Clock
	publisher Publisher
}

func NewService(pool *pgxpool.Pool, clk clock.Clock, publisher Publisher) *Service {
	if publisher == nil {
		publisher = NopPublisher{}
	}
	return &Service{pool: pool, q: sqlc.New(pool), clock: clk, publisher: publisher}
}

func (s *Service) Queries() *sqlc.Queries {
	return s.q
}

func DepotLocation(d sqlc.Depot) *time.Location {
	loc, err := time.LoadLocation(d.Timezone)
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}

func DateIn(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Detail, bool, error) {
	if in.IdempotencyKey != "" {
		existing, err := s.q.GetOrderByIdempotencyKey(ctx, sqlc.GetOrderByIdempotencyKeyParams{DepotID: in.DepotID, IdempotencyKey: &in.IdempotencyKey})
		if err == nil {
			detail, err := s.Load(ctx, in.DepotID, existing.ID)
			return detail, false, err
		}
		if !db.IsNoRows(err) {
			return Detail{}, false, fmt.Errorf("cek idempotensi: %w", err)
		}
	}

	var detail Detail
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		detail, err = s.createTx(ctx, s.q.WithTx(tx), in)
		return err
	})
	if err != nil {
		if db.IsUniqueViolation(err) && db.ConstraintName(err) == "orders_idempotency_idx" && in.IdempotencyKey != "" {
			existing, ferr := s.q.GetOrderByIdempotencyKey(ctx, sqlc.GetOrderByIdempotencyKeyParams{DepotID: in.DepotID, IdempotencyKey: &in.IdempotencyKey})
			if ferr != nil {
				return Detail{}, false, fmt.Errorf("ambil pesanan idempoten: %w", ferr)
			}
			detail, err := s.Load(ctx, in.DepotID, existing.ID)
			return detail, false, err
		}
		return Detail{}, false, err
	}
	s.publisher.OrderCreated(in.DepotID, ToView(detail.Order, detail.Items, nil))
	return detail, true, nil
}

func (s *Service) createTx(ctx context.Context, q *sqlc.Queries, in CreateInput) (Detail, error) {
	now := s.clock.Now()
	customer, err := q.GetCustomerForUpdate(ctx, sqlc.GetCustomerForUpdateParams{ID: in.CustomerID, DepotID: in.DepotID})
	if err != nil {
		if db.IsNoRows(err) {
			return Detail{}, ErrCustomerNotFound
		}
		return Detail{}, fmt.Errorf("kunci pelanggan: %w", err)
	}
	depot, err := q.GetDepot(ctx, in.DepotID)
	if err != nil {
		return Detail{}, fmt.Errorf("ambil depot: %w", err)
	}
	loc := DepotLocation(depot)
	today := DateIn(now, loc)
	scheduled := today
	if in.ScheduledDate != nil {
		scheduled = *in.ScheduledDate
		if scheduled.Before(today) || scheduled.After(today.AddDate(0, 0, maxScheduleDays)) {
			return Detail{}, ErrScheduleRange
		}
	}

	refillProduct, err := q.GetRefillProduct(ctx, in.DepotID)
	hasRefill := err == nil
	if err != nil && !db.IsNoRows(err) {
		return Detail{}, fmt.Errorf("ambil produk isi ulang: %w", err)
	}

	lines, err := s.resolveLines(ctx, q, in, customer, refillProduct, hasRefill)
	if err != nil {
		return Detail{}, err
	}

	loyalty := LoyaltyInput{StampCount: customer.StampCount}
	if depot.LoyaltyEvery != nil && hasRefill {
		loyalty.Every = *depot.LoyaltyEvery
		loyalty.RefillPrice = refillProduct.Price
		loyalty.HasActiveFreeOrder, err = q.HasActiveFreeOrder(ctx, customer.ID)
		if err != nil {
			return Detail{}, fmt.Errorf("cek pesanan gratis aktif: %w", err)
		}
	}
	quote, err := BuildQuote(lines, in.Fulfilment, depot.DeliveryFee, loyalty)
	if err != nil {
		return Detail{}, err
	}

	seq, err := q.NextOrderNumber(ctx, sqlc.NextOrderNumberParams{DepotID: in.DepotID, Day: today})
	if err != nil {
		return Detail{}, fmt.Errorf("nomor urut pesanan: %w", err)
	}
	token, err := idgen.RandomToken(trackTokenBytes)
	if err != nil {
		return Detail{}, err
	}

	delivery := DeliveryCopy{Name: customer.Name, Phone: customer.Phone, Address: customer.Address, Note: customer.AddressNote}
	if in.Delivery != nil {
		delivery = *in.Delivery
	}
	var idemKey *string
	if in.IdempotencyKey != "" {
		idemKey = &in.IdempotencyKey
	}
	var confirmedAt *time.Time
	if in.Status == StatusConfirmed {
		confirmedAt = &now
	}
	order, err := q.CreateOrder(ctx, sqlc.CreateOrderParams{
		ID: idgen.NewID(), DepotID: in.DepotID, CustomerID: customer.ID, Code: idgen.OrderCode(today, int(seq)),
		Source: in.Source, Status: in.Status, Fulfilment: in.Fulfilment, ScheduledDate: scheduled,
		DeliveryName: delivery.Name, DeliveryPhone: delivery.Phone, DeliveryAddress: delivery.Address, DeliveryNote: delivery.Note, Note: in.Note,
		RefillQty: quote.RefillQty, FreeQty: quote.FreeQty, Subtotal: quote.Subtotal, DeliveryFee: quote.DeliveryFee, Discount: quote.Discount, Total: quote.Total,
		ReminderID: in.ReminderID, TrackTokenHash: crypto.HashToken(token), IdempotencyKey: idemKey, CreatedBy: in.Actor.UserID,
		CreatedAt: now, ConfirmedAt: confirmedAt,
	})
	if err != nil {
		return Detail{}, fmt.Errorf("simpan pesanan: %w", err)
	}

	items := make([]sqlc.OrderItem, 0, len(quote.Lines))
	for _, l := range quote.Lines {
		item := sqlc.OrderItem{ID: idgen.NewID(), OrderID: order.ID, ProductID: uuid.MustParse(l.ProductID), ProductName: l.ProductName, ProductKind: l.ProductKind, UnitPrice: l.UnitPrice, Qty: l.Qty, LineTotal: l.LineTotal}
		if err := q.InsertOrderItem(ctx, sqlc.InsertOrderItemParams(item)); err != nil {
			return Detail{}, fmt.Errorf("simpan butir pesanan: %w", err)
		}
		items = append(items, item)
	}
	if err := s.event(ctx, q, order, EventCreated, in.Actor, "", map[string]any{"status": in.Status, "source": in.Source}); err != nil {
		return Detail{}, err
	}
	if in.Status == StatusConfirmed {
		if err := s.event(ctx, q, order, EventConfirmed, in.Actor, "", nil); err != nil {
			return Detail{}, err
		}
	}
	return Detail{Order: order, Items: items, Customer: customer, Token: token}, nil
}

func (s *Service) resolveLines(ctx context.Context, q *sqlc.Queries, in CreateInput, customer sqlc.Customer, refill sqlc.Product, hasRefill bool) ([]Line, error) {
	if len(in.Items) == 0 {
		if !hasRefill {
			return nil, ErrNoRefillProduct
		}
		qty := in.RefillQty
		if qty == 0 {
			qty = customer.UsualQty
		}
		return []Line{{ProductID: refill.ID.String(), ProductName: refill.Name, ProductKind: refill.Kind, UnitPrice: refill.Price, Qty: qty}}, nil
	}
	lines := make([]Line, 0, len(in.Items))
	for _, it := range in.Items {
		p, err := q.GetProduct(ctx, sqlc.GetProductParams{ID: it.ProductID, DepotID: in.DepotID})
		if err != nil || !p.IsActive {
			if err != nil && !db.IsNoRows(err) {
				return nil, fmt.Errorf("ambil produk: %w", err)
			}
			return nil, ErrProductNotFound
		}
		lines = append(lines, Line{ProductID: p.ID.String(), ProductName: p.Name, ProductKind: p.Kind, UnitPrice: p.Price, Qty: it.Qty})
	}
	return lines, nil
}

func (s *Service) event(ctx context.Context, q *sqlc.Queries, o sqlc.Order, kind string, actor Actor, idemKey string, meta map[string]any) error {
	raw := []byte("{}")
	if len(meta) > 0 {
		encoded, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("encode meta peristiwa: %w", err)
		}
		raw = encoded
	}
	var key *string
	if idemKey != "" {
		key = &idemKey
	}
	if err := q.InsertOrderEvent(ctx, sqlc.InsertOrderEventParams{
		ID: idgen.NewID(), DepotID: o.DepotID, OrderID: o.ID, Type: kind, ActorType: actor.Type, ActorID: actor.UserID, IdempotencyKey: key, Meta: raw,
	}); err != nil {
		return fmt.Errorf("simpan peristiwa pesanan: %w", err)
	}
	return nil
}

func (s *Service) Load(ctx context.Context, depotID, orderID uuid.UUID) (Detail, error) {
	o, err := s.q.GetOrder(ctx, sqlc.GetOrderParams{ID: orderID, DepotID: depotID})
	if err != nil {
		if db.IsNoRows(err) {
			return Detail{}, httpx.NotFound("Pesanan tidak ditemukan.")
		}
		return Detail{}, fmt.Errorf("ambil pesanan: %w", err)
	}
	items, err := s.q.ListOrderItems(ctx, []uuid.UUID{o.ID})
	if err != nil {
		return Detail{}, fmt.Errorf("ambil butir pesanan: %w", err)
	}
	customer, err := s.q.GetCustomer(ctx, sqlc.GetCustomerParams{ID: o.CustomerID, DepotID: depotID})
	if err != nil {
		return Detail{}, fmt.Errorf("ambil pelanggan pesanan: %w", err)
	}
	return Detail{Order: o, Items: items, Customer: customer}, nil
}

func (s *Service) ItemsFor(ctx context.Context, orders []sqlc.Order) (map[uuid.UUID][]sqlc.OrderItem, error) {
	ids := make([]uuid.UUID, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.ID)
	}
	if len(ids) == 0 {
		return map[uuid.UUID][]sqlc.OrderItem{}, nil
	}
	items, err := s.q.ListOrderItems(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ambil butir pesanan: %w", err)
	}
	return groupItems(items), nil
}

func (s *Service) CourierNames(ctx context.Context, depotID uuid.UUID) (map[uuid.UUID]string, error) {
	users, err := s.q.ListUsersByDepot(ctx, depotID)
	if err != nil {
		return nil, fmt.Errorf("daftar pengguna: %w", err)
	}
	names := make(map[uuid.UUID]string, len(users))
	for _, u := range users {
		names[u.ID] = u.Name
	}
	return names, nil
}

func (s *Service) Confirm(ctx context.Context, depotID, orderID uuid.UUID, actor Actor) (Detail, error) {
	now := s.clock.Now()
	return s.transition(ctx, depotID, orderID, func(q *sqlc.Queries) (sqlc.Order, error) {
		o, err := q.ConfirmOrder(ctx, sqlc.ConfirmOrderParams{ID: orderID, DepotID: depotID, ConfirmedAt: &now})
		if err != nil {
			return sqlc.Order{}, fmt.Errorf("konfirmasi pesanan: %w", err)
		}
		return o, s.event(ctx, q, o, EventConfirmed, actor, "", nil)
	})
}

func (s *Service) Assign(ctx context.Context, depotID, orderID, courierID uuid.UUID, actor Actor) (Detail, error) {
	courier, err := s.q.GetUserInDepot(ctx, sqlc.GetUserInDepotParams{ID: courierID, DepotID: depotID})
	if err != nil || courier.Role != "courier" || !courier.IsActive {
		if err != nil && !db.IsNoRows(err) {
			return Detail{}, fmt.Errorf("cari kurir: %w", err)
		}
		return Detail{}, ErrCourierNotFound
	}
	return s.transition(ctx, depotID, orderID, func(q *sqlc.Queries) (sqlc.Order, error) {
		o, err := q.AssignCourier(ctx, sqlc.AssignCourierParams{ID: orderID, DepotID: depotID, CourierID: &courierID})
		if err != nil {
			return sqlc.Order{}, fmt.Errorf("tugaskan kurir: %w", err)
		}
		return o, s.event(ctx, q, o, EventAssigned, actor, "", map[string]any{"courier_id": courierID})
	})
}

func (s *Service) Dispatch(ctx context.Context, depotID, orderID uuid.UUID, actor Actor, idemKey string) (Detail, error) {
	if idemKey != "" {
		done, err := s.q.OrderEventExists(ctx, sqlc.OrderEventExistsParams{OrderID: orderID, IdempotencyKey: &idemKey})
		if err != nil {
			return Detail{}, fmt.Errorf("cek idempotensi: %w", err)
		}
		if done {
			return s.Load(ctx, depotID, orderID)
		}
	}
	now := s.clock.Now()
	return s.transition(ctx, depotID, orderID, func(q *sqlc.Queries) (sqlc.Order, error) {
		if err := s.requireAssigned(ctx, q, depotID, orderID, actor); err != nil {
			return sqlc.Order{}, err
		}
		var courierID *uuid.UUID
		if actor.Role == "courier" {
			courierID = actor.UserID
		}
		o, err := q.DispatchOrder(ctx, sqlc.DispatchOrderParams{ID: orderID, DepotID: depotID, DispatchedAt: &now, CourierID: courierID})
		if err != nil {
			return sqlc.Order{}, fmt.Errorf("berangkatkan pesanan: %w", err)
		}
		return o, s.event(ctx, q, o, EventDispatched, actor, idemKey, nil)
	})
}

func (s *Service) requireAssigned(ctx context.Context, q *sqlc.Queries, depotID, orderID uuid.UUID, actor Actor) error {
	if actor.Role != "courier" || actor.UserID == nil {
		return nil
	}
	o, err := q.GetOrderForUpdate(ctx, sqlc.GetOrderForUpdateParams{ID: orderID, DepotID: depotID})
	if err != nil {
		return fmt.Errorf("kunci pesanan: %w", err)
	}
	if o.CourierID == nil || *o.CourierID != *actor.UserID {
		return ErrNotAssigned
	}
	return nil
}

func (s *Service) Cancel(ctx context.Context, depotID, orderID uuid.UUID, reason string, actor Actor) (Detail, error) {
	now := s.clock.Now()
	return s.transition(ctx, depotID, orderID, func(q *sqlc.Queries) (sqlc.Order, error) {
		o, err := q.CancelOrder(ctx, sqlc.CancelOrderParams{ID: orderID, DepotID: depotID, CancelledAt: &now, CancelReason: &reason})
		if err != nil {
			return sqlc.Order{}, fmt.Errorf("batalkan pesanan: %w", err)
		}
		if err := q.ReopenReminderForCancelledOrder(ctx, &o.ID); err != nil {
			return sqlc.Order{}, fmt.Errorf("buka ulang pengingat: %w", err)
		}
		if err := audit.Record(ctx, q, audit.Entry{DepotID: depotID, UserID: actor.UserID, Action: audit.ActionOrderCancelled, EntityType: "order", EntityID: &o.ID, Meta: map[string]any{"reason": reason}, IP: actor.IP}); err != nil {
			return sqlc.Order{}, err
		}
		return o, s.event(ctx, q, o, EventCancelled, actor, "", map[string]any{"reason": reason})
	})
}

func (s *Service) MarkPaid(ctx context.Context, depotID, orderID uuid.UUID, method *string, actor Actor) (Detail, error) {
	now := s.clock.Now()
	return s.transition(ctx, depotID, orderID, func(q *sqlc.Queries) (sqlc.Order, error) {
		o, err := q.MarkOrderPaid(ctx, sqlc.MarkOrderPaidParams{ID: orderID, DepotID: depotID, PaidAt: &now, PaymentMethod: method})
		if err != nil {
			return sqlc.Order{}, fmt.Errorf("tandai lunas: %w", err)
		}
		return o, s.event(ctx, q, o, EventPaid, actor, "", nil)
	})
}

func (s *Service) transition(ctx context.Context, depotID, orderID uuid.UUID, apply func(q *sqlc.Queries) (sqlc.Order, error)) (Detail, error) {
	var updated sqlc.Order
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		updated, err = apply(s.q.WithTx(tx))
		return err
	})
	if err != nil {
		if db.IsNoRows(err) {
			return s.explainNoRows(ctx, depotID, orderID)
		}
		return Detail{}, err
	}
	detail, err := s.Load(ctx, depotID, updated.ID)
	if err != nil {
		return Detail{}, err
	}
	s.publisher.OrderUpdated(depotID, ToView(detail.Order, detail.Items, nil))
	return detail, nil
}

func (s *Service) explainNoRows(ctx context.Context, depotID, orderID uuid.UUID) (Detail, error) {
	if _, err := s.q.GetOrder(ctx, sqlc.GetOrderParams{ID: orderID, DepotID: depotID}); err != nil {
		if db.IsNoRows(err) {
			return Detail{}, httpx.NotFound("Pesanan tidak ditemukan.")
		}
		return Detail{}, fmt.Errorf("ambil pesanan: %w", err)
	}
	return Detail{}, httpx.InvalidTransition("")
}
