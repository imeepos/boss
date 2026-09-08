package odn

import (
	"context"
	"os"
	"testing"

	"github.com/ymm-001/boss/internal/pkg/database"
)

// TestNextFacilityCode_Integration 取号口径真实 PG 回归(BOSS_PG_TEST_DSN 未设置时跳过)。
// W7 实锤缺陷回归:序号=前缀+网格之后的 3 位,不得把网格位并入;本测试走真实
// pgxpool + 真实 SQL,守住 pgxmock 覆盖不到的参数定型/语法层($2::int 显式定型的由来)。
// 用网格 96/97 隔离,不碰 102 存量 91/98 数据;t.Cleanup 自清造数。
func TestNextFacilityCode_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	s := NewPGStore(pool)

	cleanup := func() {
		pool.Exec(ctx, `DELETE FROM odn_facility
			WHERE prv_code='PHL001' AND city_prefix='MNL' AND grid_code IN (96,97)`)
		pool.Exec(ctx, `DELETE FROM odn_grid
			WHERE prv_code='PHL001' AND city_prefix='MNL' AND grid_code IN (96,97)`)
	}
	cleanup()
	t.Cleanup(cleanup)
	defer func() {
		cleanup() // 成功路径显式清理;t.Cleanup 兜底跑在池关闭后会静默失效,勿依赖
		pool.Close()
	}()

	if err := s.CreateGrid(ctx, Grid{PrvCode: "PHL001", CityPrefix: "MNL",
		GridCode: 96, Status: GridActive}); err != nil {
		t.Fatalf("CreateGrid 96: %v", err)
	}
	if err := s.CreateGrid(ctx, Grid{PrvCode: "PHL001", CityPrefix: "MNL",
		GridCode: 97, Status: GridActive}); err != nil {
		t.Fatalf("CreateGrid 97: %v", err)
	}
	mk := func(code string, grid int16) {
		t.Helper()
		if err := s.CreateFacility(ctx, Facility{Code: code, Kind: KindPole,
			PrvCode: "PHL001", CityPrefix: "MNL", GridCode: grid,
			Name: "取号回归" + code}); err != nil {
			t.Fatalf("CreateFacility %s: %v", code, err)
		}
	}
	mk("P96011", 96)
	mk("P96012", 96)
	mk("P97001", 97)

	// P91 同型场景(网格段不被并入序号):96 网格现存 012 → 顺延 013,而非误报用尽。
	got, err := s.NextFacilityCode(ctx, KindPole, 96)
	if err != nil || got != "P96013" {
		t.Fatalf("grid96 got=%q err=%v, want P96013/nil", got, err)
	}
	// 无既有编码的网格从 001 起。
	got, err = s.NextFacilityCode(ctx, KindPole, 97)
	if err != nil || got != "P97002" {
		t.Fatalf("grid97 got=%q err=%v, want P97002/nil", got, err)
	}
	// 跨网格互不影响:再取 96 仍顺延,不被 97 的取号污染。
	got, err = s.NextFacilityCode(ctx, KindPole, 96)
	if err != nil || got != "P96013" {
		t.Fatalf("grid96 again got=%q err=%v, want P96013/nil", got, err)
	}
	// 顺序模式:真实 PG 下 5 位序号紧随前缀;102 库 TW 行被并行造数影响时只验口径不锁号。
	got, err = s.NextFacilityCode(ctx, KindTower, 0)
	if err != nil {
		t.Fatalf("TW: %v", err)
	}
	if k, grid, seq, verr := ValidateFacilityCode(got); verr != nil || k != KindTower || grid != 0 || seq < 1 {
		t.Fatalf("TW got=%q 解析 k=%s grid=%d seq=%d err=%v", got, k, grid, seq, verr)
	}
}
