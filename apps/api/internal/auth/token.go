package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
	RoleOwner       = "owner"
	RoleCourier     = "courier"
)

var ErrInvalidToken = errors.New("token tidak valid")

type Claims struct {
	DepotID  uuid.UUID `json:"did"`
	Role     string    `json:"role"`
	FamilyID uuid.UUID `json:"fid"`
	jwt.RegisteredClaims
}

type TokenIssuer struct {
	secret []byte
}

func NewTokenIssuer(secret []byte) *TokenIssuer {
	return &TokenIssuer{secret: secret}
}

func (t *TokenIssuer) Issue(userID, depotID, familyID uuid.UUID, role string, now time.Time) (string, error) {
	claims := Claims{
		DepotID:  depotID,
		Role:     role,
		FamilyID: familyID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("tanda tangan token: %w", err)
	}
	return signed, nil
}

func (t *TokenIssuer) Parse(raw string, now time.Time) (Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) {
		return t.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(func() time.Time { return now }),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.Role != RoleOwner && claims.Role != RoleCourier {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

func (c Claims) UserID() uuid.UUID {
	id, _ := uuid.Parse(c.Subject)
	return id
}
