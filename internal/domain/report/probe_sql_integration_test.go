package report

// 探针真库回归:ReconCounts/CompensationTasks 的每条 SQL 必须能在真实 schema 上执行。
// 起因:loy_ledgers/gis_points/coupons.offer_id/status='FAILED' vs SMALLINT 等幽灵引用
// 只有 mock 测试,从未真库执行,而"单项失败即整轮失败"的 DailyRecon 因此永远落不了
// recon-daily 快照(2026-09 schema 审计 CRITICAL 组)。只读探针,零造数。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" \
//   go test ./internal/domain/report/ -run TestProbeSQL -v -count=1

import (
	"context"
	"os"
	"testing"

	"github.com/ymm-001/boss/internal/pkg/database"
)

func TestProbeSQL_RealPG(t *testing.T) {
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
	s := NewPGStore(pool)

	checks, err := s.ReconCounts(ctx)
	if err != nil {
		t.Fatalf("ReconCounts: %v", err)
	}
	if len(checks) != len(reconQueries) {
		t.Fatalf("checks=%d, want %d(每条探针都必须真库可执行)", len(checks), len(reconQueries))
	}
	for _, c := range checks {
		if c.Count < 0 {
			t.Fatalf("negative count: %+v", c)
		}
	}
	if _, err := s.CompensationTasks(ctx); err != nil {
		t.Fatalf("CompensationTasks: %v", err)
	}
}
