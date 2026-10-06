package main

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/benditandayusaputra/depotin/apps/api/internal/auth"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

func TestRegisterCreatesDepotOwnerAndDefaultProduct(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	s := c.register("Depot Tirta Sejuk", "0812-1111-0001")
	if s.User.Role != auth.RoleOwner || s.User.Phone != "6281211110001" || s.Depot.Slug != "depot-tirta-sejuk" {
		t.Fatalf("unexpected session %+v", s)
	}
	if _, ok := c.cookies["dp_at"]; !ok {
		t.Fatal("access cookie missing")
	}
	if _, ok := c.cookies["dp_rt"]; !ok {
		t.Fatal("refresh cookie missing")
	}
	var count int
	if err := h.deps.pool.QueryRow(context.Background(), "SELECT count(*) FROM products WHERE depot_id = $1 AND kind = 'refill'", s.Depot.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("default product count %d", count)
	}
	me := c.mustStatus(c.do(http.MethodGet, "/api/v1/auth/me", nil), http.StatusOK)
	var meData sessionData
	me.data(t, &meData)
	if meData.User.ID != s.User.ID {
		t.Fatal("me returned different user")
	}

	second := h.client().register("Depot Tirta Sejuk", "0812-1111-0002")
	if second.Depot.Slug != "depot-tirta-sejuk-2" {
		t.Fatalf("slug collision not handled: %s", second.Depot.Slug)
	}
	dup := c.do(http.MethodPost, "/api/v1/auth/register", map[string]any{
		"depot_name": "Lain", "name": "Orang", "phone": "081211110001", "password": testPassword,
	})
	c.mustStatus(dup, http.StatusUnprocessableEntity)
}

func TestRegisterValidation(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	res := c.do(http.MethodPost, "/api/v1/auth/register", map[string]any{
		"depot_name": "AB", "name": "X", "phone": "123", "password": "pendek",
	})
	c.mustStatus(res, http.StatusUnprocessableEntity)
	if res.errorCode(t) != httpx.CodeValidationFailed {
		t.Fatal("expected validation_failed")
	}
	unknown := c.do(http.MethodPost, "/api/v1/auth/register", map[string]any{
		"depot_name": "Depot Baru", "name": "Pemilik", "phone": "081211110009", "password": testPassword, "role": "owner",
	})
	c.mustStatus(unknown, http.StatusUnprocessableEntity)
}

func TestLoginLockoutAfterFiveFailures(t *testing.T) {
	h := newHarness(t)
	owner := h.client()
	owner.register("Depot Kunci", "081211110010")

	for i := 1; i <= 5; i++ {
		c := h.client()
		c.ip = "198.51.100." + strconv.Itoa(i)
		want := http.StatusUnauthorized
		if i == auth.MaxFailedLogins {
			want = http.StatusConflict
		}
		c.mustStatus(c.login("081211110010", "salah-"+strconv.Itoa(i)), want)
	}
	locked := h.client()
	locked.ip = "198.51.100.99"
	res := locked.login("081211110010", testPassword)
	locked.mustStatus(res, http.StatusConflict)

	h.clock.Advance(auth.LockDuration + time.Second)
	after := h.client()
	after.ip = "198.51.100.98"
	after.mustStatus(after.login("081211110010", testPassword), http.StatusOK)
}

func TestLoginRateLimitPerIPAndPhone(t *testing.T) {
	h := newHarness(t)
	h.client().register("Depot Laju", "081211110020")
	c := h.client()
	for i := 1; i <= 5; i++ {
		res := c.login("081211110020", "salah")
		if res.Status != http.StatusUnauthorized && res.Status != http.StatusConflict {
			t.Fatalf("attempt %d status %d", i, res.Status)
		}
	}
	res := c.login("081211110020", "salah")
	c.mustStatus(res, http.StatusTooManyRequests)
	if res.Header.Get(fiber.HeaderRetryAfter) == "" {
		t.Fatal("retry-after missing")
	}
	h.clock.Advance(auth.LockDuration + time.Second)
	other := h.client()
	other.ip = "198.51.100.50"
	other.mustStatus(other.login("081211110020", testPassword), http.StatusOK)
}

func TestLoginDoesNotRevealUnknownPhone(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	res := c.login("081299999999", "apa-saja")
	c.mustStatus(res, http.StatusUnauthorized)
	if res.errorCode(t) != httpx.CodeUnauthenticated {
		t.Fatal("expected unauthenticated")
	}
}

