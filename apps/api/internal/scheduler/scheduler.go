package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/db/sqlc"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/clock"
	"github.com/benditandayusaputra/depotin/apps/api/internal/reminder"
)

const (
	tickEvery         = 30 * time.Second
	reminderHour      = 6
	keepaliveInterval = 4 * time.Minute
	taskTimeout       = 2 * time.Minute
)

type Config struct {
	Location       *time.Location
	DemoResetHour  int
	KeepaliveUntil time.Time
}

type Scheduler struct {
	pool      *pgxpool.Pool
	q         *sqlc.Queries
	clock     clock.Clock
	log       *slog.Logger
	reminders *reminder.Service
	resetDemo func(ctx context.Context) error
	cfg       Config

	nextReminders time.Time
	nextDemoReset time.Time
	nextKeepalive time.Time
}

func New(pool *pgxpool.Pool, clk clock.Clock, log *slog.Logger, reminders *reminder.Service, resetDemo func(ctx context.Context) error, cfg Config) *Scheduler {
	if cfg.Location == nil {
		cfg.Location = time.FixedZone("WIB", 7*3600)
	}
	now := clk.Now()
	s := &Scheduler{pool: pool, q: sqlc.New(pool), clock: clk, log: log, reminders: reminders, resetDemo: resetDemo, cfg: cfg}
	s.nextReminders = nextDaily(now, reminderHour, cfg.Location)
	if cfg.DemoResetHour >= 0 && resetDemo != nil {
		s.nextDemoReset = nextDaily(now, cfg.DemoResetHour, cfg.Location)
	}
	if now.Before(cfg.KeepaliveUntil) {
		s.nextKeepalive = now
	}
	return s
}

func nextDaily(now time.Time, hour int, loc *time.Location) time.Time {
	local := now.In(loc)
	next := time.Date(local.Year(), local.Month(), local.Day(), hour, 0, 0, 0, loc)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Tick(ctx)
		}
	}
}

func (s *Scheduler) Tick(ctx context.Context) {
	now := s.clock.Now()
	if !s.nextReminders.IsZero() && !now.Before(s.nextReminders) {
		s.nextReminders = nextDaily(now, reminderHour, s.cfg.Location)
		s.runTask(ctx, "antrean pengingat", s.refreshAllReminders)
	}
	if !s.nextDemoReset.IsZero() && !now.Before(s.nextDemoReset) {
		s.nextDemoReset = nextDaily(now, s.cfg.DemoResetHour, s.cfg.Location)
		s.runTask(ctx, "setel ulang demo", s.resetDemo)
	}
	if !s.nextKeepalive.IsZero() && !now.Before(s.nextKeepalive) {
		if now.Before(s.cfg.KeepaliveUntil) {
			s.nextKeepalive = now.Add(keepaliveInterval)
			s.runTask(ctx, "penjaga hidup", s.keepalive)
		} else {
			s.nextKeepalive = time.Time{}
		}
	}
}

func (s *Scheduler) runTask(ctx context.Context, name string, task func(ctx context.Context) error) {
	taskCtx, cancel := context.WithTimeout(ctx, taskTimeout)
	defer cancel()
	start := s.clock.Now()
	if err := task(taskCtx); err != nil {
		s.log.ErrorContext(ctx, "tugas terjadwal gagal", "task", name, "err", err.Error())
		return
	}
	s.log.InfoContext(ctx, "tugas terjadwal selesai", "task", name, "duration_ms", s.clock.Now().Sub(start).Milliseconds())
}

func (s *Scheduler) refreshAllReminders(ctx context.Context) error {
	depots, err := s.q.ListDepots(ctx)
	if err != nil {
		return fmt.Errorf("daftar depot: %w", err)
	}
	for _, d := range depots {
		if _, err := s.reminders.Refresh(ctx, d); err != nil {
			return fmt.Errorf("depot %s: %w", d.Slug, err)
		}
	}
	return nil
}

func (s *Scheduler) keepalive(ctx context.Context) error {
	var one int
	if err := s.pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("select 1: %w", err)
	}
	return nil
}

func (s *Scheduler) NextRuns() (reminders, demoReset, keepalive time.Time) {
	return s.nextReminders, s.nextDemoReset, s.nextKeepalive
}
