package main

// AST 提取:路由注册 → 路由键 → permCode 全集。
// 支持的注册形态(与 internal/httpapi/admin 现状一一对应,出现新形态须在此补解析并 --check 拦截):
//   g.GET("/p", requirePerm(a.User, "menu:x"), h)   内联门禁
//   perm := requirePerm(a.User, "menu:x")           var 门禁(同函数内传引用)
//   sub := g.Group("/p", requirePerm(...))          组级门禁(前缀累积)
//   g.Use(requirePerm(...))                          组后续路由门禁
//   registerSub(g, a, perm)                          跨 register 函数传组/门禁

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strings"
)

var httpMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true}

var paramRe = regexp.MustCompile(`:([a-zA-Z0-9_]+)`)

// gen 单次提取会话:包内函数表 + 路由权限集。
type gen struct {
	funcs  map[string]*ast.FuncDecl
	routes map[string]map[string]bool // "GET /orders" → permCode 集合
}

// newGen 解析目录下全部非测试 Go 文件,建函数名索引。
func newGen(dir string) (*gen, error) {
	g := &gen{funcs: map[string]*ast.FuncDecl{}, routes: map[string]map[string]bool{}}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool { return !fi.IsDir() && !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, d := range f.Decls {
				if fn, ok := d.(*ast.FuncDecl); ok && fn.Name != nil {
					g.funcs[fn.Name.Name] = fn
				}
			}
		}
	}
	return g, nil
}

// extract 从未被其他函数调用的组参数函数(register 入口)出发求值全图。
func (g *gen) extract() error {
	called := map[string]bool{}
	for _, fn := range g.funcs {
		ast.Inspect(fn, func(n ast.Node) bool {
			if ce, ok := n.(*ast.CallExpr); ok {
				if id, ok := ce.Fun.(*ast.Ident); ok {
					called[id.Name] = true
				}
			}
			return true
		})
	}
	entries := 0
	for name, fn := range g.funcs {
		if called[name] || !hasGroupParam(fn) {
			continue
		}
		entries++
		g.evalFunc(fn, nil, map[string]bool{name: true})
	}
	if entries == 0 {
		return fmt.Errorf("未找到 register 入口(带 *gin.RouterGroup 参数且未被调用)")
	}
	return nil
}

// argVal 求值期实参:组值/门禁码集/字符串值(路径或权限码经变量中转)。
type argVal struct {
	group *groupVal
	perm  map[string]bool
	str   *string
}

// groupVal 组求值态:路径前缀 + 组级/Use 门禁码集。
type groupVal struct {
	prefix string
	perms  map[string]bool
}

// evalFunc 在实参环境下求值单个函数体;返回 return 出的组值;visiting 防环。
// 组型参数无实参时(入口 Register 的 *gin.Engine)绑定为空根组。
func (g *gen) evalFunc(fn *ast.FuncDecl, args []argVal, visiting map[string]bool) argVal {
	scope := map[string]argVal{}
	ai := 0
	for _, p := range fn.Type.Params.List {
		for _, n := range p.Names {
			if ai < len(args) {
				scope[n.Name] = args[ai]
			} else if isGroupType(p.Type) {
				scope[n.Name] = argVal{group: &groupVal{perms: map[string]bool{}}}
			}
			ai++
		}
	}
	return g.walkStmts(fn.Body.List, scope, visiting)
}

// isGroupType 类型是否为 *gin.RouterGroup / *gin.Engine。
func isGroupType(t ast.Expr) bool {
	star, ok := t.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	return ok && (sel.Sel.Name == "RouterGroup" || sel.Sel.Name == "Engine")
}

// walkStmts 顺序处理语句块;if/块递归(注册分支并集,权限码并集偏保守放行);
// 遇 return 出组值则回传(assign 链 `authed := registerAdminAuthRoot(r,...)` 依赖此值)。
func (g *gen) walkStmts(list []ast.Stmt, scope map[string]argVal, visiting map[string]bool) argVal {
	var ret argVal
	for _, st := range list {
		switch v := st.(type) {
		case *ast.AssignStmt:
			g.walkAssign(v, scope, visiting)
		case *ast.ExprStmt:
			g.walkExprStmt(v.X, scope, visiting)
		case *ast.ReturnStmt:
			if ret.group == nil && ret.perm == nil && len(v.Results) > 0 {
				if id, ok := v.Results[0].(*ast.Ident); ok {
					ret = scope[id.Name]
				}
			}
		case *ast.IfStmt:
			g.walkStmts([]ast.Stmt{v.Body}, scope, visiting)
			if v.Else != nil {
				g.walkStmts([]ast.Stmt{v.Else}, scope, visiting)
			}
		case *ast.BlockStmt:
			g.walkStmts(v.List, scope, visiting)
		}
	}
	return ret
}

