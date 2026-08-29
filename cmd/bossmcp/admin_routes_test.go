package main

// admin 端目录过滤与身份绑定测试(httptest 假后端,不经 LLM):
// 目录面必须 = 当前账号权限;权限查询失败一律拒发目录;BOSS_API_KEY 不得兜底 admin 端。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ymm-001/boss/internal/apiclient"
)

// adminBackend 按路径前缀分流:/auth/me 回 meJSON,其余回 okJSON。
func adminBackend(t *testing.T, meJSON string, meStatus int) (*httptest.Server, *recording) {
	t.Helper()
	rec := &recording{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.Path, rec.Key = r.URL.Path, r.Header.Get("X-API-Key")
		if strings.HasSuffix(r.URL.Path, "/auth/me") {
			w.WriteHeader(meStatus)
			_, _ = w.Write([]byte(meJSON))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":null}`))
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// serverWithAdmin 装配 admin 端指向 backend。
func serverWithAdmin(backend, adminKey string) *server {
	s := newServer(func(string) string { return "" })
	s.Portals["admin"] = &portalCfg{Name: "admin", Prefix: "/api/admin/v1",
		Client: apiclient.New(backend, adminKey), EnvKeys: []string{"BOSS_ADMIN_API_KEY"}}
	return s
}

// routeLines 目录正文中路由行集合("METHOD /path")。
func routeLines(text string) map[string]bool {
	out := map[string]bool{}
	for _, ln := range strings.Split(text, "\n") {
		f := strings.Fields(ln)
		if len(f) >= 2 && isRouteMethod(f[0]) {
			out[f[0]+" "+f[1]] = true
		}
	}
	return out
}

// isRouteMethod 目录行首字段是否为 HTTP 方法。
func isRouteMethod(s string) bool {
	switch s {
	case "GET", "POST", "PUT", "DELETE", "PATCH":
		return true
	}
	return false
}

func TestAdminRoutesFilteredByPermission(t *testing.T) {
	me := `{"code":0,"msg":"ok","data":{"accountId":257,"username":"dispatch_li","realName":"李调度",
		"roleCode":"technician","permissionCodes":["menu:order","menu:dispatch","menu:resource"]}}`
	srv, _ := adminBackend(t, me, http.StatusOK)
	s := serverWithAdmin(srv.URL, "boss_admin_key")

	text, isErr := callTool(t, s, "boss_routes", `{"portal":"admin"}`)
	if isErr {
		t.Fatalf("unexpected error: %s", text)
	}
	if !strings.Contains(text, "dispatch_li") || !strings.Contains(text, "role=technician") {
		t.Errorf("identity missing: %s", text[:200])
	}
	lines := routeLines(text)
	if !lines["GET /orders"] {
		t.Error("menu:order 账号应看到 GET /orders")
	}
	if lines["GET /bills"] {
		t.Error("无 menu:billing 不得看到 GET /bills")
	}
	if lines["GET /api-keys"] {
		t.Error("无 menu:apikey 不得看到 GET /api-keys")
	}
	if !lines["GET /auth/me"] {
		t.Error("无门禁路由 GET /auth/me 应保留")
	}
	if !strings.Contains(text, "403") {
		t.Error("目录应说明权限外接口将被 403")
	}
}

func TestAdminRoutesFilterParamAppliesAfterPerm(t *testing.T) {
	me := `{"code":0,"msg":"ok","data":{"accountId":258,"username":"cashier_wang","roleCode":"ops",
		"permissionCodes":["menu:billing"]}}`
	srv, _ := adminBackend(t, me, http.StatusOK)
	s := serverWithAdmin(srv.URL, "k")

	text, isErr := callTool(t, s, "boss_routes", `{"portal":"admin","filter":"orders"}`)
	if isErr {
		t.Fatalf("unexpected error: %s", text)
	}
	if strings.Contains(text, "\nGET /orders") {
		t.Error("filter 命中不得越过权限过滤:menu:billing 账号不应见 GET /orders")
	}
}

func TestAdminRoutesTemplateKeySeesTemplateSubset(t *testing.T) {
	me := `{"code":0,"msg":"ok","data":{"accountId":300,"username":"partner_bot","roleCode":"partner",
		"templateCode":"partner-orders-read","permissionCodes":["menu:partner-orders"]}}`
	srv, _ := adminBackend(t, me, http.StatusOK)
	s := serverWithAdmin(srv.URL, "k")

	text, isErr := callTool(t, s, "boss_routes", `{"portal":"admin"}`)
	if isErr {
		t.Fatalf("unexpected error: %s", text)
	}
	lines := routeLines(text)
	if !lines["GET /partner/orders"] && !lines["GET /partner/commissions"] {
		// 模板账号至少应看到 menu:partner-orders 域的一条路由或仅剩无门禁路由
		if len(lines) > 40 {
			t.Errorf("受限模板目录过大: %d 行", len(lines))
		}
	}
	if !strings.Contains(text, "partner-orders-read") {
		t.Error("应标注受限模板")
	}
}

func TestAdminRoutesFailsClosedWhenPermUnknown(t *testing.T) {
	srv, _ := adminBackend(t, `{"code":40100,"msg":"invalid api key"}`, http.StatusOK)
	s := serverWithAdmin(srv.URL, "bad_key")

	text, isErr := callTool(t, s, "boss_routes", `{"portal":"admin"}`)
	if !isErr {
		t.Fatal("权限查询失败必须拒发目录")
	}
	if !strings.Contains(text, "BOSS_ADMIN_API_KEY") || strings.Contains(text, "\nGET ") {
		t.Errorf("fail-closed 泄目录: %s", text[:min(200, len(text))])
	}
}

func TestAdminRoutesMissingKeyFailsFast(t *testing.T) {
	srv, _ := adminBackend(t, `{}`, http.StatusOK)
	s := serverWithAdmin(srv.URL, "")

	text, isErr := callTool(t, s, "boss_routes", `{"portal":"admin"}`)
	if !isErr || !strings.Contains(text, "BOSS_ADMIN_API_KEY") {
		t.Errorf("isErr=%v text=%s", isErr, text)
	}
}

func TestAdminWhoamiUsesAuthMe(t *testing.T) {
	me := `{"code":0,"msg":"ok","data":{"accountId":1,"username":"admin","permissionCodes":["menu:order"]}}`
	srv, rec := adminBackend(t, me, http.StatusOK)
	s := serverWithAdmin(srv.URL, "boss_admin_key")

	text, isErr := callTool(t, s, "boss_whoami", `{"portal":"admin"}`)
	if isErr {
		t.Fatalf("unexpected error: %s", text)
	}
	if rec.Path != "/api/admin/v1/auth/me" || rec.Key != "boss_admin_key" {
		t.Errorf("path=%s key=%s", rec.Path, rec.Key)
	}
}

func TestResolveKeyAdminNoFallback(t *testing.T) {
	env := map[string]string{"BOSS_API_KEY": "fallback", "BOSS_ADMIN_API_KEY": "adm"}
	getenv := func(k string) string { return env[k] }
	if key, name := resolveKey(getenv, "admin"); key != "adm" || name != "BOSS_ADMIN_API_KEY" {
		t.Errorf("admin = %q(%q)", key, name)
	}
	delete(env, "BOSS_ADMIN_API_KEY")
	if key, name := resolveKey(getenv, "admin"); key != "" || name != "BOSS_ADMIN_API_KEY" {
		t.Errorf("admin 无专属 key 时不得用 BOSS_API_KEY 兜底: %q(%q)", key, name)
	}
}

func TestAdminCallPassesAdminKey(t *testing.T) {
	srv, rec := adminBackend(t, `{"code":0,"msg":"ok","data":null}`, http.StatusOK)
	s := serverWithAdmin(srv.URL, "boss_admin_key")

	text, isErr := callTool(t, s, "boss_call", `{"portal":"admin","method":"GET","path":"/orders"}`)
	if isErr {
		t.Fatalf("unexpected error: %s", text)
	}
	if rec.Path != "/api/admin/v1/orders" || rec.Key != "boss_admin_key" {
		t.Errorf("path=%s key=%s", rec.Path, rec.Key)
	}
	if _, isErr := callTool(t, s, "boss_call", `{"portal":"admin","method":"GET","path":"/api/user/v1/orders"}`); !isErr {
		t.Error("跨端路径必须拒绝")
	}
}

// min 逐字面量最小值(兼容老工具链时可直接内联)。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 防御:toolDefs 的 portal 枚举必须含 admin(防手滑回退)。
func TestToolDefsIncludeAdminPortal(t *testing.T) {
	raw, _ := json.Marshal(toolDefs())
	if !strings.Contains(string(raw), `"admin"`) {
		t.Error("tools/list 缺 admin 端枚举")
	}
}
