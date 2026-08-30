package main

// helper 路由增量求值单测:udList 形态(helper 内 g.METHOD(变量))、组前缀传递、
// 直连字面量不受影响。

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

const helperFixtureSrc = `package adminapi

func RegisterThing(g *gin.RouterGroup) {
	udList(g, "/faqs")
	sub := g.Group("/geo")
	udList(sub, "/countries")
	g.GET("/direct", nil)
}

func udList(g *gin.RouterGroup, path string) {
	g.GET(path, nil)
	g.POST(path, nil)
}
`

func TestEvalHelperRoutes(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", helperFixtureSrc, 0)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			decls[fn.Name.Name] = fn
		}
	}

	paths := map[string]bool{}
	methoded := map[string]bool{}
	evalHelperRoutes(decls, paths, methoded)

	wantMethoded := []string{
		"GET /faqs", "POST /faqs",
		"GET /geo/countries", "POST /geo/countries",
		"GET /direct",
	}
	for _, k := range wantMethoded {
		if !methoded[k] {
			t.Fatalf("missing %q in %v", k, methoded)
		}
	}
	for _, p := range []string{"/faqs", "/geo/countries", "/direct"} {
		if !paths[p] {
			t.Fatalf("paths missing %q: %v", p, paths)
		}
	}
}

func TestEvalHelperRoutesUnknownVarNotGuessed(t *testing.T) {
	// helper 内路径来自调用点未绑定的变量(如拼接/常量):不得瞎猜,保持静默跳过。
	src := `package adminapi

func RegisterOther(g *gin.RouterGroup) {
	udOther(g)
}

func udOther(g *gin.RouterGroup) {
	path := "/from-const"
	g.GET(path, nil)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			decls[fn.Name.Name] = fn
		}
	}
	paths := map[string]bool{}
	methoded := map[string]bool{}
	evalHelperRoutes(decls, paths, methoded)
	if len(paths) != 0 || len(methoded) != 0 {
		t.Fatalf("unbound variable path must not be guessed: %v / %v", paths, methoded)
	}
}

// 回归(A6):for-range 字符串字面量切片注册(stripe.go /pay/stripe/{done,cancel}
// 形态)必须逐值展开;多组形参入口(uauth 与 pub 并存)各自可见。
func TestEvalRangeLoopRoutes(t *testing.T) {
	src := `package userapi

func RegisterStripe(pub, uauth *gin.RouterGroup, a *app.Application) {
	uauth.POST("/payments/stripe/intent", nil)
	registerStripePayDone(pub)
}

func registerStripePayDone(pub *gin.RouterGroup) {
	for _, p := range []string{"/pay/stripe/done", "/pay/stripe/cancel"} {
		pub.GET(p, nil)
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	decls := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			decls[fn.Name.Name] = fn
		}
	}
	paths := map[string]bool{}
	methoded := map[string]bool{}
	evalHelperRoutes(decls, paths, methoded)
	for _, k := range []string{
		"POST /payments/stripe/intent",
		"GET /pay/stripe/done", "GET /pay/stripe/cancel",
	} {
		if !methoded[k] {
			t.Fatalf("missing %q in %v", k, methoded)
		}
	}
}
