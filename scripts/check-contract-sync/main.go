// Command check-contract-sync 契约同步机械门禁。
// 三项检查,任一 fail 即退出码 1(CI/Makefile 接入点):
//
//	A. Go 实际注册路由必须在 api/openapi/{admin,user,worker}.yaml 之一出现(实现超前契约=漂移)。
//	B. json tag 必须 lowerCamelCase(fields.md §0 全局强制)。
//	C. 非 test 的 .go 文件不得超过 300 行(AGENTS.md 红线),存量超标走 baseline 豁免。
//
// 已知且有意接受的差异登记在 check-contract-sync.baseline(本目录),新增差异即 fail,
// 迫使每次漂移显式过账(改 baseline 或改契约)。
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

const repoRoot = "../.." // 相对本目录运行: go run ./scripts/check-contract-sync

// resolveRoot 兼容两种调用位置:脚本目录(../..)与仓库根(.)。
// 判据:候选目录下存在 internal/app 即视为仓库根。
func resolveRoot(flagVal string) string {
	for _, cand := range []string{flagVal, "."} {
		if _, err := os.Stat(filepath.Join(cand, "internal", "app")); err == nil {
			return cand
		}
	}
	return flagVal
}

func main() {
	root := flag.String("root", repoRoot, "repo root")
	flag.Parse()
	rootVal := resolveRoot(*root)
	fails := 0
	fails += checkRoutes(rootVal)
	fails += checkJSONTags(rootVal)
	fails += checkFileLen(rootVal)
	if fails > 0 {
		fmt.Printf("check-contract-sync: %d 项失败\n", fails)
		os.Exit(1)
	}
	fmt.Println("check-contract-sync: OK")
}

// ---------- A. 路由 <-> OpenAPI 对账 ----------

var httpMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true}

// extractRoutes 用 go/ast 提取三端路由(internal/httpapi/{admin,user,worker})。
// 识别两种模式: <id> := <recv>.Group("pfx") 与 <recv>.METHOD("/path")。
// 路由已从 internal/app 迁至 internal/httpapi(2026-08 三端拆分),此处跟随迁移;
// 按端分别归集,与 api/openapi/{admin,user,worker}.yaml 逐端对账(三端路径可重名)。
func extractRoutes(root string) (map[string]map[string]bool, error) {
	routes := map[string]map[string]bool{}
	fset := token.NewFileSet()
	for _, face := range []string{"admin", "user", "worker"} {
		dir := filepath.Join(root, "internal", "httpapi", face)
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			return nil, err
		}
		faceRoutes := map[string]bool{}
		for _, pkg := range pkgs {
			for _, f := range pkg.Files {
				for _, decl := range f.Decls {
					routesFromDecl(decl, faceRoutes)
				}
			}
		}
		routes[face] = faceRoutes
	}
	return routes, nil
}

func routesFromDecl(decl ast.Decl, routes map[string]bool) {
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
			if recv, path, ok := methodCall(st.X); ok {
				routes[normalizePath(prefix[recv]+path)] = true
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

// methodCall 识别 <recv>.METHOD("/path") 调用。
func methodCall(e ast.Expr) (recvID, path string, ok bool) {
	ce, ok := e.(*ast.CallExpr)
	if !ok {
		return "", "", false
	}
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok || !httpMethods[sel.Sel.Name] || len(ce.Args) == 0 {
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

func unquote(s string) string { return strings.Trim(s, "`\"") }

// normalizePath gin :param → openapi {param};剥离三端 /api/{face}/v1 前缀。
func normalizePath(p string) string {
	for _, pfx := range []string{"/api/admin/v1", "/api/user/v1", "/api/worker/v1", "/api/v1"} {
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
	routes, err := extractRoutes(root)
	if err != nil {
		fmt.Println("A: 解析路由失败:", err)
		return 1
	}
	base := loadBaseline()
	fails := 0
	total := 0
	for _, face := range []string{"admin", "user", "worker"} {
		specPaths := map[string]bool{}
		if err := collectSpecPaths(filepath.Join(root, "api/openapi", face+".yaml"), specPaths); err != nil {
			fmt.Println("A: 解析", face+".yaml", "失败:", err)
			return 1
		}
		var miss []string
		for r := range routes[face] {
			if !specPaths[r] {
				miss = append(miss, r)
			}
		}
		sort.Strings(miss)
		for _, p := range miss {
			if base["route:"+face+p] {
				continue
			}
			fmt.Println("A FAIL 路由已实现但契约未登记:", face, p)
			fails++
		}
		total += len(routes[face])
	}
	if fails == 0 {
		fmt.Printf("A OK 三端 %d 条路由全部有契约\n", total)
	}
	return fails
}

var specPathRe = regexp.MustCompile(`^  (/[^:\s]+):\s*\{?\s*\$ref`)

func collectSpecPaths(path string, out map[string]bool) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if m := specPathRe.FindStringSubmatch(line); m != nil {
			out[m[1]] = true
		}
	}
	return nil
}

// ---------- B. json tag 命名 ----------

var tagNameRe = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

func checkJSONTags(root string) int {
	fails := 0
	for _, dir := range []string{"internal", "pkg", "cmd"} {
		filepath.Walk(filepath.Join(root, dir), func(p string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			fails += tagsInFile(p)
			return nil
		})
	}
	if fails == 0 {
		fmt.Println("B OK json tag 全部 lowerCamelCase")
	}
	return fails
}

func tagsInFile(path string) int {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		fmt.Println("B: 解析失败", path, err)
		return 1
	}
	fails := 0
	ast.Inspect(f, func(n ast.Node) bool {
		st, ok := n.(*ast.StructType)
		if !ok {
			return true
		}
		for _, fld := range st.Fields.List {
			if fld.Tag == nil {
				continue
			}
			name := tagName(unquote(fld.Tag.Value))
			if name == "" || name == "-" {
				continue
			}
			if !tagNameRe.MatchString(name) {
				pos := fset.Position(fld.Tag.Pos())
				fmt.Printf("B FAIL %s:%d json:\"%s\" 非 lowerCamelCase\n", path, pos.Line, name)
				fails++
			}
		}
		return true
	})
	return fails
}

func tagName(tag string) string {
	for _, part := range strings.Fields(tag) {
		if v, ok := strings.CutPrefix(part, "json:"); ok {
			return strings.Split(unquote(v), ",")[0]
		}
	}
	return ""
}

// ---------- C. 文件行数红线(存量豁免走 baseline) ----------

func checkFileLen(root string) int {
	fails := 0
	base := loadBaseline()
	for _, dir := range []string{"internal", "pkg", "cmd"} {
		filepath.Walk(filepath.Join(root, dir), func(p string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			n := countLines(p)
			if n > 300 {
				rel, _ := filepath.Rel(root, p)
				if base["len:"+rel] {
					return nil
				}
				fmt.Printf("C FAIL %s %d 行,超 300 红线\n", rel, n)
				fails++
			}
			return nil
		})
	}
	if fails == 0 {
		fmt.Println("C OK 非测试文件均 <=300 行(豁免除外)")
	}
	return fails
}

func countLines(p string) int {
	b, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	return strings.Count(string(b), "\n") + 1
}

// ---------- baseline ----------

// baselinePath baseline 与本文件同目录(与运行 cwd 无关)。
func baselinePath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "check-contract-sync.baseline")
}

func loadBaseline() map[string]bool {
	m := map[string]bool{}
	if b, err := os.ReadFile(baselinePath()); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			l = strings.TrimSpace(l)
			if l != "" && !strings.HasPrefix(l, "#") {
				m[l] = true
			}
		}
	}
	return m
}
