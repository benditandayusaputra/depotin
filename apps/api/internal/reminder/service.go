package reminder

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/customer"
	"github.com/benditandayusaputra/depotin/apps/api/internal/order"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/httpx"
)

const (
	StatusQueued  = "queued"
	StatusSent    = "sent"
	StatusOrdered = "ordered"
	StatusSkipped = "skipped"
	StatusExpired = "expired"

	recentDays = 3
	listLimit  = 200
)

type Notifier interface {
	ReminderQueued(depotID uuid.UUID, count int64)
}

type nopNotifier struct{}

func (nopNotifier) ReminderQueued(uuid.UUID, int64) {}

type Service struct {
	pool     *pgxpool.Pool
	q        *sqlc.Queries
	clock    clock.Clock
	links    *customer.Links
	notifier Notifier
}

func NewService(pool *pgxpool.Pool, clk clock.Clock, links *customer.Links, notifier Notifier) *Service {
	if notifier == nil {
		notifier = nopNotifier{}
	}
	return &Service{pool: pool, q: sqlc.New(pool), clock: clk, links: links, notifier: notifier}
}

func (s *Service) Refresh(ctx context.Context, depot sqlc.Depot) (int64, error) {
	loc := order.DepotLocation(depot)
	now := s.clock.Now()
	today := order.DateIn(now, loc)
	startOfToday := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	dueBefore := startOfToday.AddDate(0, 0, int(depot.ReminderLeadDays)+1)

	var queued int64
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if _, err := q.ExpireStaleReminders(ctx, sqlc.ExpireStaleRemindersParams{DepotID: depot.ID, BeforeDate: today.AddDate(0, 0, -recentDays)}); err != nil {
			return fmt.Errorf("kedaluwarsakan pengingat: %w", err)
		}
		var err error
		queued, err = q.QueueReminders(ctx, sqlc.QueueRemindersParams{
			DepotID: depot.ID, DueDate: today, DueBefore: dueBefore, RecentAfter: today.AddDate(0, 0, -recentDays),
		})
		if err != nil {
			return fmt.Errorf("susun antrean pengingat: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if queued > 0 {
		s.notifier.ReminderQueued(depot.ID, queued)
	}
	return queued, nil
}

type Item struct {
	Reminder sqlc.Reminder
	Customer sqlc.Customer
}

func (s *Service) List(ctx context.Context, depot sqlc.Depot) ([]Item, error) {
	today := order.DateIn(s.clock.Now(), order.DepotLocation(depot))
	rows, err := s.q.ListReminders(ctx, sqlc.ListRemindersParams{
		DepotID: depot.ID, Limit: listLimit, Statuses: []string{StatusQueued, StatusSent}, FromDate: today.AddDate(0, 0, -recentDays),
	})
	if err != nil {
		return nil, fmt.Errorf("daftar pengingat: %w", err)
	}
	items := make([]Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, Item{Reminder: r.Reminder, Customer: r.Customer})
	}
	return items, nil
}

type Sent struct {
	Reminder sqlc.Reminder
	Customer sqlc.Customer
	Link     string
	WaURL    string
}

func (s *Service) Send(ctx context.Context, depot sqlc.Depot, reminderID, userID uuid.UUID) (Sent, error) {
	now := s.clock.Now()
	var out Sent
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		r, err := q.MarkReminderSent(ctx, sqlc.MarkReminderSentParams{ID: reminderID, DepotID: depot.ID, SentAt: &now, SentBy: &userID})
		if err != nil {
			if db.IsNoRows(err) {
				return s.explainMissing(ctx, depot.ID, reminderID)
			}
			return fmt.Errorf("tandai pengingat terkirim: %w", err)
		}
		cust, err := q.GetCustomerForUpdate(ctx, sqlc.GetCustomerForUpdateParams{ID: r.CustomerID, DepotID: depot.ID})
		if err != nil {
			return fmt.Errorf("kunci pelanggan: %w", err)
		}
		token, err := s.links.EnsureToken(ctx, q, cust, now)
		if err != nil {
			return err
		}
		link := s.links.PersonalURLWithReminder(token, r.ID)
		out = Sent{Reminder: r, Customer: cust, Link: link, WaURL: customer.WhatsAppURL(cust.Phone, Message(cust.Name, depot.Name, link))}
		return nil
	})
	return out, err
}

func (s *Service) Skip(ctx context.Context, depotID, reminderID uuid.UUID) (sqlc.Reminder, error) {
	r, err := s.q.SkipReminder(ctx, sqlc.SkipReminderParams{ID: reminderID, DepotID: depotID})
	if err != nil {
		if db.IsNoRows(err) {
			return sqlc.Reminder{}, s.explainMissing(ctx, depotID, reminderID)
		}
		return sqlc.Reminder{}, fmt.Errorf("lewati pengingat: %w", err)
	}
	return r, nil
}

func (s *Service) explainMissing(ctx context.Context, depotID, reminderID uuid.UUID) error {
	if _, err := s.q.GetReminder(ctx, sqlc.GetReminderParams{ID: reminderID, DepotID: depotID}); err != nil {
		if db.IsNoRows(err) {
			return httpx.NotFound("Pengingat tidak ditemukan.")
		}
		return fmt.Errorf("ambil pengingat: %w", err)
	}
	return httpx.InvalidTransition("Pengingat sudah diproses.")
}

func (s *Service) QueuedCount(ctx context.Context, depot sqlc.Depot) (int64, error) {
	today := order.DateIn(s.clock.Now(), order.DepotLocation(depot))
	n, err := s.q.CountQueuedReminders(ctx, sqlc.CountQueuedRemindersParams{DepotID: depot.ID, FromDate: today.AddDate(0, 0, -recentDays)})
	if err != nil {
		return 0, fmt.Errorf("hitung antrean pengingat: %w", err)
	}
	return n, nil
}

func Message(customerName, depotName, link string) string {
	return fmt.Sprintf("Halo %s, ini %s. Perkiraan kami air galon di rumah hampir habis. Mau diantar hari ini? Pesan sekali ketuk di sini: %s", customerName, depotName, link)
}
