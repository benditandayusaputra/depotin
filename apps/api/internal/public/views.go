package public

import (
	"time"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/phone"
	"github.com/benditandayusaputra/depotin/apps/api/internal/product"
)

type DepotView struct {
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	Phone             string `json:"phone"`
	Address           string `json:"address"`
	OpenTime          string `json:"open_time"`
	CloseTime         string `json:"close_time"`
	DeliveryFee       int64  `json:"delivery_fee"`
	IsAcceptingOrders bool   `json:"is_accepting_orders"`
}

type DepotPage struct {
	Depot    DepotView      `json:"depot"`
	Products []product.View `json:"products"`
}

func depotView(d sqlc.Depot) DepotView {
	return DepotView{
		Name: d.Name, Slug: d.Slug, Phone: d.Phone, Address: d.Address,
		OpenTime: shortTime(d.OpenTime), CloseTime: shortTime(d.CloseTime),
		DeliveryFee: d.DeliveryFee, IsAcceptingOrders: d.IsAcceptingOrders,
	}
}

func shortTime(t string) string {
	if len(t) >= 5 {
		return t[:5]
	}
	return t
}

type TrackView struct {
	Code            string           `json:"code"`
	Status          string           `json:"status"`
	Fulfilment      string           `json:"fulfilment"`
	ScheduledDate   string           `json:"scheduled_date"`
	DeliveryName    string           `json:"delivery_name"`
	DeliveryPhone   string           `json:"delivery_phone"`
	DeliveryAddress string           `json:"delivery_address"`
	RefillQty       int32            `json:"refill_qty"`
	FreeQty         int32            `json:"free_qty"`
	Total           int64            `json:"total"`
	PaymentStatus   string           `json:"payment_status"`
	Items           []order.ItemView `json:"items"`
	CanCancel       bool             `json:"can_cancel"`
	CreatedAt       time.Time        `json:"created_at"`
	ConfirmedAt     *time.Time       `json:"confirmed_at"`
	DispatchedAt    *time.Time       `json:"dispatched_at"`
	DeliveredAt     *time.Time       `json:"delivered_at"`
	CancelledAt     *time.Time       `json:"cancelled_at"`
	Depot           DepotView        `json:"depot"`
}

func trackView(o sqlc.Order, items []sqlc.OrderItem, d sqlc.Depot) TrackView {
	full := order.ToView(o, items, nil)
	return TrackView{
		Code: full.Code, Status: full.Status, Fulfilment: full.Fulfilment, ScheduledDate: full.ScheduledDate,
		DeliveryName: full.DeliveryName, DeliveryPhone: phone.Mask(full.DeliveryPhone), DeliveryAddress: full.DeliveryAddress,
		RefillQty: full.RefillQty, FreeQty: full.FreeQty, Total: full.Total, PaymentStatus: full.PaymentStatus, Items: full.Items,
		CanCancel: o.Status == order.StatusPending, CreatedAt: full.CreatedAt, ConfirmedAt: full.ConfirmedAt,
		DispatchedAt: full.DispatchedAt, DeliveredAt: full.DeliveredAt, CancelledAt: full.CancelledAt, Depot: depotView(d),
	}
}

type MeCustomer struct {
	Name        string `json:"name"`
	PhoneMasked string `json:"phone_masked"`
	UsualQty    int32  `json:"usual_qty"`
	LoanBalance int32  `json:"loan_balance"`
	StampCount  int32  `json:"stamp_count"`
	Area        string `json:"area"`
}

type MeDepot struct {
	DepotView
	RefillPrice  int64  `json:"refill_price"`
	LoyaltyEvery *int32 `json:"loyalty_every"`
}

type MePage struct {
	Customer     MeCustomer  `json:"customer"`
	Depot        MeDepot     `json:"depot"`
	ActiveOrders []TrackView `json:"active_orders"`
	RecentOrders []TrackView `json:"recent_orders"`
}

type CreatedOrder struct {
	Code       string `json:"code"`
	Status     string `json:"status"`
	TrackToken string `json:"track_token"`
}