// walkAssign 处理 x := <rhs>:组建组值;requirePerm 建门禁值;register 调用递归。
func (g *gen) walkAssign(st *ast.AssignStmt, scope map[string]argVal, visiting map[string]bool) {
	if st.Tok != token.DEFINE || len(st.Lhs) != 1 || len(st.Rhs) != 1 {
		return
	}
	name, ok := st.Lhs[0].(*ast.Ident)
	if !ok {
		return
	}
	ce, ok := st.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}
	if gv, ok := g.asGroupCall(ce, scope); ok {
		scope[name.Name] = argVal{group: gv}
		return
	}
	if codes, ok := asRequirePerm(ce, scope); ok {
		scope[name.Name] = argVal{perm: codes}
		return
	}
	if s, ok := strValue(ce, scope); ok {
		scope[name.Name] = argVal{str: &s}
		return
	}
	// x := registerSub(g, a):被调方注册路由(副作用)并可能回传组值。
	if ret := g.callFunc(ce, scope, visiting); ret.group != nil || ret.perm != nil {
		scope[name.Name] = ret
	}
}

// strValue 求值字符串表达式:字面量或绑定字符串值的 ident(路径/权限码中转变量)。
func strValue(e ast.Expr, scope map[string]argVal) (string, bool) {
	if id, ok := e.(*ast.Ident); ok {
		if av, inScope := scope[id.Name]; inScope && av.str != nil {
			return *av.str, true
		}
		return "", false
	}
	s, ok := strLit(e)
	return s, ok
}

// walkExprStmt 处理表达式语句:路由注册/Use/跨函数调用。
func (g *gen) walkExprStmt(x ast.Expr, scope map[string]argVal, visiting map[string]bool) {
	ce, ok := x.(*ast.CallExpr)
	if !ok {
		return
	}
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok {
		g.callFunc(ce, scope, visiting)
		return
	}
	recv, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}
	m := sel.Sel.Name
	av, inScope := scope[recv.Name]
	switch {
	case httpMethods[m] && inScope && av.group != nil:
		g.recordRoute(av.group, m, ce, scope)
	case m == "Use" && inScope && av.group != nil:
		mergeCodes(av.group.perms, permArgs(ce.Args, scope))
	default:
		g.callFunc(ce, scope, visiting)
	}
}

// asGroupCall 识别 x.Group(path, 门禁...) 求值为新组值;x 须为已知组。
func (g *gen) asGroupCall(ce *ast.CallExpr, scope map[string]argVal) (*groupVal, bool) {
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Group" || len(ce.Args) == 0 {
		return nil, false
	}
	recv, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil, false
	}
	av, inScope := scope[recv.Name]
	if !inScope || av.group == nil {
		return nil, false
	}
	prefix, ok := strValue(ce.Args[0], scope)
	if !ok {
		return nil, false
	}
	codes := map[string]bool{}
	mergeCodes(codes, av.group.perms)
	mergeCodes(codes, permArgs(ce.Args[1:], scope))
	return &groupVal{prefix: joinPath(av.group.prefix, prefix), perms: codes}, true
}

// callFunc 跨 register 函数调用:实参含组/门禁值才递归(handler 调用无组值自然跳过);
// 返回被调方 return 的组值(如 registerAdminAuthRoot)。
func (g *gen) callFunc(ce *ast.CallExpr, scope map[string]argVal, visiting map[string]bool) argVal {
	id, ok := ce.Fun.(*ast.Ident)
	if !ok {
		return argVal{}
	}
	fn, ok := g.funcs[id.Name]
	if !ok || visiting[id.Name] || !hasGroupParam(fn) {
		return argVal{}
	}
	args := make([]argVal, len(ce.Args))
	any := false
	for ai, a := range ce.Args {
		args[ai] = g.resolveArg(a, scope)
		any = any || args[ai].group != nil || args[ai].perm != nil || args[ai].str != nil
	}
	if !any {
		return argVal{}
	}
	next := map[string]bool{id.Name: true}
	for k := range visiting {
		next[k] = true
	}
	return g.evalFunc(fn, args, next)
}

