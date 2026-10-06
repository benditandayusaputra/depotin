package prediction

import (
	"math"
	"testing"
	"time"
)

var base = time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

func days(n float64) time.Duration {
	return time.Duration(n * 24 * float64(time.Hour))
}

func deliveries(qty int32, gapsInDays ...float64) []Delivery {
	out := []Delivery{{DeliveredAt: base, RefillQty: qty}}
	at := base
	for _, g := range gapsInDays {
		at = at.Add(days(g))
		out = append(out, Delivery{DeliveredAt: at, RefillQty: qty})
	}
	return out
}

func near(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func TestNoHistory(t *testing.T) {
	res, ok := Predict(nil, 4)
	if ok || res.Confidence != ConfidenceNone {
		t.Fatalf("expected no prediction, got %+v ok=%v", res, ok)
	}
	res, ok = Predict([]Delivery{{DeliveredAt: base, RefillQty: 0}}, 4)
	if ok || res.Confidence != ConfidenceNone {
		t.Fatalf("zero qty deliveries must not predict, got %+v", res)
	}
}

func TestSingleDeliveryUsesDepotDefault(t *testing.T) {
	res, ok := Predict(deliveries(2), 4)
	if !ok {
		t.Fatal("expected prediction")
	}
	if res.DaysPerGallon != 4 || res.Confidence != ConfidenceLow || res.Samples != 0 {
		t.Fatalf("got %+v", res)
	}
	if !res.PredictedEmptyAt.Equal(base.Add(days(8))) {
		t.Fatalf("predicted %v", res.PredictedEmptyAt)
	}
	if res.Explanation != "Belum ada pola, dipakai angka bawaan depot 4 hari per galon." {
		t.Fatalf("explanation %q", res.Explanation)
	}
}

func TestStablePattern(t *testing.T) {
	res, ok := Predict(deliveries(2, 7, 7, 7, 7, 7), 4)
	if !ok {
		t.Fatal("expected prediction")
	}
	if res.Samples != 5 || res.Confidence != ConfidenceHigh {
		t.Fatalf("got %+v", res)
	}
	if !near(res.DaysPerGallon, (5*3.5+4)/6, 0.01) {
		t.Fatalf("days per gallon %v", res.DaysPerGallon)
	}
	last := base.Add(days(35))
	expected := last.Add(time.Duration(2 * res.DaysPerGallon * 24 * float64(time.Hour)))
	if !near(res.PredictedEmptyAt.Sub(expected).Hours(), 0, 1) {
		t.Fatalf("predicted %v want about %v", res.PredictedEmptyAt, expected)
	}
	if res.Explanation != "Biasanya 1 galon habis dalam 3,6 hari, dihitung dari 6 pesanan terakhir." {
		t.Fatalf("explanation %q", res.Explanation)
	}
}

func TestLongGapInTheMiddleIsIgnored(t *testing.T) {
	withGap, _ := Predict(deliveries(1, 4, 4, 40, 4, 4), 4)
	stable, _ := Predict(deliveries(1, 4, 4, 4, 4, 4), 4)
	if withGap.Samples != 4 {
		t.Fatalf("outlier should be dropped, samples=%d", withGap.Samples)
	}
	if !near(withGap.DaysPerGallon, stable.DaysPerGallon, 0.01) {
		t.Fatalf("gap changed estimate: %v vs %v", withGap.DaysPerGallon, stable.DaysPerGallon)
	}
	if withGap.Confidence != ConfidenceHigh {
		t.Fatalf("confidence %s", withGap.Confidence)
	}
}

func TestChangingQuantityNormalisesPerGallon(t *testing.T) {
	list := []Delivery{
		{DeliveredAt: base, RefillQty: 1},
		{DeliveredAt: base.Add(days(4)), RefillQty: 2},
		{DeliveredAt: base.Add(days(12)), RefillQty: 1},
		{DeliveredAt: base.Add(days(16)), RefillQty: 3},
	}
	res, ok := Predict(list, 4)
	if !ok || res.Samples != 3 {
		t.Fatalf("got %+v", res)
	}
	if !near(res.DaysPerGallon, 4, 0.01) {
		t.Fatalf("days per gallon %v", res.DaysPerGallon)
	}
	if !res.PredictedEmptyAt.Equal(list[3].DeliveredAt.Add(days(12))) {
		t.Fatalf("predicted %v", res.PredictedEmptyAt)
	}
	if res.Confidence != ConfidenceMedium {
		t.Fatalf("confidence %s", res.Confidence)
	}
}

func TestTwoDeliveriesSameDay(t *testing.T) {
	list := []Delivery{
		{DeliveredAt: base, RefillQty: 1},
		{DeliveredAt: base.Add(2 * time.Hour), RefillQty: 1},
	}
	res, ok := Predict(list, 4)
	if !ok || res.Samples != 1 || res.Confidence != ConfidenceLow {
		t.Fatalf("got %+v", res)
	}
	if !near(res.DaysPerGallon, (0.25+4)/2, 0.01) {
		t.Fatalf("days per gallon %v", res.DaysPerGallon)
	}
}

func TestClampAndCaps(t *testing.T) {
	fast, _ := Predict(deliveries(50, 0.3, 0.3, 0.3, 0.3), 4)
	if fast.DaysPerGallon < minDaysPerGallon {
		t.Fatalf("below minimum: %v", fast.DaysPerGallon)
	}
	slow, _ := Predict(deliveries(1, 200, 200, 200, 200), 30)
	if slow.DaysPerGallon > maxDaysPerGallon {
		t.Fatalf("above maximum: %v", slow.DaysPerGallon)
	}
	many, _ := Predict(deliveries(1, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3), 4)
	if many.Samples != MaxDeliveries-1 {
		t.Fatalf("should use at most %d deliveries, samples=%d", MaxDeliveries, many.Samples)
	}
}

func TestIrregularPatternIsMediumConfidence(t *testing.T) {
	res, _ := Predict(deliveries(1, 2, 6, 2, 6, 2), 4)
	if res.Samples != 5 || res.Confidence != ConfidenceMedium {
		t.Fatalf("got %+v", res)
	}
}

func TestUnsortedInputIsSorted(t *testing.T) {
	list := deliveries(1, 4, 4, 4)
	list[0], list[3] = list[3], list[0]
	res, _ := Predict(list, 4)
	sorted, _ := Predict(deliveries(1, 4, 4, 4), 4)
	if res.DaysPerGallon != sorted.DaysPerGallon || !res.PredictedEmptyAt.Equal(sorted.PredictedEmptyAt) {
		t.Fatalf("order dependence: %+v vs %+v", res, sorted)
	}
}
