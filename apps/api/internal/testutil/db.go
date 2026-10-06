package testutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/db"
	"github.com/benditandayusaputra/depotin/apps/api/internal/platform/idgen"
)

func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	baseURL := os.Getenv("DATABASE_URL")
	if baseURL == "" {
		t.Skip("DATABASE_URL kosong, tes integrasi dilewati")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	schema := "t_" + strings.ToLower(strings.ReplaceAll(idgen.NewID().String(), "-", ""))
	admin, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatalf("sambung basis data: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("buat skema uji: %v", err)
	}
	if err := admin.Close(ctx); err != nil {
		t.Fatalf("tutup koneksi admin: %v", err)
	}

	schemaURL := withSearchPath(baseURL, schema)
	if _, err := db.MigrateUp(ctx, schemaURL); err != nil {
		t.Fatalf("migrasi skema uji: %v", err)
	}
	pool, err := db.NewPool(ctx, schemaURL)
	if err != nil {
		t.Fatalf("pool uji: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelCleanup()
		conn, err := pgx.Connect(cleanupCtx, baseURL)
		if err != nil {
			return
		}
		_, _ = conn.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE")
		_ = conn.Close(cleanupCtx)
	})
	return pool
}

func withSearchPath(url, schema string) string {
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%ssearch_path=%s,public", url, sep, schema)
}
