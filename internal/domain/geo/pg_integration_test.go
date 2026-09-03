package geo

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ymm-001/boss/internal/pkg/database"
)

// TestPGStore_Integration 端到端验证 geo 域 CRUD(需真实 PostgreSQL)。
//
//	运行: BOSS_PG_TEST_DSN="host=127.0.0.1 port=25432 user=boss password=boss dbname=boss sslmode=disable" \
//		go test ./internal/domain/geo/ -run TestPGStore_Integration -v
//
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
	// 幂等清理上次遗留的测试数据(ZZ 测试国及其派生)。
	if _, err := pool.Exec(ctx, `DELETE FROM geo_subdivision_i18n WHERE subdivision_code LIKE 'ZZ-%';
		DELETE FROM geo_subdivision WHERE code LIKE 'ZZ-%';
		DELETE FROM geo_country_i18n WHERE country_code='ZZ';
		DELETE FROM country_time_zone WHERE country_code='ZZ';
		DELETE FROM country_currency WHERE country_code='ZZ';
		DELETE FROM country_calling_code WHERE country_code='ZZ';
		DELETE FROM geo_country WHERE alpha2='ZZ'`); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	testCountryCRUD(ctx, t, s)
	testSubdivisionCRUD(ctx, t, s)
	testImport(ctx, t, s)
	testSubdivisionQuery(ctx, t, s)
	testDefaultCountry(ctx, t, s, pool)
}

// testCountryCRUD 国家建/查/改/译名/属性/停用闭环。
func testCountryCRUD(ctx context.Context, t *testing.T, s *PGStore) {
	t.Helper()
	const alpha2 = "ZZ"
	if err := s.CreateCountry(ctx, Country{
		Alpha2: alpha2, Alpha3: "ZZZ", NumericCode: "999", ShortName: "Testland",
		Status: "INDEPENDENT", ContinentCode: "AS",
	}); err != nil {
		t.Fatalf("CreateCountry: %v", err)
	}
	if err := s.CreateCountry(ctx, Country{Alpha2: alpha2, Alpha3: "ZZZ",
		NumericCode: "998", ShortName: "Dup", Status: "INDEPENDENT", ContinentCode: "AS"}); err != ErrDuplicate {
		t.Fatalf("duplicate want ErrDuplicate, got %v", err)
	}

	d, err := s.GetCountry(ctx, alpha2)
	if err != nil {
		t.Fatalf("GetCountry: %v", err)
	}
	if d.ShortName != "Testland" || len(d.Attrs.TimeZones) != 0 {
		t.Fatalf("detail mismatch: %+v", d)
	}

	if err := s.UpdateCountry(ctx, alpha2, Country{Alpha3: "ZZZ", NumericCode: "999",
		ShortName: "Testland2", Status: "INDEPENDENT", ContinentCode: "EU"}); err != nil {
		t.Fatalf("UpdateCountry: %v", err)
	}
	if err := s.AddCountryName(ctx, alpha2, CountryName{Locale: "zh-Hans", Name: "测试国", NameType: "STANDARD"}); err != nil {
		t.Fatalf("AddCountryName: %v", err)
	}
	if err := s.ReplaceCountryAttrs(ctx, alpha2, CountryAttrs{
		TimeZones: []string{"Asia/Shanghai"}, CallingCodes: []string{"999"},
		Currencies: []Currency{{Currency: "TSC", IsPrimary: true, MinorUnit: 2}},
	}); err != nil {
		t.Fatalf("ReplaceCountryAttrs: %v", err)
	}
	d, _ = s.GetCountry(ctx, alpha2)
	if len(d.Names) != 1 || len(d.Attrs.TimeZones) != 1 || d.Attrs.Currencies[0].Currency != "TSC" {
		t.Fatalf("names/attrs mismatch: %+v", d)
	}

	if err := s.SetCountryActive(ctx, alpha2, false); err != nil {
		t.Fatalf("SetCountryActive: %v", err)
	}
	if _, err := s.GetCountry(ctx, "QQ"); err != ErrNotFound {
		t.Fatalf("missing want ErrNotFound, got %v", err)
	}
}

