package app_test

// PORT MVP 验收集辅件:连接池、专用清理、送积分规则种子。

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ymm-001/boss/internal/domain/loy"
	"github.com/ymm-001/boss/internal/pkg/config"
)

// portalMvpPool 打开独立连接池(cleanup 先于 app.Close 排序由 t.Cleanup LIFO 保证)。
func portalMvpPool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// portalMvpCleanup 清理门户侧残留:门户账号、积分账本、券与模板、发票。
// 订单/账单/客户/工单由 seedE2E 的统一清理覆盖。
func portalMvpCleanup(t *testing.T, ctx context.Context, pool *pgxpool.Pool, phone string, custID int64) {
	t.Helper()
	type stmt struct {
		q    string
		args []any
	}
	stmts := []stmt{
		{`DELETE FROM portal_accounts WHERE phone = $1`, []any{phone}},
		{`DELETE FROM loy_point_entries WHERE customer_id = $1`, []any{custID}},
		{`DELETE FROM loy_point_ledgers WHERE customer_id = $1`, []any{custID}},
		{`UPDATE loy_earn_rules SET status='DISABLED' WHERE status='ENABLED' AND points_per_yuan=1 AND min_cents=0`, nil},
		{`DELETE FROM coupon_redemptions WHERE customer_id = $1`, []any{custID}},
		{`DELETE FROM coupons WHERE customer_id = $1`, []any{custID}},
		{`DELETE FROM coupon_templates WHERE name LIKE 'MVP验收券-%'`, nil},
		{`DELETE FROM invoices WHERE bill_id IN (SELECT id FROM bills WHERE customer_id = $1 AND bill_no LIKE 'BILL-MVP-%')`, []any{custID}},
	}
	t.Cleanup(func() {
		for _, st := range stmts {
			if _, err := pool.Exec(ctx, st.q, st.args...); err != nil {
				t.Logf("portal mvp cleanup(尽力而为): %v", err)
			}
		}
	})
}

// promotion2EarnRule 缴费送积分种子:1 元=1 分,无门槛,永不过期。
func promotion2EarnRule() loy.EarnRule {
	return loy.EarnRule{PointsPerYuan: 1, MinCents: 0, ExpireDays: 0, Status: "ENABLED"}
}

var _ = config.Config{} // 保持 config 引用与主文件一致
var _ = time.Now
