package geo

import (
	"context"
	"os"
	"testing"

	"github.com/ymm-001/boss/internal/pkg/database"
)

// TestODNRegionCityCodes_Integration 校验 migrations/000075 ODN PRV/城市前缀映射
// 与 PSGC 权威数据的 FK 完整性与关键样本(需真实 PostgreSQL,DSN 未设置时跳过)。
func TestODNRegionCityCodes_Integration(t *testing.T) {
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

	var nRegion, nCity int
	if err := pool.QueryRow(ctx, "select count(*) from odn_region_code").Scan(&nRegion); err != nil {
		t.Fatalf("count odn_region_code: %v", err)
	}
	if err := pool.QueryRow(ctx, "select count(*) from odn_city_code").Scan(&nCity); err != nil {
		t.Fatalf("count odn_city_code: %v", err)
	}
	// 规范:83 省 - 1 预留位(PHL075) = 82;城市前缀 119(规范索引全量)。
	if nRegion != 82 {
		t.Fatalf("odn_region_code 期望 82 行,实际 %d", nRegion)
	}
	if nCity != 119 {
		t.Fatalf("odn_city_code 期望 119 行,实际 %d", nCity)
	}

	// FK 完整性:所有 psgc_code 必须存在于 geo_subdivision。
	var dangling int
	err = pool.QueryRow(ctx, `select
		(select count(*) from odn_region_code r left join geo_subdivision g on g.code=r.psgc_code where g.code is null)
	+	(select count(*) from odn_city_code c left join geo_subdivision g on g.code=c.psgc_code where g.code is null)`).Scan(&dangling)
	if err != nil {
		t.Fatalf("dangling check: %v", err)
	}
	if dangling != 0 {
		t.Fatalf("悬挂 psgc 引用 %d 条", dangling)
	}

	// 关键样本抽查。
	samples := []struct{ prv, prefix, want string }{
		{"PHL001", "MNL", "PH-1380600000"}, // 马尼拉
		{"PHL001", "QZN", "PH-1381300000"}, // 奎松市
		{"PHL047", "CEB", "PH-0730600000"}, // 宿务市(HUC 直辖中米沙鄢)
		{"PHL066", "DVO", "PH-1130700000"}, // 达沃市(HUC)
	}
	for _, s := range samples {
		var got string
		err := pool.QueryRow(ctx,
			"select psgc_code from odn_city_code where prv_code=$1 and city_prefix=$2",
			s.prv, s.prefix).Scan(&got)
		if err != nil || got != s.want {
			t.Fatalf("%s/%s 期望 %s,实际 %s(err=%v)", s.prv, s.prefix, s.want, got, err)
		}
	}
}
