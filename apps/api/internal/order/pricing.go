package order

import "errors"

const (
	MinQty = 1
	MaxQty = 50
)

var (
	ErrEmptyOrder    = errors.New("pesanan tidak punya barang")
	ErrQtyOutOfRange = errors.New("jumlah di luar rentang")
)

type Line struct {
	ProductID   string
	ProductName string
	ProductKind string
	UnitPrice   int64
	Qty         int32
	LineTotal   int64
}

type LoyaltyInput struct {
	Every              int32
	StampCount         int32
	HasActiveFreeOrder bool
	RefillPrice        int64
}

type Quote struct {
	Lines       []Line
	RefillQty   int32
	FreeQty     int32
	Subtotal    int64
	DeliveryFee int64
	Discount    int64
	Total       int64
}

func BuildQuote(lines []Line, fulfilment string, deliveryFee int64, loyalty LoyaltyInput) (Quote, error) {
	if len(lines) == 0 {
		return Quote{}, ErrEmptyOrder
	}
	var q Quote
	q.Lines = make([]Line, 0, len(lines))
	for _, l := range lines {
		if l.Qty < MinQty || l.Qty > MaxQty {
			return Quote{}, ErrQtyOutOfRange
		}
		l.LineTotal = l.UnitPrice * int64(l.Qty)
		q.Lines = append(q.Lines, l)
		q.Subtotal += l.LineTotal
		if l.ProductKind == "refill" {
			q.RefillQty += l.Qty
		}
	}
	if fulfilment == FulfilmentDelivery {
		q.DeliveryFee = deliveryFee
	}
	q.FreeQty = freeGallons(loyalty, q.RefillQty)
	q.Discount = int64(q.FreeQty) * loyalty.RefillPrice
	if q.Discount > q.Subtotal {
		q.Discount = q.Subtotal
	}
	q.Total = q.Subtotal + q.DeliveryFee - q.Discount
	return q, nil
}

func freeGallons(l LoyaltyInput, refillQty int32) int32 {
	if l.Every <= 0 || refillQty < 1 || l.HasActiveFreeOrder || l.StampCount < l.Every {
		return 0
	}
	return 1
}

func StampsAfterDelivery(current, refillQty, freeQty, every int32) int32 {
	if every <= 0 {
		return current
	}
	next := current + (refillQty - freeQty) - freeQty*every
	if next < 0 {
		return 0
	}
	return next
}
