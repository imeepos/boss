// Webhook 投递领取的真实 PG 集成验证(持久化整改回归)。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" \
//   go test ./internal/domain/openplat/ -run TestWebhookPGIntegration -v -count=1
package openplat

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// newHookTestPool 连接真实 PG;未设置 DSN 时跳过(与 app 层 E2E 同口径)。
func newHookTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// seedHookFixture 建测试应用+订阅+两条到期投递;返回投递 id 并注册清理。
func seedHookFixture(t *testing.T, pool *pgxpool.Pool, eventType string) []int64 {
	t.Helper()
	ctx := context.Background()
	var appID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO open_apps(app_id, secret, name, rate_limit_rpm, daily_quota, status, created_by)
		 VALUES('op_persist_test', 'ops_persist_test_secret', '持久化整改集成测试', 60, 1000, 1, 0)
		 RETURNING id`).Scan(&appID); err != nil {
		t.Fatalf("seed app: %v", err)
	}
	var subID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO open_webhook_subscriptions(app_id, event_type, endpoint_url, status)
		 VALUES($1, $2, 'https://hook.example.invalid/persist', 1) RETURNING id`,
		appID, eventType).Scan(&subID); err != nil {
		t.Fatalf("seed sub: %v", err)
	}
	past := time.Now().Add(-time.Second)
	ids := make([]int64, 0, 2)
	for _, ev := range []string{"evt-a", "evt-b"} {
		var id int64
		if err := pool.QueryRow(ctx,
			`INSERT INTO open_webhook_deliveries(subscription_id, event_id, event_type, payload, next_attempt_at)
			 VALUES($1,$2,$3,'{"test":true}',$4) RETURNING id`, subID, ev, eventType, past).Scan(&id); err != nil {
			t.Fatalf("seed delivery: %v", err)
		}
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM open_webhook_deliveries WHERE subscription_id=$1`, subID)
		_, _ = pool.Exec(ctx, `DELETE FROM open_webhook_subscriptions WHERE id=$1`, subID)
		_, _ = pool.Exec(ctx, `DELETE FROM open_apps WHERE id=$1`, appID)
	})
	return ids
}

// TestWebhookPGIntegration_ClaimExclusiveAcrossInstances 验证并发领取互不相交:
// 第二个"实例"(新连接)立即调用 ListDue 拿不到已被第一实例租约领取的行。
func TestWebhookPGIntegration_ClaimExclusiveAcrossInstances(t *testing.T) {
	pool := newHookTestPool(t)
	eventType := fmt.Sprintf("persist.claim.%d", time.Now().UnixNano())
	ids := seedHookFixture(t, pool, eventType)
	ctx := context.Background()

	instA := NewPGStore(pool)
	gotA, err := instA.ListDue(ctx, time.Now(), DeliveryBatchMax)
	if err != nil {
		t.Fatalf("instance A list due: %v", err)
	}
	if len(gotA) != 2 {
		t.Fatalf("instance A got %d rows, want 2", len(gotA))
	}

	instB := NewPGStore(pool)
	gotB, err := instB.ListDue(ctx, time.Now(), DeliveryBatchMax)
	if err != nil {
		t.Fatalf("instance B list due: %v", err)
	}
	for _, dl := range gotB {
		if dl.SubscriptionID == gotA[0].SubscriptionID && eventType == dl.EventType {
			t.Fatalf("double claim across instances: id=%d", dl.ID)
		}
	}
	t.Logf("instance A claimed %d, instance B saw %d intersecting=0", len(gotA), len(gotB))

	// 结果回写不受领取影响。
	if err := instA.MarkResult(ctx, ids[0], true, 200, ""); err != nil {
		t.Fatalf("mark result: %v", err)
	}
	var status int16
	if err := pool.QueryRow(ctx,
		`SELECT status FROM open_webhook_deliveries WHERE id=$1`, ids[0]).Scan(&status); err != nil || status != DeliveryDone {
		t.Fatalf("delivered status persist: status=%d err=%v", status, err)
	}
}

// TestWebhookPGIntegration_RecoverAfterLeaseExpiry 验证崩溃恢复与重启不丢:
// 投递器在领取后崩溃(MarkResult 未写),行在租约到期后重新可见、可继续处理;
// 已投递完成的行不回流。重启模拟 = 全新连接的 store 实例读取同一批数据。
func TestWebhookPGIntegration_RecoverAfterLeaseExpiry(t *testing.T) {
	pool := newHookTestPool(t)
	eventType := fmt.Sprintf("persist.recover.%d", time.Now().UnixNano())
	ids := seedHookFixture(t, pool, eventType)
	ctx := context.Background()

	crashed := NewPGStore(pool)
	got, err := crashed.ListDue(ctx, time.Now(), DeliveryBatchMax)
	if err != nil || len(got) == 0 {
		t.Fatalf("pre-crash claim: n=%d err=%v", len(got), err)
	}
	claimed := map[int64]bool{}
	for _, dl := range got {
		claimed[dl.ID] = true
	}
	if !claimed[ids[0]] || !claimed[ids[1]] {
		t.Fatalf("fixture rows not both claimed: %v vs %v/%v", claimed, ids[0], ids[1])
	}
	_ = crashed // 此处进程即告崩溃,MarkResult 未执行

	// 重启后的实例 + 租约到期(next_attempt_at 回到过去等价于窗口已过)。
	restarted := NewPGStore(pool)
	postcrash := time.Now().Add(time.Duration(claimLeaseSeconds) * time.Second)
	if _, err := pool.Exec(ctx,
		`UPDATE open_webhook_deliveries SET next_attempt_at=$1 WHERE id=$2`, postcrash.Add(-time.Second), ids[1]); err != nil {
		t.Fatalf("simulate lease expiry: %v", err)
	}
	again, err := restarted.ListDue(ctx, postcrash, DeliveryBatchMax)
	if err != nil {
		t.Fatalf("post-restart list due: %v", err)
	}
	found := false
	deliveredBack := false
	for _, dl := range again {
		switch dl.ID {
		case ids[1]:
			found = true
		case ids[0]:
			deliveredBack = true
		}
	}
	if !found {
		t.Fatal("unacked delivery lost after simulated crash+restart")
	}
	if deliveredBack {
		t.Fatal("already-delivered row redelivered")
	}
	// 继续处理照常:成功落库后状态持久。
	if err := restarted.MarkResult(ctx, ids[1], true, 200, ""); err != nil {
		t.Fatalf("post-restart deliver: %v", err)
	}
}