// resolveArg 实参求值:组 ident → 组值;requirePerm/门禁 ident → 码集;字符串字面量/ident → 串值;其余零值。
func (g *gen) resolveArg(a ast.Expr, scope map[string]argVal) argVal {
	if ce, ok := a.(*ast.CallExpr); ok {
		if codes, ok := asRequirePerm(ce, scope); ok {
			return argVal{perm: codes}
		}
	}
	if id, ok := a.(*ast.Ident); ok {
		if av, inScope := scope[id.Name]; inScope {
			return av
		}
		return argVal{}
	}
	if s, ok := strLit(a); ok {
		return argVal{str: &s}
	}
	return argVal{}
}

// recordRoute 登记一条路由:全路径 = 组前缀 + 注册路径;码集 = 组级 ∪ 路由级。
// 同键多次登记(条件分支)取并集:任一路径要求该码即保留,避免漏配导致目录面超发。
func (g *gen) recordRoute(gv *groupVal, method string, ce *ast.CallExpr, scope map[string]argVal) {
	if len(ce.Args) == 0 {
		return
	}
	p, ok := strValue(ce.Args[0], scope)
	if !ok {
		return
	}
	codes := map[string]bool{}
	mergeCodes(codes, gv.perms)
	mergeCodes(codes, permArgs(ce.Args[1:], scope))
	key := method + " " + joinPath(gv.prefix, paramRe.ReplaceAllString(p, "{$1}"))
	if g.routes[key] == nil {
		g.routes[key] = codes
		return
	}
	mergeCodes(g.routes[key], codes)
}

// permArgs 收集实参里的门禁码:requirePerm 内联或已求值门禁 ident。
func permArgs(args []ast.Expr, scope map[string]argVal) map[string]bool {
	codes := map[string]bool{}
	for _, a := range args {
		if ce, ok := a.(*ast.CallExpr); ok {
			if c, ok := asRequirePerm(ce, scope); ok {
				mergeCodes(codes, c)
			}
			continue
		}
		if id, ok := a.(*ast.Ident); ok {
			if av, inScope := scope[id.Name]; inScope && av.perm != nil {
				mergeCodes(codes, av.perm)
			}
		}
	}
	return codes
}

// asRequirePerm 识别 requirePerm(a.User, <code>):code 为字面量或字符串值 ident
//(requirePerm 是包内唯一 Authz 包装点)。
func asRequirePerm(ce *ast.CallExpr, scope map[string]argVal) (map[string]bool, bool) {
	id, ok := ce.Fun.(*ast.Ident)
	if !ok || id.Name != "requirePerm" || len(ce.Args) < 2 {
		return nil, false
	}
	code, ok := strValue(ce.Args[1], scope)
	if !ok {
		return nil, false
	}
	return map[string]bool{code: true}, true
}

// hasGroupParam 函数是否带 gin 组/引擎参数(*gin.RouterGroup 或 *gin.Engine)。
func hasGroupParam(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}
	for _, p := range fn.Type.Params.List {
		if isGroupType(p.Type) {
			return true
		}
	}
	return false
}

// strLit 字符串字面量提取(不带反引号 raw string 时同样取值)。
func strLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	return strings.Trim(lit.Value, "`\""), true
}

// mergeCodes dst ∪= src。
func mergeCodes(dst, src map[string]bool) {
	for c := range src {
		dst[c] = true
	}
}

// joinPath gin 组前缀拼接(b 恒以 / 开头;a 空为根)。
func joinPath(a, b string) string {
	return strings.TrimRight(a, "/") + b
}

// sortedRoutes 稳定输出:键字典序,码集字典序切片。
func (g *gen) sortedRoutes() [][2]any {
	keys := make([]string, 0, len(g.routes))
	for k := range g.routes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([][2]any, 0, len(keys))
	for _, k := range keys {
		codes := make([]string, 0, len(g.routes[k]))
		for c := range g.routes[k] {
			codes = append(codes, c)
		}
		sort.Strings(codes)
		out = append(out, [2]any{k, codes})
	}
	return out
}
