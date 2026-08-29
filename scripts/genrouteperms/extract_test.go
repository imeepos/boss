package main

// genrouteperms 提取逻辑单测:合成 fixture 锁定每种注册形态的语义,
// 防止新注册形态/重构改变提取结果而 --check 漂移门禁无法察觉(它只比对产物)。
// 另含真实树 sanity:当前 internal/httpapi/admin 提取数与关键绑定抽查。

import (
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// writeFixture 写合成注册源码到临时目录并完成提取。
func writeFixture(t *testing.T, src string) map[string]map[string]bool {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "routes.go"), []byte("package adminapi\n\n"+src), 0o644); err != nil {
		t.Fatal(err)
	}
	g, err := newGen(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.extract(); err != nil {
		t.Fatal(err)
	}
	return g.routes
}

func expectRoute(t *testing.T, routes map[string]map[string]bool, key string, want []string) {
	t.Helper()
	got, ok := routes[key]
	if !ok {
		t.Fatalf("缺路由 %s; 实有 %d 条", key, len(routes))
	}
	gotSlice := make([]string, 0, len(got))
	for c := range got {
		gotSlice = append(gotSlice, c)
	}
	sort.Strings(gotSlice)
	if strings.Join(gotSlice, ",") != strings.Join(want, ",") {
		t.Fatalf("%s = [%s], want [%s]", key, strings.Join(gotSlice, ","), strings.Join(want, ","))
	}
}

func TestExtractInlineAndVarPerm(t *testing.T) {
	routes := writeFixture(t, `
func Register(r *gin.Engine, a *app.Application) {
	g := r.Group("/api/admin/v1")
	g.GET("/open", openHandler(a))
	g.GET("/inline", requirePerm(a.User, "menu:inline"), h(a))
	perm := requirePerm(a.User, "menu:var")
	g.POST("/var", perm, h(a))
}
`)
	expectRoute(t, routes, "GET /api/admin/v1/open", nil)
	expectRoute(t, routes, "GET /api/admin/v1/inline", []string{"menu:inline"})
	expectRoute(t, routes, "POST /api/admin/v1/var", []string{"menu:var"})
}

func TestExtractGroupPermAndPrefix(t *testing.T) {
	routes := writeFixture(t, `
func Register(r *gin.Engine, a *app.Application) {
	g := r.Group("/api/admin/v1")
	sub := g.Group("/orders", requirePerm(a.User, "menu:order"))
	sub.GET("", listHandler(a))
	sub.GET("/:orderNo", getHandler(a))
	plain := g.Group("/misc")
	plain.GET("/ping", pingHandler(a))
}
`)
	expectRoute(t, routes, "GET /api/admin/v1/orders", []string{"menu:order"})
	expectRoute(t, routes, "GET /api/admin/v1/orders/{orderNo}", []string{"menu:order"})
	expectRoute(t, routes, "GET /api/admin/v1/misc/ping", nil)
}

func TestExtractUseInheritance(t *testing.T) {
	routes := writeFixture(t, `
func Register(r *gin.Engine, a *app.Application) {
	g := r.Group("/api/admin/v1")
	g.Use(requirePerm(a.User, "menu:golbal"))
	g.GET("/early", h(a))
	g.GET("/late", h(a))
}
`)
	expectRoute(t, routes, "GET /api/admin/v1/early", []string{"menu:golbal"})
	expectRoute(t, routes, "GET /api/admin/v1/late", []string{"menu:golbal"})
}

func TestExtractCrossFunctionAndReturnValue(t *testing.T) {
	routes := writeFixture(t, `
func Register(r *gin.Engine, a *app.Application) {
	g := r.Group("/api/admin/v1")
	authed := registerAuthRoot(g, a)
	registerOrders(authed, a)
}

func registerAuthRoot(g *gin.RouterGroup, a *app.Application) *gin.RouterGroup {
	g.POST("/auth/login", loginHandler(a))
	authed := g.Group("")
	authed.GET("/auth/me", meHandler(a))
	return authed
}

func registerOrders(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:order")
	g.GET("/orders", perm, listHandler(a))
	registerOrderItem(g, a, perm)
}

func registerOrderItem(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/orders/:orderNo", perm, getHandler(a))
}
`)
	expectRoute(t, routes, "POST /api/admin/v1/auth/login", nil)
	expectRoute(t, routes, "GET /api/admin/v1/auth/me", nil)
	expectRoute(t, routes, "GET /api/admin/v1/orders", []string{"menu:order"})
	expectRoute(t, routes, "GET /api/admin/v1/orders/{orderNo}", []string{"menu:order"})
}

func TestExtractStringIntermediates(t *testing.T) {
	routes := writeFixture(t, `
func Register(r *gin.RouterGroup, a *app.Application) {
	udList(r, a, "/faqs", "menu:userdata", listFn)
}

func udList(g *gin.RouterGroup, a *app.Application, path, perm string, fn interface{}) {
	g.GET(path, requirePerm(a.User, perm), wrap(fn))
}
`)
	expectRoute(t, routes, "GET /faqs", []string{"menu:userdata"})
}

func TestExtractBranchUnion(t *testing.T) {
	routes := writeFixture(t, `
func Register(r *gin.RouterGroup, a *app.Application) {
	if a.Backup == nil {
		r.GET("/backup/tables", unavailableHandler(a))
		return
	}
	perm := requirePerm(a.User, "menu:backup")
	r.GET("/backup/tables", perm, listHandler(a))
}
`)
	// 并集语义:任一分支要求该码即保留(空集分支不降级权限要求)
	expectRoute(t, routes, "GET /backup/tables", []string{"menu:backup"})
}

func TestExtractCycleGuard(t *testing.T) {
	routes := writeFixture(t, `
func Register(r *gin.RouterGroup, a *app.Application) {
	aCallsB(r, a)
}

func aCallsB(g *gin.RouterGroup, a *app.Application) {
	g.GET("/from-a", h(a))
	bCallsA(g, a)
}

func bCallsA(g *gin.RouterGroup, a *app.Application) {
	aCallsB(g, a)
}
`)
	expectRoute(t, routes, "GET /from-a", nil)
	if len(routes) != 1 {
		t.Fatalf("环防护失效: %d 条", len(routes))
	}
}

func TestRenderDeterministicAndFormatted(t *testing.T) {
	in := [][2]any{
		{"GET /b", []string{"menu:a", "menu:b"}},
		{"GET /a", []string{}},
	}
	once := render(in)
	twice := render(in)
	if once != twice {
		t.Fatal("render 非幂等,--check 比对不可靠")
	}
	if !strings.Contains(once, `"GET /a": {},`) {
		t.Fatalf("空码集路由渲染缺失: %s", once)
	}
	if _, err := format.Source([]byte(once)); err != nil {
		t.Fatalf("render 产物未过 go/format: %v", err)
	}
}

func TestExtractRealTreeSanity(t *testing.T) {
	g, err := newGen(filepath.Join("..", "..", "internal", "httpapi", "admin"))
	if err != nil {
		t.Skipf("真实树不可达(非仓库根执行): %v", err)
	}
	if err := g.extract(); err != nil {
		t.Fatal(err)
	}
	if len(g.routes) < 400 {
		t.Fatalf("真实树仅提取 %d 条,解析逻辑与源码脱节", len(g.routes))
	}
	expectRoute(t, g.routes, "GET /api/admin/v1/orders", []string{"menu:order"})
	expectRoute(t, g.routes, "GET /api/admin/v1/bills", []string{"menu:billing"})
	expectRoute(t, g.routes, "POST /api/admin/v1/auth/login", nil)
}