func TestRefreshRotatesAndDetectsReuse(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	c.register("Depot Rotasi", "081211110030")
	oldRefresh := c.cookies["dp_rt"]
	oldAccess := c.cookies["dp_at"]

	h.clock.Advance(time.Minute)
	c.mustStatus(c.do(http.MethodPost, "/api/v1/auth/refresh", nil), http.StatusOK)
	newRefresh := c.cookies["dp_rt"]
	if newRefresh == oldRefresh || c.cookies["dp_at"] == oldAccess {
		t.Fatal("tokens were not rotated")
	}
	c.mustStatus(c.do(http.MethodGet, "/api/v1/auth/me", nil), http.StatusOK)

	attacker := h.client()
	attacker.cookies["dp_rt"] = oldRefresh
	attacker.mustStatus(attacker.do(http.MethodPost, "/api/v1/auth/refresh", nil), http.StatusUnauthorized)

	victim := h.client()
	victim.cookies["dp_rt"] = newRefresh
	victim.mustStatus(victim.do(http.MethodPost, "/api/v1/auth/refresh", nil), http.StatusUnauthorized)
}

func TestExpiredAccessTokenNeedsRefresh(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	c.register("Depot Kedaluwarsa", "081211110040")
	h.clock.Advance(auth.AccessTokenTTL + time.Minute)
	c.mustStatus(c.do(http.MethodGet, "/api/v1/auth/me", nil), http.StatusUnauthorized)
	c.mustStatus(c.do(http.MethodPost, "/api/v1/auth/refresh", nil), http.StatusOK)
	c.mustStatus(c.do(http.MethodGet, "/api/v1/auth/me", nil), http.StatusOK)
}

func TestLogoutAllRevokesEveryDevice(t *testing.T) {
	h := newHarness(t)
	phone := "081211110050"
	first := h.client()
	first.register("Depot Keluar", phone)
	second := h.client()
	second.mustStatus(second.login(phone, testPassword), http.StatusOK)

	first.mustStatus(first.do(http.MethodPost, "/api/v1/auth/logout-all", nil), http.StatusOK)
	if _, ok := first.cookies["dp_at"]; ok {
		t.Fatal("cookies should be cleared")
	}
	second.mustStatus(second.do(http.MethodPost, "/api/v1/auth/refresh", nil), http.StatusUnauthorized)
}

func TestLogoutRevokesOnlyCurrentFamily(t *testing.T) {
	h := newHarness(t)
	phone := "081211110055"
	first := h.client()
	first.register("Depot Keluar Satu", phone)
	second := h.client()
	second.mustStatus(second.login(phone, testPassword), http.StatusOK)
	first.mustStatus(first.do(http.MethodPost, "/api/v1/auth/logout", nil), http.StatusOK)
	second.mustStatus(second.do(http.MethodPost, "/api/v1/auth/refresh", nil), http.StatusOK)
}

func TestChangePassword(t *testing.T) {
	h := newHarness(t)
	phone := "081211110060"
	c := h.client()
	c.register("Depot Sandi", phone)
	wrong := c.do(http.MethodPost, "/api/v1/auth/password", map[string]any{"current_password": "bukan", "new_password": "sandi-baru-panjang"})
	c.mustStatus(wrong, http.StatusUnprocessableEntity)
	c.mustStatus(c.do(http.MethodPost, "/api/v1/auth/password", map[string]any{"current_password": testPassword, "new_password": "sandi-baru-panjang"}), http.StatusOK)
	c.mustStatus(c.do(http.MethodGet, "/api/v1/auth/me", nil), http.StatusOK)
	fresh := h.client()
	fresh.mustStatus(fresh.login(phone, testPassword), http.StatusUnauthorized)
	fresh.mustStatus(fresh.login(phone, "sandi-baru-panjang"), http.StatusOK)
}

func TestOriginCheckOnMutations(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	body := map[string]any{"phone": "081211110070", "password": testPassword}
	noOrigin := c.do(http.MethodPost, "/api/v1/auth/login", body, func(r *http.Request) { r.Header.Del(fiber.HeaderOrigin) })
	c.mustStatus(noOrigin, http.StatusForbidden)
	badOrigin := c.do(http.MethodPost, "/api/v1/auth/login", body, func(r *http.Request) { r.Header.Set(fiber.HeaderOrigin, "https://evil.example") })
	c.mustStatus(badOrigin, http.StatusForbidden)
	badType := c.do(http.MethodPost, "/api/v1/auth/login", body, func(r *http.Request) { r.Header.Set(fiber.HeaderContentType, "text/plain") })
	c.mustStatus(badType, http.StatusUnprocessableEntity)
}

