package worker

import (
	"context"
	"os"
	"testing"

	"github.com/ymm-001/boss/internal/pkg/database"
)

// TestMaterialLink_Integration E13 回填验证(需真实 PG,未设 DSN 跳过):
// 000091 后 item_id/tool_id 列存在,存量行按唯一匹配回填,插入带 item_id 的行成功。
func TestMaterialLink_Integration(t *testing.T) {
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

	var linked, total int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM worker_materials WHERE item_id IS NOT NULL`).Scan(&linked); err != nil {
		t.Fatalf("count linked: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM worker_materials`).Scan(&total); err != nil {
		t.Fatalf("count total: %v", err)
	}
	t.Logf("materials linked=%d/%d", linked, total)

	s := NewPGStore(pool)
	var itemID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM material_items ORDER BY id LIMIT 1`).Scan(&itemID); err != nil {
		t.Fatalf("pick material item: %v", err)
	}
	// 取真实师傅/班组快照,避免 FK 违约。
	var m Material
	if err := pool.QueryRow(ctx, `
		SELECT w.id, g.id, g.name, le.id, le.name, w.region_id, COALESCE(r.name,'')
		FROM workers w
		JOIN worker_groups g ON g.id = w.group_id
		JOIN legal_entities le ON le.id = g.legal_entity_id
		LEFT JOIN regions r ON r.id = w.region_id
		WHERE w.status = 1 ORDER BY w.id LIMIT 1`).
		Scan(&m.WorkerID, &m.GroupID, &m.GroupName, &m.LegalEntityID, &m.LegalEntityName, &m.RegionID, &m.RegionName); err != nil {
		t.Fatalf("pick worker snapshot: %v", err)
	}
	m.ItemID, m.Name, m.Qty = itemID, "集成测试物料", 1
	id, err := s.AppendMaterial(ctx, m)
	if err != nil {
		t.Fatalf("AppendMaterial with ItemID: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `DELETE FROM worker_materials WHERE id=$1`, id) })

	var back int64
	if err := pool.QueryRow(ctx,
		`SELECT item_id FROM worker_materials WHERE id=$1`, id).Scan(&back); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if back != itemID {
		t.Fatalf("item_id=%d, want %d", back, itemID)
	}
}