// testSubdivisionCRUD 区划建/查/改/译名/停用闭环。
func testSubdivisionCRUD(ctx context.Context, t *testing.T, s *PGStore) {
	t.Helper()
	const code = "ZZ-01"
	if err := s.CreateSubdivision(ctx, Subdivision{
		Code: code, CountryCode: "ZZ", Level: 1, Category: "region", OSMAdminLevel: 4,
	}); err != nil {
		t.Fatalf("CreateSubdivision: %v", err)
	}
	if err := s.CreateSubdivision(ctx, Subdivision{
		Code: code, CountryCode: "ZZ", Level: 1, Category: "region", OSMAdminLevel: 4,
	}); err != ErrDuplicate {
		t.Fatalf("duplicate want ErrDuplicate, got %v", err)
	}

	list, err := s.ListSubdivisions(ctx, SubdivisionFilter{CountryCode: "ZZ"})
	if err != nil || len(list) != 1 {
		t.Fatalf("ListSubdivisions: %v len=%d", err, len(list))
	}
	if err := s.AddSubdivisionName(ctx, code, SubdivisionName{Locale: "en", Name: "Test Region", NameType: "STANDARD"}); err != nil {
		t.Fatalf("AddSubdivisionName: %v", err)
	}
	names, err := s.ListSubdivisionNames(ctx, code)
	if err != nil || len(names) != 1 {
		t.Fatalf("ListSubdivisionNames: %v len=%d", err, len(names))
	}
	if err := s.UpdateSubdivision(ctx, code, Subdivision{CountryCode: "ZZ",
		Level: 1, Category: "state", OSMAdminLevel: 4}); err != nil {
		t.Fatalf("UpdateSubdivision: %v", err)
	}
	if err := s.SetSubdivisionActive(ctx, code, false); err != nil {
		t.Fatalf("SetSubdivisionActive: %v", err)
	}
	if err := s.UpdateSubdivision(ctx, "ZZ-404", Subdivision{}); err != ErrNotFound {
		t.Fatalf("missing want ErrNotFound, got %v", err)
	}
}

// testImport 批量导入:插入 + 幂等重放 + is_active 不被覆盖。
func testImport(ctx context.Context, t *testing.T, s *PGStore) {
	t.Helper()
	data := ImportData{
		Countries: []Country{{Alpha2: "ZZ", Alpha3: "ZZZ", NumericCode: "999",
			ShortName: "Testland2", Status: "INDEPENDENT", ContinentCode: "AS"}},
		CountryNames: []CountryNameRow{{CountryCode: "ZZ",
			Name: CountryName{Locale: "zh-Hans", Name: "测试国2", NameType: "STANDARD"}}},
		Subdivisions: []Subdivision{{Code: "ZZ-01", CountryCode: "ZZ",
			Level: 1, Category: "province"}},
		SubdivisionNames: []SubdivisionNameRow{{SubdivisionCode: "ZZ-01",
			Name: SubdivisionName{Locale: "zh-Hans", Name: "测试省", NameType: "STANDARD"}}},
	}
	n, err := s.Import(ctx, data)
	if err != nil || n.Countries != 1 || n.Subdivisions != 1 {
		t.Fatalf("Import: %v counts=%+v", err, n)
	}
	// 幂等重放:同一载荷再次导入不报错、结果一致。
	if _, err = s.Import(ctx, data); err != nil {
		t.Fatalf("Import replay: %v", err)
	}
	d, err := s.GetCountry(ctx, "ZZ")
	if err != nil || d.ShortName != "Testland2" || len(d.Names) != 1 {
		t.Fatalf("post-import country mismatch: %+v err=%v", d, err)
	}
	// is_active 保留:导入前已停用,upsert 不应复活。
	if d.IsActive {
		t.Fatalf("import must not reactivate inactive country")
	}
	list, err := s.ListSubdivisions(ctx, SubdivisionFilter{CountryCode: "ZZ"})
	if err != nil || len(list) != 1 || list[0].Category != "province" {
		t.Fatalf("post-import subdivision mismatch: %+v err=%v", list, err)
	}
}

