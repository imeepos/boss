package order

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPartnerTenantLockIntegration verifies FOR UPDATE blocks another transaction.
func TestPartnerTenantLockIntegration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var entityID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM legal_entities ORDER BY id LIMIT 1`).Scan(&entityID); err != nil {
		t.Fatal(err)
	}
	tx1, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := lockPartnerTenant(ctx, tx1, entityID); err != nil {
		t.Fatal(err)
	}
	tx2, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	locked := make(chan error, 1)
	go func() {
		locked <- lockPartnerTenant(ctx, tx2, entityID)
	}()
	select {
	case err := <-locked:
		_ = tx1.Rollback(ctx)
		_ = tx2.Rollback(ctx)
		t.Fatalf("second transaction acquired tenant lock early: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := tx1.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-locked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second transaction did not acquire lock after rollback")
	}
	if err := tx2.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
}
