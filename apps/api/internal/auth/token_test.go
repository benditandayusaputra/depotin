package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestIssueAndParse(t *testing.T) {
	issuer := NewTokenIssuer([]byte("secret-secret-secret-secret-secret"))
	now := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	userID, depotID, familyID := uuid.New(), uuid.New(), uuid.New()

	raw, err := issuer.Issue(userID, depotID, familyID, RoleOwner, now)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := issuer.Parse(raw, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID() != userID || claims.DepotID != depotID || claims.FamilyID != familyID || claims.Role != RoleOwner {
		t.Fatalf("claims mismatch: %+v", claims)
	}
	if _, err := issuer.Parse(raw, now.Add(AccessTokenTTL+time.Second)); err == nil {
		t.Fatal("expired token should fail")
	}
	other := NewTokenIssuer([]byte("another-secret-another-secret-another"))
	if _, err := other.Parse(raw, now); err == nil {
		t.Fatal("wrong secret should fail")
	}
}

func TestParseRejectsOtherAlgorithms(t *testing.T) {
	issuer := NewTokenIssuer([]byte("secret-secret-secret-secret-secret"))
	now := time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC)
	claims := Claims{DepotID: uuid.New(), Role: RoleOwner, FamilyID: uuid.New(), RegisteredClaims: jwt.RegisteredClaims{
		Subject: uuid.NewString(), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}}
	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Parse(none, now); err == nil {
		t.Fatal("alg none should be rejected")
	}
	hs512, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte("secret-secret-secret-secret-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Parse(hs512, now); err == nil {
		t.Fatal("HS512 should be rejected")
	}
}
