package geo

import (
	"context"
	"os"
	"testing"

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

	list, err := s.ListSubdivisions(ctx, "ZZ", "")
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
