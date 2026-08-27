package openplat

// InsertDeliveries 真实 PG 回归:首次出现匹配订阅时,载荷 []byte 走 $3::jsonb 后
// 必须成功落库(bytea→jsonb 隐式转型不存在的 22P02 是这次修复的对象)。
// 运行:
//   BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" \
//     go test ./internal/domain/openplat/ -run TestInsertDeliveries_RealPG -v -count=1

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestInsertDeliveries_RealPG(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过真实 PG 集成测试")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	ctx := context.Background()
	eventType := fmt.Sprintf("persist.insert.%d", time.Now().UnixNano())

	var appID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO open_apps(app_id, secret, name, rate_limit_rpm, daily_quota, status, created_by)
		 VALUES('op_persist_insert','ops_persist_insert','InsertDeliveries集成测试',60,1000,1,0)
		 RETURNING id`).Scan(&appID); err != nil {
		t.Fatalf("seed app: %v", err)
	}
	var subID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO open_webhook_subscriptions(app_id, event_type, endpoint_url, status)
		 VALUES($1,$2,'https://hook.example.invalid/insert',1) RETURNING id`,
		appID, eventType).Scan(&subID); err != nil {
		t.Fatalf("seed sub: %v", err)
	}
	t.Cleanup(func() {
		cp, err := pgxpool.New(context.Background(), dsn)
		if err != nil {
			t.Logf("cleanup pool: %v", err)
			return
		}
		defer cp.Close()
		c, c2 := context.WithTimeout(context.Background(), 10*time.Second)
		defer c2()
		_, _ = cp.Exec(c, `DELETE FROM open_webhook_deliveries WHERE subscription_id=$1`, subID)
		_, _ = cp.Exec(c, `DELETE FROM open_webhook_subscriptions WHERE id=$1`, subID)
		_, _ = cp.Exec(c, `DELETE FROM open_apps WHERE id=$1`, appID)
	})

	s := NewPGStore(pool)
	eventID := fmt.Sprintf("evt-%d", time.Now().UnixNano())
	body := []byte(`{"type":"persist.insert.test","n":42,"nested":{"k":"v"}}`)
	n, err := s.InsertDeliveries(ctx, eventType, eventID, body)
	if err != nil {
		t.Fatalf("InsertDeliveries 必返 22P02 历史已修: %v", err)
	}
	if n < 1 {
		t.Fatalf("匹配订阅存在时 InsertDeliveries 应至少建 1 行,n=%d", n)
	}

	// 载荷以 JSONB 落库且语义保持(PG 规范化键序,故按 JSON 结构比对)。
	var gotText string
	if err := pool.QueryRow(ctx,
		`SELECT payload::text FROM open_webhook_deliveries WHERE event_id=$1`, eventID).Scan(&gotText); err != nil {
		t.Fatalf("read back payload: %v", err)
	}
	gotJSON := jsonValid(t, gotText)
	wantJSON := jsonValid(t, string(body))
	if gotJSON != wantJSON {
		t.Fatalf("payload 语义不一致:\n got=%s\nwant=%s", gotJSON, wantJSON)
	}

	// 幂等:同 (subscription_id, event_id) 重放返回 n=0,无新行。
	n2, err := s.InsertDeliveries(ctx, eventType, eventID, body)
	if err != nil {
		t.Fatalf("InsertDeliveries 幂等重放: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("幂等重放应返回 n=0,got n=%d", n2)
	}
}

// jsonValid 解析并以规范化键序重编码,用于 JSONB 落库语义对比。
func jsonValid(t *testing.T, s string) string {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("invalid json %q: %v", s, err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	return string(b)
}
