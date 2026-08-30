// 检查 A:Go 实际注册路由必须在 api/openapi/{admin,user,worker}.yaml 之一出现
// (实现超前契约 = 漂移)。用 go/ast 提取三端路由,逐端与契约对账。
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var httpMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true}

// routeSets 一次 AST 提取的两种视图:paths 仅路径(A 检查用),methoded 带
// 方法前缀 "GET /orders"(A2 方法级对账用);key 均经 normalizePath 归一。
type routeSets struct {
	paths    map[string]map[string]bool
	methoded map[string]map[string]bool
}

// extractRoutes 识别两种模式: <id> := <recv>.Group("pfx") 与 <recv>.METHOD("/path")。
// 路由已从 internal/app 迁至 internal/httpapi(2026-08 三端拆分),此处跟随迁移;
// 按端分别归集,与 api/openapi/{admin,user,worker}.yaml 逐端对账(三端路径可重名)。
func extractRoutes(root string) (routeSets, error) {
	rs := routeSets{paths: map[string]map[string]bool{}, methoded: map[string]map[string]bool{}}
	fset := token.NewFileSet()
	for _, face := range []string{"admin", "user", "worker", "open"} {
		dir := filepath.Join(root, "internal", "httpapi", face)
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			return rs, err
		}
		facePaths := map[string]bool{}
		faceMethoded := map[string]bool{}
		for _, pkg := range pkgs {
			for _, f := range pkg.Files {
				for _, decl := range f.Decls {
					routesFromDecl(decl, facePaths, faceMethoded)
				}
			}
		}
		rs.paths[face] = facePaths
		rs.methoded[face] = faceMethoded
	}
	return rs, nil
}

func routesFromDecl(decl ast.Decl, paths, methoded map[string]bool) {
	fn, ok := decl.(*ast.FuncDecl)
	if !ok || !strings.HasPrefix(strings.ToLower(fn.Name.Name), "register") {
		return
	}
	prefix := map[string]string{"g": ""} // ident -> 路径前缀;寄存器形参 g 前缀 ""
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok && len(as.Lhs) == 1 {
			id, ok := as.Lhs[0].(*ast.Ident)
			if !ok {
				return true
			}
			recv, pfx, ok := groupPrefix(as.Rhs)
			if !ok {
				return true
			}
			prefix[id.Name] = prefix[recv] + pfx
			return true
		}
		if st, ok := n.(*ast.ExprStmt); ok {
			if recv, method, path, ok := methodCall(st.X); ok {
				p := normalizePath(prefix[recv] + path)
				paths[p] = true
				methoded[method+" "+p] = true
			}
		}
		return true
	})
}

// groupPrefix 识别 <recv>.Group("pfx", ...) 调用,返回 (recv ident, recv前缀+pfx)。
func groupPrefix(exprs []ast.Expr) (recvID, pfx string, ok bool) {
	ce, ok := exprs[0].(*ast.CallExpr)
	if !ok {
		return "", "", false
	}
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Group" || len(ce.Args) == 0 {
		return "", "", false
	}
	lit, ok := ce.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", "", false
	}
	id, _ := sel.X.(*ast.Ident)
	if id == nil {
		return "", "", false
	}
	return id.Name, unquote(lit.Value), true
}

// methodCall 识别 <recv>.METHOD("/path") 调用,返回 recv、HTTP 方法与路径。
func methodCall(e ast.Expr) (recvID, method, path string, ok bool) {
	ce, ok := e.(*ast.CallExpr)
	if !ok {
		return "", "", "", false
	}
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok || !httpMethods[sel.Sel.Name] || len(ce.Args) == 0 {
		return "", "", "", false
	}
	lit, ok := ce.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", "", "", false
	}
	id, _ := sel.X.(*ast.Ident)
	if id == nil {
		return "", "", "", false
	}
	return id.Name, sel.Sel.Name, unquote(lit.Value), true
}

// normalizePath gin :param → openapi {param};剥离三端 /api/{face}/v1 前缀。
func normalizePath(p string) string {
	for _, pfx := range []string{"/api/admin/v1", "/api/user/v1", "/api/worker/v1", "/api/open/v1", "/api/v1"} {
		p = strings.TrimPrefix(p, pfx)
	}
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, ":") {
			segs[i] = "{" + s[1:] + "}"
		}
	}
	return strings.Join(segs, "/")
}

func checkRoutes(root string) int {
	rs, err := extractRoutes(root)
	if err != nil {
		fmt.Println("A: 解析路由失败:", err)
		return 1
	}
	base := loadBaseline()
	fails := 0
	total := 0
	for _, face := range []string{"admin", "user", "worker", "open"} {
		fails += checkFaceRoutes(root, face, rs.paths[face], base)
		total += len(rs.paths[face])
	}
	if fails == 0 {
		fmt.Printf("A OK 三端 %d 条路由全部有契约\n", total)
	}
	fails += checkA2(root, rs, base)
	return fails
}

func checkFaceRoutes(root, face string, faceRoutes map[string]bool, base map[string]bool) int {
	specPaths := map[string]bool{}
	if err := collectSpecPaths(root, face, specPaths); err != nil {
		fmt.Println("A: 解析", face+"/*", "失败:", err)
		return 1
	}
	var miss []string
	for r := range faceRoutes {
		if !specPaths[r] {
			miss = append(miss, r)
		}
	}
	sort.Strings(miss)
	fails := 0
	for _, p := range miss {
		if base["route:"+face+p] {
			continue
		}
		fmt.Println("A FAIL 路由已实现但契约未登记:", face, p)
		fails++
	}
	return fails
}

// specPathLineRe 匹配 OpenAPI path 项行:`  /foo: ...`(两空格缩进的 path key)。
// 兼容两种场景:
//   - 顶层 {face}.yaml 里形如 `  /foo: { $ref: '...' }` 的 $ref 转发;
//   - {face}/*.yaml 子文件里形如 `  /foo:` 后接 `    get:` / `    post:` 等方法块;
//
// 均由同一正则捕获,故不再限定末尾必须是 $ref。
//
// 排除 YAML 锚点(&foo:)、更深缩进的 operationId/summary 等子项。
var specPathLineRe = regexp.MustCompile(`^  (/[^:\s]+):\s*(\{|$)`)

// collectSpecPaths 扫描 api/openapi/<face>.yaml 与 api/openapi/<face>/ 下的子文件,
// 仅收集当前 face 的契约,防 admin/user 文件互相污染;任何形如 `  /path:` 的 path 项
// 都登记到 out(同源多文件重复视为同一路径)。
func collectSpecPaths(root, face string, out map[string]bool) error {
	base := filepath.Join(root, "api/openapi")
	if err := scanSpecFile(filepath.Join(base, face+".yaml"), out); err != nil {
		return err
	}
	dir := filepath.Join(base, face)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		if err := scanSpecFile(filepath.Join(dir, e.Name()), out); err != nil {
			return err
		}
	}
	return nil
}

// scanSpecFile 单文件扫描:提取所有 `  /path:` 行。
func scanSpecFile(path string, out map[string]bool) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if m := specPathLineRe.FindStringSubmatch(line); m != nil {
			out[m[1]] = true
		}
	}
	return nil
}
