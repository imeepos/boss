package odn

import (
	"context"
	"os"
	"testing"

	"github.com/ymm-001/boss/internal/pkg/database"
)

// f64p float64 指针(设备坐标可空字段测试用)。
func f64p(v float64) *float64 { return &v }

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

// TestODNSiteDevice_Integration 局点 + 核心链路设备端到端(需真实 PostgreSQL)。
func TestODNSiteDevice_Integration(t *testing.T) {
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
		pool.Exec(ctx, `DELETE FROM odn_device WHERE prv_code='PHL001' AND city_prefix='MNL'
			AND code IN ('SNW990','OLT990','OLT991','OCC990','ODB990','ODB990-2','SDB990')`)
		pool.Exec(ctx, `DELETE FROM odn_site WHERE prv_code='PHL001' AND city_prefix='MNL' AND site_no=998`)
	}
	cleanup()
	t.Cleanup(cleanup)

	// 局点:MNL998(避开规范预留 MNL150~999 之外的测试位? 998 在预留段,测试专用)。
	if err := s.CreateSite(ctx, Site{PrvCode: "PHL001", CityPrefix: "MNL",
		SiteNo: 998, Name: "集成测试局点"}); err != nil {
		t.Fatalf("CreateSite: %v", err)
	}
	sites, err := s.ListSites(ctx, "PHL001", "MNL")
	if err != nil || len(sites) == 0 {
		t.Fatalf("ListSites: %v %d", err, len(sites))
	}

	// 顶层设备 SNW(全网唯一)与 OCC(市域唯一)。
	if err := s.CreateDevice(ctx, Device{Code: "SNW990", Kind: DevSNW,
		PrvCode: "PHL001", CityPrefix: "MNL", SiteNo: 998}); err != nil {
		t.Fatalf("CreateDevice SNW990: %v", err)
	}
	// 设备坐标(000143):带 lat/lng 创建 → 列表回读。
	if err := s.CreateDevice(ctx, Device{Code: "OLT990", Kind: DevOLT,
		PrvCode: "PHL001", CityPrefix: "MNL", Lat: f64p(14.55), Lng: f64p(120.98)}); err != nil {
		t.Fatalf("CreateDevice OLT990(带坐标): %v", err)
	}
	olts, err := s.ListDevices(ctx, DevOLT, "PHL001", "MNL")
	if err != nil || len(olts) != 1 {
		t.Fatalf("ListDevices OLT: %v %d", err, len(olts))
	}
	if olts[0].Lat == nil || olts[0].Lng == nil || *olts[0].Lat != 14.55 || *olts[0].Lng != 120.98 {
		t.Fatalf("设备坐标回读失败: lat=%v lng=%v", olts[0].Lat, olts[0].Lng)
	}
	// 无坐标设备(E16:site_no 无 FK,域层守护——挂未备案局点必须拒绝)。
	if err := s.CreateDevice(ctx, Device{Code: "OLT991", Kind: DevOLT,
		PrvCode: "PHL001", CityPrefix: "MNL", SiteNo: 997}); err == nil {
		t.Fatal("OLT 挂未备案局点 997 应拒绝(ErrSiteMissing)")
	}
	// SNW990 无坐标创建 → Lat/Lng 为 nil。
	snws, err := s.ListDevices(ctx, DevSNW, "PHL001", "MNL")
	if err != nil || len(snws) != 1 {
		t.Fatalf("ListDevices SNW: %v %d", err, len(snws))
	}
	if snws[0].Lat != nil || snws[0].Lng != nil {
		t.Fatalf("无坐标设备应 Lat/Lng=nil,实际 %v %v", snws[0].Lat, snws[0].Lng)
	}
	if err := s.CreateDevice(ctx, Device{Code: "OCC990", Kind: DevOCC,
		PrvCode: "PHL001", CityPrefix: "MNL"}); err != nil {
		t.Fatalf("CreateDevice OCC990: %v", err)
	}
	// 归属链:ODB 必须挂在 OCC 下;错挂 OLT 下拒绝。
	devs, err := s.ListDevices(ctx, DevOCC, "PHL001", "MNL")
	if err != nil || len(devs) != 1 {
		t.Fatalf("ListDevices OCC: %v %d", err, len(devs))
	}
	occID := devs[0].ID
	if err := s.CreateDevice(ctx, Device{Code: "ODB990", Kind: DevODB,
		PrvCode: "PHL001", CityPrefix: "MNL", ParentID: 999999}); err == nil {
		t.Fatal("上级不存在应拒绝")
	}
	if err := s.CreateDevice(ctx, Device{Code: "ODB990", Kind: DevODB,
		PrvCode: "PHL001", CityPrefix: "MNL", ParentID: occID}); err != nil {
		t.Fatalf("CreateDevice ODB990: %v", err)
	}
	// 同址扩容:ODB990-2 挂同一 OCC。
	if err := s.CreateDevice(ctx, Device{Code: "ODB990-2", Kind: DevODB,
		PrvCode: "PHL001", CityPrefix: "MNL", ParentID: occID}); err != nil {
		t.Fatalf("CreateDevice ODB990-2: %v", err)
	}
	// SDB 必须挂 ODB 下,挂 OCC 拒绝。
	if err := s.CreateDevice(ctx, Device{Code: "SDB990", Kind: DevSDB,
		PrvCode: "PHL001", CityPrefix: "MNL", ParentID: occID}); err == nil {
		t.Fatal("SDB 挂 OCC 应拒绝(须挂 ODB)")
	}
	odbs, _ := s.ListDevices(ctx, DevODB, "PHL001", "MNL")
	if len(odbs) != 2 {
		t.Fatalf("ODB 期望 2 台(含扩容),实际 %d", len(odbs))
	}
	for _, d := range odbs {
		if d.ParentID != occID {
			t.Fatalf("ODB %s 归属错误 parent=%d", d.Code, d.ParentID)
		}
	}

	// 报废锁定 + 局点退役联动。
	if err := s.RetireDevice(ctx, occID); err != nil {
		t.Fatalf("RetireDevice: %v", err)
	}
	if err := s.RetireSite(ctx, "PHL001", "MNL", 998); err == nil {
		t.Fatal("SNW990 仍挂局点 998,退役应失败")
	}
}