// testSubdivisionQuery 区划列表增强:parentCode 下钻/顶层语义/keyword/limit 截断/hasChildren 两态。
// 结构:ZZ-Q0(顶层 region) → ZZ-Q1(启用)+ZZ-Q2(停用);译名挂 ZZ-Q1(en STANDARD)。
func testSubdivisionQuery(ctx context.Context, t *testing.T, s *PGStore) {
	t.Helper()
	if err := s.CreateSubdivision(ctx, Subdivision{Code: "ZZ-Q0", CountryCode: "ZZ", Level: 1, Category: "region"}); err != nil {
		t.Fatalf("create ZZ-Q0: %v", err)
	}
	if err := s.CreateSubdivision(ctx, Subdivision{
		Code: "ZZ-Q1", CountryCode: "ZZ", ParentCode: "ZZ-Q0", Level: 2, Category: "province",
	}); err != nil {
		t.Fatalf("create ZZ-Q1: %v", err)
	}
	if err := s.CreateSubdivision(ctx, Subdivision{
		Code: "ZZ-Q2", CountryCode: "ZZ", ParentCode: "ZZ-Q0", Level: 2, Category: "province",
	}); err != nil {
		t.Fatalf("create ZZ-Q2: %v", err)
	}
	if err := s.SetSubdivisionActive(ctx, "ZZ-Q2", false); err != nil {
		t.Fatalf("deactivate ZZ-Q2: %v", err)
	}
	if err := s.AddSubdivisionName(ctx, "ZZ-Q1", SubdivisionName{Locale: "en", Name: "Quezon Region", NameType: "STANDARD"}); err != nil {
		t.Fatalf("add name: %v", err)
	}

	// parentCode 下钻:直接子节点含停用行(停用仅影响 hasChildren,不影响列表可见性)。
	q0 := "ZZ-Q0"
	kids, err := s.ListSubdivisions(ctx, SubdivisionFilter{CountryCode: "ZZ", ParentCode: &q0})
	if err != nil || len(kids) != 2 {
		t.Fatalf("drilldown: %v len=%d", err, len(kids))
	}
	byCode := map[string]bool{}
	for _, d := range kids {
		byCode[d.Code] = d.HasChildren
	}
	if !byCode["ZZ-Q1"] || byCode["ZZ-Q2"] {
		t.Fatalf("hasChildren mismatch: %v", byCode)
	}

	// 顶层语义:parentCode 传空值取 parent 为空的节点。
	empty := ""
	roots, err := s.ListSubdivisions(ctx, SubdivisionFilter{CountryCode: "ZZ", ParentCode: &empty})
	if err != nil {
		t.Fatalf("roots: %v", err)
	}
	rootIn := map[string]bool{}
	rootHas := map[string]bool{}
	for _, d := range roots {
		rootIn[d.Code] = true
		rootHas[d.Code] = d.HasChildren
	}
	if !rootIn["ZZ-Q0"] || !rootIn["ZZ-01"] {
		t.Fatalf("roots mismatch: %v", rootIn)
	}
	if !rootHas["ZZ-Q0"] {
		t.Fatalf("ZZ-Q0 should have active children")
	}

	// keyword 译名命中(ILIKE 大小写不敏感)+ locale 译名回显。
	list, err := s.ListSubdivisions(ctx, SubdivisionFilter{CountryCode: "ZZ", Locale: "en", Keyword: "quezon"})
	if err != nil || len(list) != 1 || list[0].Code != "ZZ-Q1" || list[0].DisplayName != "Quezon Region" {
		t.Fatalf("keyword name: %v %+v", err, list)
	}

	// keyword 编码命中 + limit 截断(命中 3 行只取码序第一)。
	list, err = s.ListSubdivisions(ctx, SubdivisionFilter{CountryCode: "ZZ", Keyword: "ZZ-Q", Limit: 1})
	if err != nil || len(list) != 1 || list[0].Code != "ZZ-Q0" {
		t.Fatalf("keyword code+limit: %v %+v", err, list)
	}
}

// testDefaultCountry 默认国家读:未配置→空值对象;已配置→大写 alpha-2;非法→未配置态。
func testDefaultCountry(ctx context.Context, t *testing.T, s *PGStore, pool *pgxpool.Pool) {
	t.Helper()
	cleanup := func(stage string) {
		if _, err := pool.Exec(ctx, `DELETE FROM biz_params WHERE key='geo.default_country'`); err != nil {
			t.Fatalf("cleanup param(%s): %v", stage, err)
		}
	}
	cleanup("setup")
	defer cleanup("teardown")

	d, err := s.GetDefaultCountry(ctx)
	if err != nil || d.Configured || d.CountryCode != "" {
		t.Fatalf("unconfigured: %+v err=%v", d, err)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO biz_params(key,value) VALUES ('geo.default_country','"ph"')`); err != nil {
		t.Fatalf("seed param: %v", err)
	}
	if d, err = s.GetDefaultCountry(ctx); err != nil || !d.Configured || d.CountryCode != "PH" {
		t.Fatalf("configured: %+v err=%v", d, err)
	}

	if _, err := pool.Exec(ctx, `UPDATE biz_params SET value='"ZZZ"' WHERE key='geo.default_country'`); err != nil {
		t.Fatalf("seed invalid: %v", err)
	}
	if d, err = s.GetDefaultCountry(ctx); err != nil || d.Configured || d.CountryCode != "" {
		t.Fatalf("invalid value: %+v err=%v", d, err)
	}
}
