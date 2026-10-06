package customer

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/crypto"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
)

const (
	linkTokenBytes  = 16
	personalPath    = "/p/"
	whatsAppBaseURL = "https://wa.me/"
)

var ErrNoToken = errors.New("pelanggan belum punya token")

type Links struct {
	webOrigin string
	sealer    *crypto.Sealer
}

func NewLinks(webOrigin string, sealer *crypto.Sealer) *Links {
	return &Links{webOrigin: webOrigin, sealer: sealer}
}

func (l *Links) Issue(ctx context.Context, q *sqlc.Queries, depotID, customerID uuid.UUID, now time.Time) (string, error) {
	token, err := idgen.RandomToken(linkTokenBytes)
	if err != nil {
		return "", err
	}
	sealed, err := l.sealer.Seal([]byte(token))
	if err != nil {
		return "", err
	}
	rows, err := q.SetCustomerToken(ctx, sqlc.SetCustomerTokenParams{
		ID: customerID, DepotID: depotID, TokenHash: crypto.HashToken(token), TokenEnc: sealed, TokenRotatedAt: &now,
	})
	if err != nil {
		return "", fmt.Errorf("simpan token: %w", err)
	}
	if rows == 0 {
		return "", fmt.Errorf("simpan token: %w", ErrNoToken)
	}
	return token, nil
}

func (l *Links) Reveal(c sqlc.Customer) (string, error) {
	if len(c.TokenEnc) == 0 {
		return "", ErrNoToken
	}
	plain, err := l.sealer.Open(c.TokenEnc)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (l *Links) EnsureToken(ctx context.Context, q *sqlc.Queries, c sqlc.Customer, now time.Time) (string, error) {
	token, err := l.Reveal(c)
	if err == nil {
		return token, nil
	}
	if !errors.Is(err, ErrNoToken) {
		return "", err
	}
	return l.Issue(ctx, q, c.DepotID, c.ID, now)
}

func (l *Links) PersonalURL(token string) string {
	return l.webOrigin + personalPath + token
}

func (l *Links) PersonalURLWithReminder(token string, reminderID uuid.UUID) string {
	return l.PersonalURL(token) + "?r=" + reminderID.String()
}

func WhatsAppURL(phoneNumber, text string) string {
	return whatsAppBaseURL + phoneNumber + "?text=" + url.QueryEscape(text)
}

func ShareMessage(customerName, depotName, link string) string {
	return fmt.Sprintf("Halo %s, ini %s. Simpan link ini untuk pesan air sekali ketuk kapan saja: %s", customerName, depotName, link)
}
