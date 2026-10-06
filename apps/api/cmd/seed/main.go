package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/benditandayusaputra/depotin/apps/api/internal/config"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/seed"
)

func main() {
	if err := run(); err != nil {
		slog.Error("seed gagal", "err", err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("muat konfigurasi: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("sambung basis data: %w", err)
	}
	defer pool.Close()
	start := time.Now()
	if err := seed.Run(ctx, pool, time.Now(), cfg.WebOrigin, cfg.LinkEncKey); err != nil {
		return err
	}
	slog.Info("data contoh siap", "depot", seed.DepotSlug, "owner_phone", seed.OwnerPhone, "duration_ms", time.Since(start).Milliseconds())
	return nil
}
