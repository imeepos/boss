package partner

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

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
	suffix := strconv.FormatInt(time.Now().UnixNano()%1000000000000, 10)
	var entityID, accountID, channelID, orderID, ledgerID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM legal_entities ORDER BY id LIMIT 1`).Scan(&entityID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM accounts WHERE legal_entity_id=$1 AND role_id=(SELECT id FROM roles WHERE code='partner_admin') LIMIT 1`, entityID).Scan(&accountID); err != nil {
		t.Skipf("没有伙伴管理员: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO channels(code,name,status) VALUES($1,'E2E代理','ACTIVE') RETURNING id`, "AGENT-E2E-"+suffix).Scan(&channelID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO orders(order_no,customer_id,offer_id,address_id,stage,status,channel_id,legal_entity_id) VALUES($1,1,1,1,12,'DONE',$2,$3) RETURNING id`, "ORD-E2E-COM-"+suffix, channelID, entityID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM partner_commission_ledger WHERE id=$1`, ledgerID)
		_, _ = pool.Exec(ctx, `DELETE FROM orders WHERE id=$1`, orderID)
		_, _ = pool.Exec(ctx, `DELETE FROM channels WHERE id=$1`, channelID)
	}()
	store := NewPGStore(pool)
	ledgerID, err = store.AccrueCommission(ctx, orderID, entityID, 1000, 0.10)
	if err != nil {
		t.Fatal(err)
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
}
