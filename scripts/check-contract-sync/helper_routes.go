// helper 路由增量求值:补齐"通用 helper 经变量注册路由"形态。
// 背景:udList(g, a, "/faqs", perm, fn) 这类 helper 内部 g.GET(path,...),
// routesFromDecl 只识别 register 函数内的字面量调用,漏检此类路由——
// A2 曾据此产生 GET /faqs 假阳性豁免。本文件做**增量**求值:从 register
// 入口出发,遇到对"首参为 *gin.RouterGroup 的本地函数"的调用且组实参
// 前缀已知时,把调用点字符串字面量按位置绑定到形参,递归求值被调函数体。
package main

import (
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

// evalCtx 求值上下文:groupPrefix = 被调函数组形参名→已积累前缀;
// strArgs = 形参名→调用点绑定的字符串字面量。
type evalCtx struct {
	groupPrefix map[string]string
	strArgs     map[string]string
}

type helperEval struct {
	decls    map[string]*ast.FuncDecl
	paths    map[string]bool
	methoded map[string]bool
	visited  map[string]bool
}

// declIndexFromPkgs 从已解析包建函数名索引(仅顶层函数,方法不参与 helper 求值)。
func declIndexFromPkgs(pkgs map[string]*ast.Package) map[string]*ast.FuncDecl {
	decls := map[string]*ast.FuncDecl{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, d := range f.Decls {
				if fn, ok := d.(*ast.FuncDecl); ok && fn.Name != nil && fn.Recv == nil {
					decls[fn.Name.Name] = fn
				}
			}
		}
	}
	return decls
}

// evalHelperRoutes 对每个 register 入口做 helper 递归求值,结果并入 paths/methoded。
// 入口的所有 *gin.RouterGroup 形参播种为空前缀(等价 routesFromDecl 的缺省 0 值)。
func evalHelperRoutes(decls map[string]*ast.FuncDecl, paths, methoded map[string]bool) {
	h := &helperEval{decls: decls, paths: paths, methoded: methoded, visited: map[string]bool{}}
	for name, fn := range decls {
		if !strings.HasPrefix(strings.ToLower(name), "register") {
			continue
		}
		gp := map[string]string{}
		for _, g := range groupParams(fn) {
			gp[g] = ""
		}
		if len(gp) == 0 {
			continue
		}
		h.eval(fn, evalCtx{groupPrefix: gp, strArgs: map[string]string{}})
	}
}

// eval 单函数求值入口:防环后交给 walk。
func (h *helperEval) eval(fn *ast.FuncDecl, ctx evalCtx) {
	key := fn.Name.Name + "\x00" + ctxKey(ctx)
	if h.visited[key] {
		return
	}
	h.visited[key] = true
	if fn.Body == nil {
		return
	}
	h.walk(fn.Body, ctx.groupPrefix, ctx.strArgs)
}

// walk 遍历语句:组前缀赋值 / 字面量或已绑定变量的 METHOD 调用 / helper 递归 /
// for-range 字符串字面量切片(循环变量逐值绑定后重走循环体——stripe.go
// /pay/stripe/{done,cancel} 即此形态,A/A2 曾双双漏检)。
func (h *helperEval) walk(body ast.Node, gp map[string]string, sa map[string]string) {
	ast.Inspect(body, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok && len(as.Lhs) == 1 {
			if id, ok := as.Lhs[0].(*ast.Ident); ok {
				if recv, pfx, ok := groupPrefix(as.Rhs); ok {
					gp[id.Name] = gp[recv] + pfx
				}
			}
			return true
		}
		if rs, ok := n.(*ast.RangeStmt); ok {
			if h.bindRangeLoop(rs, gp, sa) {
				return false // 循环体已按逐值绑定重走,不再按裸树遍历
			}
			return true
		}
		if ce, ok := callOf(n); ok {
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && httpMethods[sel.Sel.Name] && len(ce.Args) > 0 {
				if recv, ok := sel.X.(*ast.Ident); ok {
					h.recordMethodCall(gp, sa, sel.Sel.Name, recv.Name, ce.Args[0])
				}
				return true
			}
			if id, ok := ce.Fun.(*ast.Ident); ok {
				h.evalCall(id.Name, ce.Args, gp, sa)
			}
		}
		return true
	})
}

