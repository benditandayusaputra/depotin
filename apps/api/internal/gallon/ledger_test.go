package gallon

import (
	"errors"
	"testing"
)

func TestApply(t *testing.T) {
	cases := []struct {
		balance, delta, want int32
		err                  error
	}{
		{0, 2, 2, nil},
		{2, -2, 0, nil},
		{1, -2, 1, ErrNegativeBalance},
		{3, 0, 3, ErrZeroDelta},
	}
	for _, tc := range cases {
		got, err := Apply(tc.balance, tc.delta)
		if got != tc.want || !errors.Is(err, tc.err) {
			t.Errorf("Apply(%d,%d) = %d,%v want %d,%v", tc.balance, tc.delta, got, err, tc.want, tc.err)
		}
	}
}

func TestDeliveryRules(t *testing.T) {
	if DeliveryDelta(2, 2) != 0 || DeliveryDelta(2, 1) != 1 || DeliveryDelta(1, 3) != -2 {
		t.Fatal("delta arithmetic")
	}
	if !ValidateReturned(3, 2, 1) || ValidateReturned(4, 2, 1) || ValidateReturned(-1, 2, 1) {
		t.Fatal("returned validation")
	}
}
