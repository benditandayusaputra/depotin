package order

import (
	"math"
	"sort"
	"time"
)

const earthRadiusKm = 6371.0

type Stop struct {
	ID        string
	Status    string
	Lat       *float64
	Lng       *float64
	Area      string
	CreatedAt time.Time
}

func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}

func SortRoute(stops []Stop, depotLat, depotLng *float64) []Stop {
	var onDelivery, located, unlocated []Stop
	for _, s := range stops {
		switch {
		case s.Status == StatusOnDelivery:
			onDelivery = append(onDelivery, s)
		case s.Lat != nil && s.Lng != nil && depotLat != nil && depotLng != nil:
			located = append(located, s)
		default:
			unlocated = append(unlocated, s)
		}
	}
	byAreaThenTime(onDelivery)
	byAreaThenTime(unlocated)
	out := make([]Stop, 0, len(stops))
	out = append(out, onDelivery...)
	if depotLat != nil && depotLng != nil {
		out = append(out, nearestNeighbour(located, *depotLat, *depotLng)...)
	}
	return append(out, unlocated...)
}

func nearestNeighbour(stops []Stop, startLat, startLng float64) []Stop {
	remaining := append([]Stop(nil), stops...)
	ordered := make([]Stop, 0, len(stops))
	curLat, curLng := startLat, startLng
	for len(remaining) > 0 {
		best := 0
		bestDist := math.Inf(1)
		for i, s := range remaining {
			d := Haversine(curLat, curLng, *s.Lat, *s.Lng)
			if d < bestDist || (d == bestDist && remaining[i].CreatedAt.Before(remaining[best].CreatedAt)) {
				best, bestDist = i, d
			}
		}
		next := remaining[best]
		ordered = append(ordered, next)
		curLat, curLng = *next.Lat, *next.Lng
		remaining = append(remaining[:best], remaining[best+1:]...)
	}
	return ordered
}

func byAreaThenTime(stops []Stop) {
	sort.SliceStable(stops, func(i, j int) bool {
		if stops[i].Area != stops[j].Area {
			return stops[i].Area < stops[j].Area
		}
		return stops[i].CreatedAt.Before(stops[j].CreatedAt)
	})
}