// bindRangeLoop 识别 `for _, p := range []string{"a","b"}`:逐字面量绑定循环变量,
// 重走循环体;非该形态(变量切片/表达式/匿名 `_`)返回 false 交回普通遍历(不猜)。
func (h *helperEval) bindRangeLoop(rs *ast.RangeStmt, gp map[string]string, sa map[string]string) bool {
	loopVar := ""
	if id, ok := rs.Value.(*ast.Ident); ok {
		loopVar = id.Name
	} else if id, ok := rs.Key.(*ast.Ident); ok {
		loopVar = id.Name
	}
	if loopVar == "" || loopVar == "_" {
		return false
	}
	lit, ok := rs.X.(*ast.CompositeLit)
	if !ok {
		return false
	}
	elems := make([]string, 0, len(lit.Elts))
	for _, e := range lit.Elts {
		if b, ok := e.(*ast.BasicLit); ok && b.Kind == token.STRING {
			elems = append(elems, unquote(b.Value))
		} else {
			return false
		}
	}
	for _, v := range elems {
		sub := make(map[string]string, len(sa)+1)
		for k, val := range sa {
			sub[k] = val
		}
		sub[loopVar] = v
		h.walk(rs.Body, gp, sub)
	}
	return true
}

// evalCall 递归被调 helper:首参组前缀已知才进入;字符串字面量按位置绑定形参。
func (h *helperEval) evalCall(name string, callArgs []ast.Expr, gp map[string]string, sa map[string]string) {
	fd, ok := h.decls[name]
	if !ok {
		return
	}
	g := firstGroupParam(fd)
	if g == "" {
		return
	}
	recvIdent, ok := callArgs[0].(*ast.Ident)
	if !ok {
		return
	}
	prefix, ok := gp[recvIdent.Name]
	if !ok {
		return
	}
	child := evalCtx{groupPrefix: map[string]string{g: prefix}, strArgs: map[string]string{}}
	for i, arg := range callArgs {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING && i < numParams(fd) {
			if pname := paramName(fd, i); pname != "" {
				child.strArgs[pname] = unquote(lit.Value)
			}
		}
	}
	h.eval(fd, child)
}

// recordMethodCall 路径实参:字面量直取;标识符查 strArgs 绑定,查不到放弃(helper 内变量拼接不猜);
// 前缀按接收者 ident 精确取(gp 可能同时存在多个组变量,任取必错)。
func (h *helperEval) recordMethodCall(gp map[string]string, sa map[string]string, method, recv string, pathArg ast.Expr) {
	var p string
	switch a := pathArg.(type) {
	case *ast.BasicLit:
		p = unquote(a.Value)
	case *ast.Ident:
		p = sa[a.Name]
	}
	if p == "" {
		return
	}
	prefix, ok := gp[recv]
	if !ok {
		return
	}
	full := normalizePath(prefix + p)
	h.paths[full] = true
	h.methoded[method+" "+full] = true
}

func ctxKey(ctx evalCtx) string {
	keys := make([]string, 0, len(ctx.groupPrefix)+len(ctx.strArgs))
	for k, v := range ctx.groupPrefix {
		keys = append(keys, k+"="+v)
	}
	for k, v := range ctx.strArgs {
		keys = append(keys, k+"="+v)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func callOf(n ast.Node) (*ast.CallExpr, bool) {
	if st, ok := n.(*ast.ExprStmt); ok {
		ce, ok := st.X.(*ast.CallExpr)
		return ce, ok
	}
	return nil, false
}

func firstGroupParam(fd *ast.FuncDecl) string {
	for _, g := range groupParams(fd) {
		return g
	}
	return ""
}

// groupParams 列出函数全部 *gin.RouterGroup 形参名(入口播种与 helper 绑定共用)。
func groupParams(fd *ast.FuncDecl) []string {
	out := []string{}
	if fd.Type.Params == nil {
		return out
	}
	for _, field := range fd.Type.Params.List {
		seg := field.Type
		if star, ok := seg.(*ast.StarExpr); ok {
			seg = star.X
		}
		sel, ok := seg.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "RouterGroup" {
			continue
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != "gin" {
			continue
		}
		for _, name := range field.Names {
			out = append(out, name.Name)
		}
	}
	return out
}

func numParams(fd *ast.FuncDecl) int {
	n := 0
	for _, f := range fd.Type.Params.List {
		n += len(f.Names)
		if len(f.Names) == 0 {
			n++
		}
	}
	return n
}

func paramName(fd *ast.FuncDecl, idx int) string {
	list := fd.Type.Params.List
	pos := 0
	for _, f := range list {
		for _, name := range f.Names {
			if pos == idx {
				return name.Name
			}
			pos++
		}
		if len(f.Names) == 0 {
			if pos == idx {
				return ""
			}
			pos++
		}
	}
	return ""
}
