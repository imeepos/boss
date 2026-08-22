package app_test

// e2e 自清理:测试收尾按 FK 依赖序删除本次 suffix 造的全部数据,防止共享库残留积累
// (历史上 276 条地址/196 客户/275 端点残留即因此而起)。尽力而为,单项失败仅记日志。

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// registerE2ECleanup 在 seedE2E 末尾注册 t.Cleanup;suffix 即各测试码的后缀(纯数字,拼接安全)。
// 注意:e2e 主测试用 t.Cleanup(pool.Close)(先注册后执行),本清理在其前运行,池仍可用。
func registerE2ECleanup(t *testing.T, ctx context.Context, pool *pgxpool.Pool, s *e2eSeed, suffix string) {
	t.Cleanup(func() {
		for _, q := range buildCleanupStmts(s.customerID,
			"E2E-"+suffix, "E2E套餐"+suffix, s.username) {
			if _, err := pool.Exec(ctx, q); err != nil {
				t.Logf("e2e cleanup(尽力而为): %s: %v", firstLine(q), err)
			}
		}
	})
}

// buildCleanupStmts 按 FK 依赖序生成删除语句:e2e/w8 地址子树 → 客户/订单/资源图谱 → 账号。
// 客户集合 = 种子客户 ∪ 挂在两棵测试地址树上的客户(W8 场景另建客户)。
func buildCleanupStmts(custID int64, chanCode, offerName, username string) []string {
	cust := fmt.Sprintf(`(SELECT id FROM customers WHERE id = %d OR address_id IN (%s))`,
		custID, addrSetSQL())
	addr := addrSetSQL()
	return []string{
		// 订单图谱(先孙后子)
		`DELETE FROM dispatch_transfers WHERE ticket_id IN (SELECT id FROM dispatch_tickets WHERE order_id IN (SELECT id FROM orders WHERE customer_id IN ` + cust + `))`,
		`DELETE FROM scan_logs WHERE order_id IN (SELECT id FROM orders WHERE customer_id IN ` + cust + `)`,
		`DELETE FROM order_stages WHERE order_id IN (SELECT id FROM orders WHERE customer_id IN ` + cust + `)`,
		`DELETE FROM activation_callbacks WHERE order_id IN (SELECT id FROM orders WHERE customer_id IN ` + cust + `)`,
		`DELETE FROM dismantles WHERE order_id IN (SELECT id FROM orders WHERE customer_id IN ` + cust + `)`,
		`DELETE FROM dispatch_tickets WHERE order_id IN (SELECT id FROM orders WHERE customer_id IN ` + cust + `)`,
		`DELETE FROM complaints WHERE customer_id IN ` + cust +
			` OR order_id IN (SELECT id FROM orders WHERE customer_id IN ` + cust + `)`,
		// 账务图谱
		`DELETE FROM payments WHERE bill_id IN (SELECT id FROM bills WHERE customer_id IN ` + cust + `)`,
		`DELETE FROM bills WHERE customer_id IN ` + cust,
		`DELETE FROM arrears WHERE customer_id IN ` + cust,
		`DELETE FROM verifications WHERE subject_type='customer' AND subject_id IN ` + cust,
		// 端口/资源图谱(端口先删四码与变更史)
		`DELETE FROM quad_links WHERE port_id IN (SELECT id FROM ports WHERE address_id IN ` + addr + `)`,
		`DELETE FROM port_change_history WHERE port_id IN (SELECT id FROM ports WHERE address_id IN ` + addr + `)`,
		`DELETE FROM ports WHERE address_id IN ` + addr,
		`DELETE FROM resource_assignments WHERE address_id IN ` + addr,
		`DELETE FROM resources WHERE address_id IN ` + addr,
		// 客户与地址树(level 自底向上,父引用 FK)
		`DELETE FROM customer_histories WHERE customer_id IN ` + cust,
		`DELETE FROM customers WHERE id IN ` + cust,
		`DELETE FROM addresses WHERE id IN ` + addr + ` AND level = 5`,
		`DELETE FROM addresses WHERE id IN ` + addr + ` AND level = 4`,
		`DELETE FROM addresses WHERE id IN ` + addr + ` AND level = 3`,
		`DELETE FROM addresses WHERE id IN ` + addr + ` AND level = 2`,
		`DELETE FROM addresses WHERE id IN ` + addr + ` AND level = 1`,
		// 渠道/套餐(码带 suffix)与账号
		`DELETE FROM channels WHERE code = '` + chanCode + `'`,
		`DELETE FROM product_offers WHERE name = '` + offerName + `'`,
		`DELETE FROM account_org_histories WHERE account_id IN (SELECT id FROM accounts WHERE username = '` + username + `')`,
		`DELETE FROM accounts WHERE username = '` + username + `'`,
	}
}

// addrSetSQL 两棵测试地址树的全部节点。按模式而非精确根匹配:W8/W8b 子测试用独立
// orderNo6 后缀建根,精确匹配会漏;^(e2e|w8b?)[0-9]+$ 是 e2e 专用命名,不碰真实数据,
// 且天然覆盖历史运行残留。
func addrSetSQL() string {
	return `(SELECT id FROM addresses WHERE path::text ~ '^(e2e|w8b?|full|cancel|ledger|w5)[0-9]+$')`
}

// firstLine 截 SQL 首行用于日志。
func firstLine(q string) string {
	for i, c := range q {
		if c == '\n' {
			return q[:i]
		}
	}
	return q
}
