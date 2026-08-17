package user

import (
	"context"
	"os"
	"testing"

	"github.com/ymm-001/boss/internal/pkg/database"
)

// TestPGStore_Integration 端到端验证「Open → Migrate → 查询」,需真实 PostgreSQL。
// 运行: BOSS_PG_TEST_DSN="host=localhost port=15432 user=boss password=boss dbname=boss sslmode=disable" go test ./internal/domain/user/ -run TestPGStore_Integration -v
// 未设置 DSN 时跳过(纯单测不依赖外部 PG)。
func TestPGStore_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("database.Migrate: %v", err)
	}

	s := NewPGStore(pool)
	got, err := s.ListLegalEntities(ctx)
	if err != nil {
		t.Fatalf("ListLegalEntities: %v", err)
	}
	// migrations/000002 种子 LEG-A/LEG-B/LEG-C。
	if len(got) < 3 {
		t.Fatalf("len=%d, want >=3 (seed LEG-A/B/C)", len(got))
	}
	t.Logf("legal_entities=%+v", got)
}
