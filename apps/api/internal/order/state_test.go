package order

import "testing"

func TestCanTransition(t *testing.T) {
	all := []string{StatusPending, StatusConfirmed, StatusOnDelivery, StatusDelivered, StatusCancelled}
	allowed := map[[2]string]bool{
		{StatusPending, StatusConfirmed}:    true,
		{StatusPending, StatusCancelled}:    true,
		{StatusConfirmed, StatusOnDelivery}: true,
		{StatusConfirmed, StatusDelivered}:  true,
		{StatusConfirmed, StatusCancelled}:  true,
		{StatusOnDelivery, StatusDelivered}: true,
		{StatusOnDelivery, StatusCancelled}: true,
	}
	for _, from := range all {
		for _, to := range all {
			want := allowed[[2]string{from, to}]
			if got := CanTransition(from, to); got != want {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
	if CanTransition("unknown", StatusConfirmed) || CanTransition(StatusPending, "unknown") {
		t.Fatal("unknown statuses must never transition")
	}
}

func TestActiveAndFinal(t *testing.T) {
	for _, s := range ActiveStatuses {
		if !IsActive(s) || IsFinal(s) {
			t.Errorf("%s should be active", s)
		}
	}
	for _, s := range []string{StatusDelivered, StatusCancelled} {
		if IsActive(s) || !IsFinal(s) {
			t.Errorf("%s should be final", s)
		}
	}
}
