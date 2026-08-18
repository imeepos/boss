package report

// 阶段9 真实 PG 集成测试:报告生成落库(迁移 000034)、同窗口幂等覆盖、Latest 取回。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" go test ./internal/domain/report/ -run TestReport -v -count=1

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/analytics"
	"github.com/ymm-001/boss/internal/pkg/database"
)

func TestReport_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	r := &ReportService{Ana: analytics.NewPGStore(pool, 50, 800), St: NewPGStore(pool)}
	at := time.Now().Truncate(time.Hour)

	snap, err := r.Generate(ctx, "daily", at)
	if err != nil {
		t.Fatal(err)
	}
	if snap.ID == 0 || len(snap.Payload) == 0 {
		t.Fatalf("snap=%+v", snap)
	}

	t.Run("同窗口幂等覆盖不新增行", func(t *testing.T) {
		again, err := r.Generate(ctx, "daily", at)
		if err != nil {
			t.Fatal(err)
		}
		if again.ID != snap.ID {
			t.Fatalf("再次生成 id=%d, want %d", again.ID, snap.ID)
		}
	})

	t.Run("Latest 取回且正文完整", func(t *testing.T) {
		got, err := r.Latest(ctx, "daily")
		if err != nil {
			t.Fatal(err)
		}
		var p Payload
		if err := json.Unmarshal(got.Payload, &p); err != nil {
			t.Fatal(err)
		}
		if len(p.Indicators) != 5 || len(p.Conclusions) == 0 {
			var keys []string
			for _, i := range p.Indicators {
				keys = append(keys, i.Key)
			}
			t.Fatalf("payload 指标=%v 结论=%d", keys, len(p.Conclusions))
		}
	})
}
