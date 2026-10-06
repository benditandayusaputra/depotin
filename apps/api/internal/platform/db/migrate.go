package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	dbassets "github.com/benditandayusaputra/depotin/apps/api/db"
)

type MigrationResult struct {
	Version int64
	State   string
}

func openMigrationDB(url string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("buka koneksi migrasi: %w", err)
	}
	return conn, nil
}

func newProvider(conn *sql.DB) (*goose.Provider, error) {
	migrations, err := fs.Sub(dbassets.Migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("baca berkas migrasi: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, conn, migrations)
	if err != nil {
		return nil, fmt.Errorf("siapkan goose: %w", err)
	}
	return provider, nil
}

func MigrateUp(ctx context.Context, url string) ([]MigrationResult, error) {
	conn, err := openMigrationDB(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	provider, err := newProvider(conn)
	if err != nil {
		return nil, err
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrasi naik: %w", err)
	}
	out := make([]MigrationResult, 0, len(results))
	for _, r := range results {
		out = append(out, MigrationResult{Version: r.Source.Version, State: "applied"})
	}
	return out, nil
}

func MigrateDown(ctx context.Context, url string) (MigrationResult, error) {
	conn, err := openMigrationDB(url)
	if err != nil {
		return MigrationResult{}, err
	}
	defer func() { _ = conn.Close() }()
	provider, err := newProvider(conn)
	if err != nil {
		return MigrationResult{}, err
	}
	result, err := provider.Down(ctx)
	if err != nil {
		return MigrationResult{}, fmt.Errorf("migrasi turun: %w", err)
	}
	return MigrationResult{Version: result.Source.Version, State: "rolled_back"}, nil
}

func MigrationStatus(ctx context.Context, url string) ([]MigrationResult, error) {
	conn, err := openMigrationDB(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	provider, err := newProvider(conn)
	if err != nil {
		return nil, err
	}
	statuses, err := provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("status migrasi: %w", err)
	}
	out := make([]MigrationResult, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, MigrationResult{Version: s.Source.Version, State: string(s.State)})
	}
	return out, nil
}
