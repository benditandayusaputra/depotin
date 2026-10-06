package gallon

import "errors"

const (
	KindDelivery   = "delivery"
	KindAdjustment = "adjustment"
	KindLost       = "lost"

	IdleAfterDays = 30
)

var (
	ErrNegativeBalance = errors.New("saldo galon tidak boleh negatif")
	ErrZeroDelta       = errors.New("perubahan tidak boleh nol")
	ErrNoteRequired    = errors.New("catatan wajib diisi")
)

func Apply(balance, delta int32) (int32, error) {
	if delta == 0 {
		return balance, ErrZeroDelta
	}
	next := balance + delta
	if next < 0 {
		return balance, ErrNegativeBalance
	}
	return next, nil
}

func DeliveryDelta(refillQty, gallonsReturned int32) int32 {
	return refillQty - gallonsReturned
}

func ValidateReturned(gallonsReturned, refillQty, loanBalance int32) bool {
	return gallonsReturned >= 0 && gallonsReturned <= refillQty+loanBalance
}
