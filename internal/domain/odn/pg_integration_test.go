package odn

import (
	"context"
	"os"
	"testing"

	"github.com/ymm-001/boss/internal/pkg/database"
)

// TestODNPassive_Integration 网格分区 + 基础设施端到端(需真实 PostgreSQL,
// BOSS_PG_TEST_DSN 未设置时跳过)。以 000075 的 PHL001/MNL 为样例城市。
func TestODNPassive_Integration(t *testing.T) {
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
	cleanup := func() {
		pool.Exec(ctx, `DELETE FROM odn_facility WHERE code IN ('P98001','MH98001','TW98001')
			AND prv_code='PHL001' AND city_prefix='MNL'`)
		pool.Exec(ctx, `DELETE FROM odn_grid WHERE prv_code='PHL001' AND city_prefix='MNL' AND grid_code=98`)
	}
	cleanup()
	t.Cleanup(cleanup)

	// 网格备案(红线 2)。
	if err := s.CreateGrid(ctx, Grid{PrvCode: "PHL001", CityPrefix: "MNL",
		GridCode: 98, Name: "集成测试网格", Coverage: "t", Status: GridActive}); err != nil {
		t.Fatalf("CreateGrid: %v", err)
	}
	// 网格重复 → ErrDuplicate。
	if err := s.CreateGrid(ctx, Grid{PrvCode: "PHL001", CityPrefix: "MNL",
		GridCode: 98, Status: GridActive}); err != ErrDuplicate {
		t.Fatalf("重复网格期望 ErrDuplicate,实际 %v", err)
	}

	// 电杆:网格段必须与所属网格一致;未备案网格拒绝。
	if err := s.CreateFacility(ctx, Facility{Code: "P99001", Kind: KindPole,
		PrvCode: "PHL001", CityPrefix: "MNL", GridCode: 98}); err == nil {
		t.Fatal("编码网格段 99 与所属网格 98 不符,应拒绝")
	}
	if err := s.CreateFacility(ctx, Facility{Code: "P99001", Kind: KindPole,
		PrvCode: "PHL001", CityPrefix: "MNL", GridCode: 99}); err != ErrGridMissing {
		t.Fatalf("未备案网格期望 ErrGridMissing,实际 %v", err)
	}
	if err := s.CreateFacility(ctx, Facility{Code: "P98001", Kind: KindPole,
		PrvCode: "PHL001", CityPrefix: "MNL", GridCode: 98}); err != nil {
		t.Fatalf("CreateFacility P98001: %v", err)
	}
	// 人井 + 铁塔(市域顺序,无网格)。
	if err := s.CreateFacility(ctx, Facility{Code: "MH98001", Kind: KindManhole,
		PrvCode: "PHL001", CityPrefix: "MNL", GridCode: 98}); err != nil {
		t.Fatalf("CreateFacility MH98001: %v", err)
	}
	if err := s.CreateFacility(ctx, Facility{Code: "TW98001", Kind: KindTower,
		PrvCode: "PHL001", CityPrefix: "MNL"}); err != nil {
		t.Fatalf("CreateFacility TW98001: %v", err)
	}

	// 列表与占用统计。
	grids, err := s.ListGrids(ctx, "PHL001", "MNL")
	if err != nil {
		t.Fatalf("ListGrids: %v", err)
	}
	found := false
	for _, g := range grids {
		if g.GridCode == 98 {
			found = true
			if g.Facilities != 2 {
				t.Fatalf("网格 98 占用期望 2(P+MH),实际 %d", g.Facilities)
			}
		}
	}
	if !found {
		t.Fatal("ListGrids 未返回网格 98")
	}
	facs, err := s.ListFacilities(ctx, KindPole, GridRef{PrvCode: "PHL001", CityPrefix: "MNL", GridCode: 98})
	if err != nil || len(facs) != 1 || facs[0].Code != "P98001" {
		t.Fatalf("ListFacilities: %v %d", err, len(facs))
	}

	// 报废永久锁定:RETIRED 后同码复用被 ErrDuplicate 拦截。
	if err := s.RetireFacility(ctx, "TW98001"); err != nil {
		t.Fatalf("RetireFacility: %v", err)
	}
	if err := s.CreateFacility(ctx, Facility{Code: "TW98001", Kind: KindTower,
		PrvCode: "PHL001", CityPrefix: "MNL"}); err != ErrDuplicate {
		t.Fatalf("报废码复用期望 ErrDuplicate,实际 %v", err)
	}
	// 在用设施未清空前网格不可退役。
	if err := s.RetireGrid(ctx, "PHL001", "MNL", 98); err == nil {
		t.Fatal("在用设施存在,网格退役应失败")
	}
}

// TestODNCable_Integration 光缆段落定向 + 纤芯顺序(需真实 PostgreSQL)。
func TestODNCable_Integration(t *testing.T) {
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
	pool.Exec(ctx, "DELETE FROM odn_cable_segment WHERE a_code='ODF001' AND b_code='OCC001'")

	// 定向:颠倒输入也归一为 ODF 为 A 端;同对端点幂等。
	seg, err := s.CreateSegment(ctx, "OCC001", "ODF001", "t")
	if err != nil {
		t.Fatalf("CreateSegment: %v", err)
	}
	if seg.ACode != "ODF001" || seg.BCode != "OCC001" {
		t.Fatalf("定向错误 A=%s B=%s", seg.ACode, seg.BCode)
	}
	if _, err := s.CreateSegment(ctx, "ODF001", "OCC001", "t"); err != nil {
		t.Fatalf("同对端点应幂等: %v", err)
	}
	// 同优先级拒绝。
	if _, err := s.CreateSegment(ctx, "P01001", "P02005", ""); err != ErrSamePriority {
		t.Fatalf("同优先级期望 ErrSamePriority,实际 %v", err)
	}

	// 纤芯 G01~G99,重复 G 号拒绝。
	if err := s.AddFiber(ctx, seg.ID, Fiber{GNo: 1, Kind: "GYTA"}); err != nil {
		t.Fatalf("AddFiber: %v", err)
	}
	if err := s.AddFiber(ctx, seg.ID, Fiber{GNo: 1}); err != ErrDuplicate {
		t.Fatalf("重复 G 号期望 ErrDuplicate,实际 %v", err)
	}
	fibers, err := s.ListFibers(ctx, seg.ID)
	if err != nil || len(fibers) != 1 || fibers[0].GNo != 1 {
		t.Fatalf("ListFibers: %v %d", err, len(fibers))
	}

	// 清理。
	pool.Exec(ctx, "DELETE FROM odn_cable_segment WHERE a_code='ODF001' AND b_code='OCC001'")
}
