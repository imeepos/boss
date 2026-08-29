package apiroutes

// 目录-权限映射一致性测试:
//  1. admin 目录(openapi 投影)每条路由必须在 AdminRoutePerms(Go 注册投影)中有据可查,
//     否则目录在暴露服务端不存在的路由(404 误导面);
//  2. 过滤语义:须持有全部权限码才可见,无码路由恒可见;
//  3. user/worker 不受权限过滤影响。

import (
	"strings"
	"testing"
)

func TestAdminCatalogRoutesAllRegistered(t *testing.T) {
	dir := ByName("admin")
	if dir == nil {
		t.Fatal("admin 端目录缺失(再生成: node scripts/gen-bossctl-routes.mjs)")
	}
	if len(AdminRoutePerms) == 0 {
		t.Fatal("AdminRoutePerms 为空(再生成: go run ./scripts/genrouteperms)")
	}
	for _, r := range dir.Routes {
		key := RouteKey(dir, r)
		if _, ok := AdminRoutePerms[key]; !ok {
			t.Errorf("目录路由无注册凭据: %s(服务端会 404)", key)
		}
	}
}

func TestAdminPermsCoveredCatalogRoutes(t *testing.T) {
	// 抽查权限绑定与 openapi 域注释一致(样例覆盖各权限域)。
	samples := map[string]string{
		"GET /api/admin/v1/orders":         "menu:order",
		"GET /api/admin/v1/bills":          "menu:billing",
		"GET /api/admin/v1/dispatch/pool":  "menu:dispatch",
		"GET /api/admin/v1/api-keys":       "menu:apikey",
		"GET /api/admin/v1/ports":          "menu:resource",
		"GET /api/admin/v1/legal-entities": "menu:company",
		"GET /api/admin/v1/auth/me":        "",
		"POST /api/admin/v1/auth/login":    "",
	}
	for key, want := range samples {
		got, ok := AdminRoutePerms[key]
		if !ok {
			t.Errorf("映射缺路由: %s", key)
			continue
		}
		joined := strings.Join(got, ",")
		if want == "" && joined != "" {
			t.Errorf("%s 应无门禁,实得 %s", key, joined)
		}
		if want != "" && joined != want {
			t.Errorf("%s 门禁 %s, want %s", key, joined, want)
		}
	}
}

func TestFilterByPerms(t *testing.T) {
	dir := ByName("admin")
	all := dir.Routes
	if len(all) == 0 {
		t.Fatal("admin 目录为空")
	}

	perm := func(codes ...string) map[string]bool {
		m := map[string]bool{}
		for _, c := range codes {
			m[c] = true
		}
		return m
	}

	// 全量权限(映射表出现过的所有码)目录全量可见
	allCodes := map[string]bool{}
	for _, codes := range AdminRoutePerms {
		for _, c := range codes {
			allCodes[c] = true
		}
	}
	full := FilterByPerms(dir, all, allCodes)
	if len(full) != len(all) {
		t.Fatalf("全量权限过滤后 %d/%d, 不应丢失路由", len(full), len(all))
	}

	// 零权限:仅剩无门禁路由(auth 自服务/公开),且必不含门禁路由
	none := FilterByPerms(dir, all, map[string]bool{})
	if len(none) == 0 || len(none) >= len(all) {
		t.Fatalf("零权限目录 %d 条,应在 (0,%d) 开区间", len(none), len(all))
	}
	for _, r := range none {
		if codes := RoutePerms(dir, r); len(codes) != 0 {
			t.Fatalf("零权限账号看到门禁路由 %s %s(%v)", r.Method, r.Path, codes)
		}
	}

	// 单一权限:含该码路由全可见,无该码门禁路由不可见
	orderOnly := FilterByPerms(dir, all, perm("menu:order"))
	hasOrder, hasBilling := false, false
	for _, r := range orderOnly {
		switch RouteKey(dir, r) {
		case "GET /api/admin/v1/orders":
			hasOrder = true
		case "GET /api/admin/v1/bills":
			hasBilling = true
		}
	}
	if !hasOrder || hasBilling {
		t.Fatalf("menu:order 过滤失真: hasOrder=%v hasBilling=%v", hasOrder, hasBilling)
	}
}

func TestUserWorkerUnfiltered(t *testing.T) {
	for _, name := range []string{"user", "worker"} {
		dir := ByName(name)
		if dir == nil {
			t.Fatalf("%s 端目录缺失", name)
		}
		for _, r := range dir.Routes {
			if codes := RoutePerms(dir, r); len(codes) != 0 {
				t.Fatalf("%s 端 %s %s 不应有权限码: %v", name, r.Method, r.Path, codes)
			}
		}
	}
}
