package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("migrasi gagal", "err", err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	command := "up"
	if len(args) > 0 {
		command = args[0]
	}
	url := os.Getenv("MIGRATE_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		return errors.New("MIGRATE_DATABASE_URL atau DATABASE_URL wajib diisi")
	}
	ctx := context.Background()

	switch command {
	case "up":
		results, err := db.MigrateUp(ctx, url)
		if err != nil {
			return fmt.Errorf("up: %w", err)
		}
		for _, r := range results {
			slog.Info("migrasi diterapkan", "version", r.Version)
		}
	case "down":
		result, err := db.MigrateDown(ctx, url)
		if err != nil {
			return fmt.Errorf("down: %w", err)
		}
		slog.Info("migrasi dibatalkan", "version", result.Version)
	case "status":
		results, err := db.MigrationStatus(ctx, url)
		if err != nil {
			return fmt.Errorf("status: %w", err)
		}
		for _, r := range results {
			slog.Info("status migrasi", "version", r.Version, "state", r.State)
		}
	default:
		return fmt.Errorf("perintah %q tidak dikenal, pakai up, down, atau status", command)
	}
	return nil
}
