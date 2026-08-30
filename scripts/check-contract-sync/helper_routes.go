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
func evalHelperRoutes(decls map[string]*ast.FuncDecl, paths, methoded map[string]bool) {
	h := &helperEval{decls: decls, paths: paths, methoded: methoded, visited: map[string]bool{}}
	for name, fn := range decls {
		if !strings.HasPrefix(strings.ToLower(name), "register") {
			continue
		}
		if g := firstGroupParam(fn); g != "" {
			h.eval(fn, evalCtx{groupPrefix: map[string]string{g: ""}, strArgs: map[string]string{}})
		}
	}
}

// eval 单函数求值:字面量 METHOD/组前缀赋值与 routesFromDecl 同规则;
// 命中 helper 调用(首参组前缀已知)则递归。
func (h *helperEval) eval(fn *ast.FuncDecl, ctx evalCtx) {
	key := fn.Name.Name + "\x00" + ctxKey(ctx)
	if h.visited[key] {
		return
	}
	h.visited[key] = true
	if fn.Body == nil {
		return
	}
	gp := ctx.groupPrefix
	sa := ctx.strArgs
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok && len(as.Lhs) == 1 {
			if id, ok := as.Lhs[0].(*ast.Ident); ok {
				if recv, pfx, ok := groupPrefix(as.Rhs); ok {
					gp[id.Name] = gp[recv] + pfx
				}
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
	if fd.Type.Params == nil || numParams(fd) == 0 {
		return ""
	}
	seg := fd.Type.Params.List[0].Type
	if star, ok := seg.(*ast.StarExpr); ok {
		seg = star.X
	}
	if sel, ok := seg.(*ast.SelectorExpr); ok && sel.Sel.Name == "RouterGroup" {
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "gin" && len(fd.Type.Params.List[0].Names) > 0 {
			return fd.Type.Params.List[0].Names[0].Name
		}
	}
	return ""
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
