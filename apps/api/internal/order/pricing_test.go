package order

import (
	"errors"
	"testing"
)

func refill(qty int32) Line {
	return Line{ProductID: "p1", ProductName: "Isi ulang galon", ProductKind: "refill", UnitPrice: 6000, Qty: qty}
}

func TestBuildQuote(t *testing.T) {
	cases := []struct {
		name       string
		lines      []Line
		fulfilment string
		fee        int64
		loyalty    LoyaltyInput
		wantTotal  int64
		wantFree   int32
		wantRefill int32
		wantErr    error
	}{
		{name: "two refills delivered", lines: []Line{refill(2)}, fulfilment: FulfilmentDelivery, fee: 2000, wantTotal: 14000, wantRefill: 2},
		{name: "pickup has no fee", lines: []Line{refill(2)}, fulfilment: FulfilmentPickup, fee: 2000, wantTotal: 12000, wantRefill: 2},
		{name: "mixed products count refill only", lines: []Line{refill(1), {ProductID: "p2", ProductKind: "new_gallon", UnitPrice: 45000, Qty: 1}}, fulfilment: FulfilmentPickup, wantTotal: 51000, wantRefill: 1},
		{name: "loyalty grants one free gallon", lines: []Line{refill(2)}, fulfilment: FulfilmentDelivery, fee: 0, loyalty: LoyaltyInput{Every: 10, StampCount: 10, RefillPrice: 6000}, wantTotal: 6000, wantFree: 1, wantRefill: 2},
		{name: "loyalty blocked by active free order", lines: []Line{refill(2)}, fulfilment: FulfilmentDelivery, loyalty: LoyaltyInput{Every: 10, StampCount: 12, HasActiveFreeOrder: true, RefillPrice: 6000}, wantTotal: 12000, wantRefill: 2},
		{name: "loyalty needs enough stamps", lines: []Line{refill(1)}, fulfilment: FulfilmentDelivery, loyalty: LoyaltyInput{Every: 10, StampCount: 9, RefillPrice: 6000}, wantTotal: 6000, wantRefill: 1},
		{name: "loyalty off", lines: []Line{refill(1)}, fulfilment: FulfilmentDelivery, loyalty: LoyaltyInput{Every: 0, StampCount: 99, RefillPrice: 6000}, wantTotal: 6000, wantRefill: 1},
		{name: "discount never exceeds subtotal", lines: []Line{{ProductID: "p1", ProductKind: "refill", UnitPrice: 0, Qty: 1}}, fulfilment: FulfilmentPickup, loyalty: LoyaltyInput{Every: 5, StampCount: 5, RefillPrice: 6000}, wantTotal: 0, wantFree: 1, wantRefill: 1},
		{name: "empty", lines: nil, wantErr: ErrEmptyOrder},
		{name: "qty too high", lines: []Line{refill(51)}, wantErr: ErrQtyOutOfRange},
		{name: "qty zero", lines: []Line{refill(0)}, wantErr: ErrQtyOutOfRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := BuildQuote(tc.lines, tc.fulfilment, tc.fee, tc.loyalty)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if q.Total != tc.wantTotal || q.FreeQty != tc.wantFree || q.RefillQty != tc.wantRefill {
				t.Fatalf("quote %+v", q)
			}
			if q.Subtotal+q.DeliveryFee-q.Discount != q.Total {
				t.Fatal("total arithmetic inconsistent")
			}
		})
	}
}

func TestStampsAfterDelivery(t *testing.T) {
	cases := []struct {
		current, refill, free, every, want int32
	}{
		{0, 2, 0, 10, 2},
		{9, 1, 0, 10, 10},
		{10, 2, 1, 10, 1},
		{10, 1, 1, 10, 0},
		{3, 1, 0, 0, 3},
	}
	for _, tc := range cases {
		if got := StampsAfterDelivery(tc.current, tc.refill, tc.free, tc.every); got != tc.want {
			t.Errorf("StampsAfterDelivery(%d,%d,%d,%d) = %d, want %d", tc.current, tc.refill, tc.free, tc.every, got, tc.want)
		}
	}
}
