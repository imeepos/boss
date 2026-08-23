package partner

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPartnerCommissionLedgerIntegration(t *testing.T) {
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
	var orderID, entityID, accountID int64
	if err := pool.QueryRow(ctx, `SELECT o.id, o.legal_entity_id FROM orders o WHERE o.channel_id IN (SELECT id FROM channels WHERE code='AGENT') ORDER BY o.id DESC LIMIT 1`).Scan(&orderID, &entityID); err != nil {
		t.Skipf("没有可用 AGENT 订单: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM accounts WHERE legal_entity_id=$1 AND role_id=(SELECT id FROM roles WHERE code='partner_admin') LIMIT 1`, entityID).Scan(&accountID); err != nil {
		t.Skipf("没有伙伴管理员: %v", err)
	}
	store := NewPGStore(pool)
	ledgerID, err := store.AccrueCommission(ctx, orderID, entityID, 1000, 0.10)
	if err != nil {
		t.Fatal(err)
	}
	if ledgerID == 0 {
		t.Fatal("ledger id is zero")
	}
	if _, err := store.AccrueCommission(ctx, orderID, entityID, 1200, 0.10); err != nil {
		t.Fatal(err)
	}
	var amount float64
	if err := pool.QueryRow(ctx, `SELECT order_amount FROM partner_commission_ledger WHERE id=$1`, ledgerID).Scan(&amount); err != nil {
		t.Fatal(err)
	}
	if amount != 1200 {
		t.Fatalf("amount=%v want=1200", amount)
	}
	if err := store.SettleCommissionLedger(ctx, ledgerID, accountID); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM partner_commission_ledger WHERE id=$1`, ledgerID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != CommissionSettled {
		t.Fatalf("status=%s", status)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM partner_commission_ledger WHERE id=$1`, ledgerID)
}
