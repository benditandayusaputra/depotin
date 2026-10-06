package prediction

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	MaxDeliveries    = 7
	minGapDays       = 0.25
	outlierFactor    = 3.0
	outlierMinCount  = 4
	ewmaAlpha        = 0.4
	minDaysPerGallon = 0.5
	maxDaysPerGallon = 30.0
	highConfidenceCV = 0.35

	ConfidenceNone   = "none"
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

type Delivery struct {
	DeliveredAt time.Time
	RefillQty   int32
}

type Result struct {
	DaysPerGallon    float64
	Samples          int
	Confidence       string
	PredictedEmptyAt time.Time
	Explanation      string
}

func Predict(deliveries []Delivery, defaultDaysPerGallon float64) (Result, bool) {
	usable := usableDeliveries(deliveries)
	if len(usable) == 0 {
		return Result{Confidence: ConfidenceNone}, false
	}
	samples := gapSamples(usable)
	kept := dropOutliers(samples)
	estimate := shrink(ewma(kept), len(kept), defaultDaysPerGallon)
	if len(kept) == 0 {
		estimate = defaultDaysPerGallon
	}
	estimate = clamp(estimate, minDaysPerGallon, maxDaysPerGallon)

	last := usable[len(usable)-1]
	hours := float64(last.RefillQty) * estimate * 24
	res := Result{
		DaysPerGallon:    round2(estimate),
		Samples:          len(kept),
		Confidence:       confidence(kept),
		PredictedEmptyAt: last.DeliveredAt.Add(time.Duration(hours * float64(time.Hour))),
	}
	res.Explanation = explain(res, len(usable))
	return res, true
}

func usableDeliveries(deliveries []Delivery) []Delivery {
	out := make([]Delivery, 0, len(deliveries))
	for _, d := range deliveries {
		if d.RefillQty > 0 && !d.DeliveredAt.IsZero() {
			out = append(out, d)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DeliveredAt.Before(out[j].DeliveredAt) })
	if len(out) > MaxDeliveries {
		out = out[len(out)-MaxDeliveries:]
	}
	return out
}

func gapSamples(deliveries []Delivery) []float64 {
	samples := make([]float64, 0, len(deliveries))
	for i := 0; i+1 < len(deliveries); i++ {
		days := deliveries[i+1].DeliveredAt.Sub(deliveries[i].DeliveredAt).Hours() / 24
		if days < minGapDays {
			days = minGapDays
		}
		samples = append(samples, days/float64(deliveries[i].RefillQty))
	}
	return samples
}

func dropOutliers(samples []float64) []float64 {
	if len(samples) < outlierMinCount {
		return samples
	}
	med := median(samples)
	kept := make([]float64, 0, len(samples))
	for _, s := range samples {
		if s > med*outlierFactor || s < med/outlierFactor {
			continue
		}
		kept = append(kept, s)
	}
	return kept
}

func median(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func ewma(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	value := samples[0]
	for _, s := range samples[1:] {
		value = ewmaAlpha*s + (1-ewmaAlpha)*value
	}
	return value
}

func shrink(estimate float64, n int, fallback float64) float64 {
	return (float64(n)*estimate + fallback) / float64(n+1)
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func confidence(samples []float64) string {
	n := len(samples)
	switch {
	case n <= 1:
		return ConfidenceLow
	case n <= 3:
		return ConfidenceMedium
	}
	if coefficientOfVariation(samples) <= highConfidenceCV {
		return ConfidenceHigh
	}
	return ConfidenceMedium
}

func coefficientOfVariation(samples []float64) float64 {
	var sum float64
	for _, s := range samples {
		sum += s
	}
	mean := sum / float64(len(samples))
	if mean == 0 {
		return 0
	}
	var sq float64
	for _, s := range samples {
		sq += (s - mean) * (s - mean)
	}
	return math.Sqrt(sq/float64(len(samples))) / mean
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func explain(r Result, deliveries int) string {
	days := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", r.DaysPerGallon), "0"), ".")
	days = strings.ReplaceAll(days, ".", ",")
	if deliveries < 2 {
		return fmt.Sprintf("Belum ada pola, dipakai angka bawaan depot %s hari per galon.", days)
	}
	return fmt.Sprintf("Biasanya 1 galon habis dalam %s hari, dihitung dari %d pesanan terakhir.", days, deliveries)
}
