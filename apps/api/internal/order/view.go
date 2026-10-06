package order

import (
	"time"

	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
)

type ItemView struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	ProductKind string    `json:"product_kind"`
	UnitPrice   int64     `json:"unit_price"`
	Qty         int32     `json:"qty"`
	LineTotal   int64     `json:"line_total"`
}

type View struct {
	ID              uuid.UUID  `json:"id"`
	Code            string     `json:"code"`
	Source          string     `json:"source"`
	Status          string     `json:"status"`
	Fulfilment      string     `json:"fulfilment"`
	ScheduledDate   string     `json:"scheduled_date"`
	CustomerID      uuid.UUID  `json:"customer_id"`
	DeliveryName    string     `json:"delivery_name"`
	DeliveryPhone   string     `json:"delivery_phone"`
	DeliveryAddress string     `json:"delivery_address"`
	DeliveryNote    string     `json:"delivery_note"`
	Note            string     `json:"note"`
	RefillQty       int32      `json:"refill_qty"`
	FreeQty         int32      `json:"free_qty"`
	Subtotal        int64      `json:"subtotal"`
	DeliveryFee     int64      `json:"delivery_fee"`
	Discount        int64      `json:"discount"`
	Total           int64      `json:"total"`
	PaymentMethod   *string    `json:"payment_method"`
	PaymentStatus   string     `json:"payment_status"`
	PaidAt          *time.Time `json:"paid_at"`
	CourierID       *uuid.UUID `json:"courier_id"`
	CourierName     *string    `json:"courier_name"`
	GallonsReturned *int32     `json:"gallons_returned"`
	ReminderID      *uuid.UUID `json:"reminder_id"`
	Items           []ItemView `json:"items"`
	CreatedAt       time.Time  `json:"created_at"`
	ConfirmedAt     *time.Time `json:"confirmed_at"`
	DispatchedAt    *time.Time `json:"dispatched_at"`
	DeliveredAt     *time.Time `json:"delivered_at"`
	CancelledAt     *time.Time `json:"cancelled_at"`
	CancelReason    *string    `json:"cancel_reason"`
}

type CourierView struct {
	ID              uuid.UUID  `json:"id"`
	Code            string     `json:"code"`
	Status          string     `json:"status"`
	Fulfilment      string     `json:"fulfilment"`
	ScheduledDate   string     `json:"scheduled_date"`
	CustomerID      uuid.UUID  `json:"customer_id"`
	DeliveryName    string     `json:"delivery_name"`
	DeliveryPhone   string     `json:"delivery_phone"`
	DeliveryAddress string     `json:"delivery_address"`
	DeliveryNote    string     `json:"delivery_note"`
	Note            string     `json:"note"`
	RefillQty       int32      `json:"refill_qty"`
	FreeQty         int32      `json:"free_qty"`
	Total           int64      `json:"total"`
	PaymentStatus   string     `json:"payment_status"`
	PaymentMethod   *string    `json:"payment_method"`
	GallonsReturned *int32     `json:"gallons_returned"`
	Items           []ItemView `json:"items"`
	Lat             *float64   `json:"lat"`
	Lng             *float64   `json:"lng"`
	Area            string     `json:"area"`
	LoanBalance     int32      `json:"loan_balance"`
	DispatchedAt    *time.Time `json:"dispatched_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

func itemViews(items []sqlc.OrderItem) []ItemView {
	out := make([]ItemView, 0, len(items))
	for _, it := range items {
		out = append(out, ItemView{ProductID: it.ProductID, ProductName: it.ProductName, ProductKind: it.ProductKind, UnitPrice: it.UnitPrice, Qty: it.Qty, LineTotal: it.LineTotal})
	}
	return out
}

func ToView(o sqlc.Order, items []sqlc.OrderItem, courierName *string) View {
	return View{
		ID: o.ID, Code: o.Code, Source: o.Source, Status: o.Status, Fulfilment: o.Fulfilment,
		ScheduledDate: o.ScheduledDate.Format(time.DateOnly), CustomerID: o.CustomerID,
		DeliveryName: o.DeliveryName, DeliveryPhone: o.DeliveryPhone, DeliveryAddress: o.DeliveryAddress, DeliveryNote: o.DeliveryNote, Note: o.Note,
		RefillQty: o.RefillQty, FreeQty: o.FreeQty, Subtotal: o.Subtotal, DeliveryFee: o.DeliveryFee, Discount: o.Discount, Total: o.Total,
		PaymentMethod: o.PaymentMethod, PaymentStatus: o.PaymentStatus, PaidAt: o.PaidAt,
		CourierID: o.CourierID, CourierName: courierName, GallonsReturned: o.GallonsReturned, ReminderID: o.ReminderID,
		Items: itemViews(items), CreatedAt: o.CreatedAt, ConfirmedAt: o.ConfirmedAt, DispatchedAt: o.DispatchedAt,
		DeliveredAt: o.DeliveredAt, CancelledAt: o.CancelledAt, CancelReason: o.CancelReason,
	}
}

func ToCourierView(o sqlc.Order, items []sqlc.OrderItem, c sqlc.Customer) CourierView {
	return CourierView{
		ID: o.ID, Code: o.Code, Status: o.Status, Fulfilment: o.Fulfilment, ScheduledDate: o.ScheduledDate.Format(time.DateOnly),
		CustomerID: o.CustomerID, DeliveryName: o.DeliveryName, DeliveryPhone: o.DeliveryPhone, DeliveryAddress: o.DeliveryAddress,
		DeliveryNote: o.DeliveryNote, Note: o.Note, RefillQty: o.RefillQty, FreeQty: o.FreeQty, Total: o.Total,
		PaymentStatus: o.PaymentStatus, PaymentMethod: o.PaymentMethod, GallonsReturned: o.GallonsReturned, Items: itemViews(items),
		Lat: c.Lat, Lng: c.Lng, Area: c.Area, LoanBalance: c.LoanBalance, DispatchedAt: o.DispatchedAt, CreatedAt: o.CreatedAt,
	}
}

func groupItems(items []sqlc.OrderItem) map[uuid.UUID][]sqlc.OrderItem {
	grouped := make(map[uuid.UUID][]sqlc.OrderItem)
	for _, it := range items {
		grouped[it.OrderID] = append(grouped[it.OrderID], it)
	}
	return grouped
}
