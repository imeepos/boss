package report

// 真实 PG 集成测试:订单环节计数器(orders.stage)与环节日志(order_stages)一致性巡检。
// 构造分叉单(orders.stage>N 但日志缺 N 行),验证 PatrolOrphans 能捕获并定位样本 id,
// 事后回滚造数不留残留。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" \
//        go test ./internal/domain/report/ -run TestPatrol_StageConsistency -v -count=1

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/database"
)

const patrolConsistencyCheck = "orders.stage vs order_stages rows"

func TestPatrol_StageConsistency_CatchesGap(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过真实 PG 集成测试")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// 造数:用唯一 order_no 插一单 orders(stage=2, 仅 1 行 order_stages),模拟「计数器到 N 而日志缺 N」。
	// 必须存在 customer_id/offer_id/address_id/channel_id/legal_entity_id(orders 表无 FK,
	// 但 PGStore.Submit 的关联校验在应用层;此处直接 SQL 走真实表,seed id=1 在 102 已存在)。
	tag := fmt.Sprintf("PATROL-CONS-%d", os.Getpid())
	orderNo := "ORD-" + tag
	var orderID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders(order_no, customer_id, offer_id, address_id, stage, status,
		                   channel_id, legal_entity_id)
		VALUES($1, 1, 1, 1, 2, 'PENDING', 1, 1) RETURNING id`, orderNo).Scan(&orderID); err != nil {
		t.Fatalf("insert test order: %v", err)
	}
	// 仅写 1 行 order_stages(stage=1),故意让 stage=2 > count=1。
	if _, err := pool.Exec(ctx,
		`INSERT INTO order_stages(order_id, stage, result) VALUES($1, 1, 'DONE')`, orderID); err != nil {
		t.Fatalf("insert test stage: %v", err)
	}

	// 清理保证不留残留:无论断言成败都执行。
	// 注意:defer pool.Close() 在 t.Cleanup 之前运行,因此清理走独立连接。
	t.Cleanup(func() {
		cleanPool, err := database.Open(context.Background(), dsn)
		if err != nil {
			t.Logf("cleanup open pool: %v", err)
			return
		}
		defer cleanPool.Close()
		cleanCtx, c2 := context.WithTimeout(context.Background(), 10*time.Second)
		defer c2()
		if _, err := cleanPool.Exec(cleanCtx, `DELETE FROM order_stages WHERE order_id = $1`, orderID); err != nil {
			t.Logf("cleanup order_stages: %v", err)
		}
		if _, err := cleanPool.Exec(cleanCtx, `DELETE FROM orders WHERE id = $1`, orderID); err != nil {
			t.Logf("cleanup orders: %v", err)
		}
	})

	// 巡检:必须命中一致性检查且样本包含本次造数的 orderID。
	s := NewPGStore(pool)
	findings, err := s.PatrolOrphans(ctx)
	if err != nil {
		t.Fatalf("PatrolOrphans: %v", err)
	}
	var finding *OrphanFinding
	for i := range findings {
		if findings[i].Check == patrolConsistencyCheck {
			finding = &findings[i]
			break
		}
	}
	if finding == nil {
		t.Fatalf("巡检项 %q 未出现在结果中;现有检查=%v", patrolConsistencyCheck, checkNames(findings))
	}
	if finding.Orphans < 1 {
		t.Fatalf("未捕获分叉单:orphans=%d", finding.Orphans)
	}
	found := false
	for _, id := range finding.SampleIDs {
		if id == orderID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("巡检未把测试造数 orderID=%d 列入样本 ids=%v", orderID, finding.SampleIDs)
	}

	// 恢复路径验证:把 stage 回退到与日志一致(1),再跑巡检应不再把该订单计入。
	if _, err := pool.Exec(ctx, `UPDATE orders SET stage = 1 WHERE id = $1`, orderID); err != nil {
		t.Fatalf("restore stage: %v", err)
	}
	findings2, err := s.PatrolOrphans(ctx)
	if err != nil {
		t.Fatalf("PatrolOrphans(recovered): %v", err)
	}
	for _, f := range findings2 {
		if f.Check != patrolConsistencyCheck {
			continue
		}
		for _, id := range f.SampleIDs {
			if id == orderID {
				t.Fatalf("恢复后巡检仍把 orderID=%d 计入分叉,orphans=%d ids=%v",
					orderID, f.Orphans, f.SampleIDs)
			}
		}
	}
}

func checkNames(fs []OrphanFinding) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, strings.SplitN(f.Check, " ", 2)[0])
	}
	return out
}