func TestCourierCannotReachOwnerRoutes(t *testing.T) {
	h := newHarness(t)
	owner := h.client()
	owner.register("Depot Peran", "081211110080")
	owner.createCourier("Kurir Satu", "081211110081")

	courier := h.client()
	courier.mustStatus(courier.login("081211110081", testPassword), http.StatusOK)
	for _, path := range []string{"/api/v1/depot", "/api/v1/users"} {
		res := courier.do(http.MethodGet, path, nil)
		courier.mustStatus(res, http.StatusForbidden)
	}
	courier.mustStatus(courier.do(http.MethodPatch, "/api/v1/depot", map[string]any{"name": "Diubah Kurir"}), http.StatusForbidden)
	courier.mustStatus(courier.do(http.MethodGet, "/api/v1/auth/me", nil), http.StatusOK)

	anon := h.client()
	anon.mustStatus(anon.do(http.MethodGet, "/api/v1/depot", nil), http.StatusUnauthorized)
}

func TestUsersAreIsolatedPerDepot(t *testing.T) {
	h := newHarness(t)
	a := h.client()
	a.register("Depot A", "081211110090")
	b := h.client()
	b.register("Depot B", "081211110091")
	courierB := b.createCourier("Kurir B", "081211110092")

	a.mustStatus(a.do(http.MethodPatch, "/api/v1/users/"+courierB, map[string]any{"name": "Dibajak"}), http.StatusNotFound)
	a.mustStatus(a.do(http.MethodPost, "/api/v1/users/"+courierB+"/reset-password", map[string]any{"password": "sandi-dibajak-123"}), http.StatusNotFound)

	list := a.mustStatus(a.do(http.MethodGet, "/api/v1/users", nil), http.StatusOK)
	var users []map[string]any
	list.data(t, &users)
	if len(users) != 1 {
		t.Fatalf("depot A should only see itself, got %d", len(users))
	}
	dup := a.do(http.MethodPost, "/api/v1/users", map[string]any{"name": "Kurir Ganda", "phone": "081211110092", "password": testPassword})
	a.mustStatus(dup, http.StatusUnprocessableEntity)
}

func TestDeactivatedCourierLosesAccess(t *testing.T) {
	h := newHarness(t)
	owner := h.client()
	owner.register("Depot Nonaktif", "081211110100")
	courierID := owner.createCourier("Kurir", "081211110101")
	courier := h.client()
	courier.mustStatus(courier.login("081211110101", testPassword), http.StatusOK)

	owner.mustStatus(owner.do(http.MethodPatch, "/api/v1/users/"+courierID, map[string]any{"is_active": false}), http.StatusOK)
	courier.mustStatus(courier.do(http.MethodPost, "/api/v1/auth/refresh", nil), http.StatusUnauthorized)
	again := h.client()
	again.mustStatus(again.login("081211110101", testPassword), http.StatusUnauthorized)

	owner.mustStatus(owner.do(http.MethodPost, "/api/v1/users/"+courierID+"/reset-password", map[string]any{"password": "sandi-reset-baru-1"}), http.StatusOK)
	owner.mustStatus(owner.do(http.MethodPatch, "/api/v1/users/"+courierID, map[string]any{"is_active": true}), http.StatusOK)
	again.mustStatus(again.login("081211110101", "sandi-reset-baru-1"), http.StatusOK)
}

func TestDepotUpdate(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	c.register("Depot Ubah", "081211110110")
	res := c.mustStatus(c.do(http.MethodPatch, "/api/v1/depot", map[string]any{
		"open_time": "08:30", "close_time": "21:00", "delivery_fee": 2000, "loyalty_every": 10, "phone": "0813-0000-1111", "default_days_per_gallon": 3.5,
	}), http.StatusOK)
	var d struct {
		OpenTime     string  `json:"open_time"`
		CloseTime    string  `json:"close_time"`
		DeliveryFee  int64   `json:"delivery_fee"`
		LoyaltyEvery *int32  `json:"loyalty_every"`
		Phone        string  `json:"phone"`
		DaysPerGal   float64 `json:"default_days_per_gallon"`
	}
	res.data(t, &d)
	if d.OpenTime != "08:30" || d.CloseTime != "21:00" || d.DeliveryFee != 2000 || d.LoyaltyEvery == nil || *d.LoyaltyEvery != 10 || d.Phone != "6281300001111" || d.DaysPerGal != 3.5 {
		t.Fatalf("unexpected depot %+v", d)
	}
	off := c.mustStatus(c.do(http.MethodPatch, "/api/v1/depot", map[string]any{"loyalty_every": 0}), http.StatusOK)
	off.data(t, &d)
	if d.LoyaltyEvery != nil {
		t.Fatal("loyalty should be off")
	}
	c.mustStatus(c.do(http.MethodPatch, "/api/v1/depot", map[string]any{"open_time": "25:00"}), http.StatusUnprocessableEntity)
}
