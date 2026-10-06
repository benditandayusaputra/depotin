package order

import (
	"testing"
	"time"
)

func f(v float64) *float64 { return &v }

func ids(stops []Stop) []string {
	out := make([]string, 0, len(stops))
	for _, s := range stops {
		out = append(out, s.ID)
	}
	return out
}

func equalIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestHaversineJakarta(t *testing.T) {
	d := Haversine(-6.2088, 106.8456, -6.9175, 107.6191)
	if d < 115 || d > 130 {
		t.Fatalf("Jakarta to Bandung should be about 120 km, got %.1f", d)
	}
	if Haversine(1, 1, 1, 1) != 0 {
		t.Fatal("same point should be zero")
	}
}

func TestSortRouteNearestNeighbourWithAreasAfter(t *testing.T) {
	t0 := time.Date(2026, 10, 6, 7, 0, 0, 0, time.UTC)
	stops := []Stop{
		{ID: "far", Status: StatusConfirmed, Lat: f(-6.30), Lng: f(106.85), CreatedAt: t0},
		{ID: "near", Status: StatusConfirmed, Lat: f(-6.21), Lng: f(106.85), CreatedAt: t0.Add(time.Minute)},
		{ID: "mid", Status: StatusConfirmed, Lat: f(-6.25), Lng: f(106.85), CreatedAt: t0.Add(2 * time.Minute)},
		{ID: "rt05-late", Status: StatusConfirmed, Area: "RT 05", CreatedAt: t0.Add(5 * time.Minute)},
		{ID: "rt03", Status: StatusConfirmed, Area: "RT 03", CreatedAt: t0.Add(6 * time.Minute)},
		{ID: "rt05-early", Status: StatusConfirmed, Area: "RT 05", CreatedAt: t0.Add(4 * time.Minute)},
		{ID: "going", Status: StatusOnDelivery, Lat: f(-6.40), Lng: f(106.85), CreatedAt: t0.Add(9 * time.Minute)},
	}
	got := ids(SortRoute(stops, f(-6.20), f(106.85)))
	want := []string{"going", "near", "mid", "far", "rt03", "rt05-early", "rt05-late"}
	if !equalIDs(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSortRouteWithoutDepotCoordinates(t *testing.T) {
	t0 := time.Date(2026, 10, 6, 7, 0, 0, 0, time.UTC)
	stops := []Stop{
		{ID: "b", Status: StatusConfirmed, Lat: f(-6.21), Lng: f(106.85), Area: "B", CreatedAt: t0},
		{ID: "a2", Status: StatusConfirmed, Area: "A", CreatedAt: t0.Add(time.Minute)},
		{ID: "a1", Status: StatusConfirmed, Area: "A", CreatedAt: t0},
	}
	got := ids(SortRoute(stops, nil, nil))
	if !equalIDs(got, []string{"a1", "a2", "b"}) {
		t.Fatalf("got %v", got)
	}
}
