package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/gallon"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
	"github.com/benditandayusaputra/depotin/apps/api/internal/prediction"
)

var ErrTooManyReturned = errors.New("galon kosong melebihi yang mungkin")

type DeliverInput struct {
	GallonsReturned int32
	PaymentMethod   string
	Paid            bool
}

func (s *Service) Deliver(ctx context.Context, depotID, orderID uuid.UUID, in DeliverInput, actor Actor, idemKey string) (Detail, error) {
	if idemKey != "" {
		done, err := s.q.OrderEventExists(ctx, sqlc.OrderEventExistsParams{OrderID: orderID, IdempotencyKey: &idemKey})
		if err != nil {
			return Detail{}, fmt.Errorf("cek idempotensi: %w", err)
		}
		if done {
			return s.Load(ctx, depotID, orderID)
		}
	}
	var updated sqlc.Order
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		updated, err = s.deliverTx(ctx, s.q.WithTx(tx), depotID, orderID, in, actor, idemKey)
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

func (s *Service) deliverTx(ctx context.Context, q *sqlc.Queries, depotID, orderID uuid.UUID, in DeliverInput, actor Actor, idemKey string) (sqlc.Order, error) {
	now := s.clock.Now()
	current, err := q.GetOrderForUpdate(ctx, sqlc.GetOrderForUpdateParams{ID: orderID, DepotID: depotID})
	if err != nil {
		if db.IsNoRows(err) {
			return sqlc.Order{}, httpx.NotFound("Pesanan tidak ditemukan.")
		}
		return sqlc.Order{}, fmt.Errorf("kunci pesanan: %w", err)
	}
	if actor.Role == roleCourier {
		if current.CourierID == nil || actor.UserID == nil || *current.CourierID != *actor.UserID {
			return sqlc.Order{}, ErrNotAssigned
		}
		if current.Status != StatusOnDelivery {
			return sqlc.Order{}, httpx.InvalidTransition("")
		}
	}
	if !CanTransition(current.Status, StatusDelivered) {
		return sqlc.Order{}, httpx.InvalidTransition("")
	}
	customer, err := q.GetCustomerForUpdate(ctx, sqlc.GetCustomerForUpdateParams{ID: current.CustomerID, DepotID: depotID})
	if err != nil {
		return sqlc.Order{}, fmt.Errorf("kunci pelanggan: %w", err)
	}
	if !gallon.ValidateReturned(in.GallonsReturned, current.RefillQty, customer.LoanBalance) {
		return sqlc.Order{}, ErrTooManyReturned
	}
	depot, err := q.GetDepot(ctx, depotID)
	if err != nil {
		return sqlc.Order{}, fmt.Errorf("ambil depot: %w", err)
	}

	paymentStatus := PaymentUnpaid
	if in.Paid {
		paymentStatus = PaymentPaid
	}
	var method *string
	if in.PaymentMethod != "" {
		method = &in.PaymentMethod
	}
	delivered, err := q.DeliverOrder(ctx, sqlc.DeliverOrderParams{
		ID: orderID, DepotID: depotID, DeliveredAt: &now, GallonsReturned: &in.GallonsReturned,
		PaymentMethod: method, PaymentStatus: paymentStatus, FromStatus: current.Status,
	})
	if err != nil {
		return sqlc.Order{}, fmt.Errorf("selesaikan pesanan: %w", err)
	}

	balance := customer.LoanBalance
	if delta := gallon.DeliveryDelta(delivered.RefillQty, in.GallonsReturned); delta != 0 {
		balance, err = gallon.Apply(customer.LoanBalance, delta)
		if err != nil {
			return sqlc.Order{}, err
		}
		if _, err := q.InsertLedger(ctx, sqlc.InsertLedgerParams{
			ID: idgen.NewID(), DepotID: depotID, CustomerID: customer.ID, OrderID: &delivered.ID, Kind: gallon.KindDelivery,
			Delta: delta, BalanceAfter: balance, CreatedBy: actor.UserID,
		}); err != nil {
			return sqlc.Order{}, fmt.Errorf("tulis buku galon: %w", err)
		}
	}

	var every int32
	if depot.LoyaltyEvery != nil {
		every = *depot.LoyaltyEvery
	}
	stamps := StampsAfterDelivery(customer.StampCount, delivered.RefillQty, delivered.FreeQty, every)

	history, err := q.ListRecentDeliveries(ctx, sqlc.ListRecentDeliveriesParams{CustomerID: customer.ID, Limit: prediction.MaxDeliveries})
	if err != nil {
		return sqlc.Order{}, fmt.Errorf("riwayat pengantaran: %w", err)
	}
	deliveries := make([]prediction.Delivery, 0, len(history))
	for _, h := range history {
		deliveries = append(deliveries, prediction.Delivery{DeliveredAt: *h.DeliveredAt, RefillQty: h.RefillQty})
	}
	params := sqlc.ApplyDeliveryToCustomerParams{
		ID: customer.ID, DepotID: depotID, LoanBalance: balance, StampCount: stamps,
		DaysPerGallon: customer.DaysPerGallon, PredictionSamples: customer.PredictionSamples, PredictionConfidence: customer.PredictionConfidence,
		PredictedEmptyAt: customer.PredictedEmptyAt, LastDeliveredAt: customer.LastDeliveredAt, LastDeliveredQty: customer.LastDeliveredQty,
	}
	if res, ok := prediction.Predict(deliveries, depot.DefaultDaysPerGallon); ok {
		days := res.DaysPerGallon
		predictedAt := res.PredictedEmptyAt
		lastQty := delivered.RefillQty
		params.DaysPerGallon = &days
		params.PredictionSamples = int32(res.Samples)
		params.PredictionConfidence = res.Confidence
		params.PredictedEmptyAt = &predictedAt
		params.LastDeliveredAt = &now
		params.LastDeliveredQty = &lastQty
	}
	if err := q.ApplyDeliveryToCustomer(ctx, params); err != nil {
		return sqlc.Order{}, fmt.Errorf("perbarui pelanggan: %w", err)
	}
	meta := map[string]any{"gallons_returned": in.GallonsReturned, "payment_status": paymentStatus}
	if err := s.event(ctx, q, delivered, EventDelivered, actor, idemKey, meta); err != nil {
		return sqlc.Order{}, err
	}
	if in.Paid {
		if err := s.event(ctx, q, delivered, EventPaid, actor, "", nil); err != nil {
			return sqlc.Order{}, err
		}
	}
	return delivered, nil
}
