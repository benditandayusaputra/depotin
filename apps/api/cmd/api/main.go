package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/benditandayusaputra/depotin/apps/api/internal/config"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
)

const shutdownTimeout = 15 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("api berhenti dengan galat", "err", err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("muat konfigurasi: %w", err)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("sambung basis data: %w", err)
	}
	defer pool.Close()

	app, err := newApp(dependencies{cfg: cfg, log: log, pool: pool})
	if err != nil {
		return fmt.Errorf("susun aplikasi: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(cfg.HTTPAddr, fiber.ListenConfig{DisableStartupMessage: true})
	}()
	log.Info("api berjalan", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	log.Info("menghentikan api")
	if err := app.ShutdownWithTimeout(shutdownTimeout); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
