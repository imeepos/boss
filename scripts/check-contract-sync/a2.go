// 检查 A2:方法级契约对账——契约登记了但未实现(整路径或仅缺方法),以及
// 实现方法与契约方法错位。是检查 A(路径级,实现→契约单向)之外的反向补全:
// /push/device 曾以 GET 探测误报 404(实为 POST),路径级对账查不出这类漂移。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// specMethodLineRe 匹配 path 块内的方法行:恰好 4 空格缩进的 get/post/put/delete/patch。
// 6 空格以上为 operationId/summary 等子项,不误捕。
var specMethodLineRe = regexp.MustCompile(`^    (get|post|put|delete|patch):`)

// collectSpecRoutes 扫描 api/openapi/<face>.yaml 与 <face>/ 子文件,产出 "GET /path" 集合;
// 与 collectSpecPaths 同文件域,防跨端污染。
func collectSpecRoutes(root, face string, out map[string]bool) error {
	base := filepath.Join(root, "api/openapi")
	if err := scanSpecRoutesFile(filepath.Join(base, face+".yaml"), out); err != nil {
		return err
	}
	dir := filepath.Join(base, face)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		if err := scanSpecRoutesFile(filepath.Join(dir, e.Name()), out); err != nil {
			return err
		}
	}
	return nil
}

// scanSpecRoutesFile 单文件扫描:path 行(2 空格)切换当前 path,其后 4 空格方法行归入该 path。
func scanSpecRoutesFile(path string, out map[string]bool) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cur := ""
	for _, line := range strings.Split(string(b), "\n") {
		if m := specPathLineRe.FindStringSubmatch(line); m != nil {
			cur = m[1]
			continue
		}
		if cur == "" {
			continue
		}
		if m := specMethodLineRe.FindStringSubmatch(line); m != nil {
			out[strings.ToUpper(m[1])+" "+cur] = true
		}
	}
	return nil
}

// checkA2 逐端对账:契约方法集 vs 实现 "METHOD path" 集;存量差异走 baseline 豁免。
func checkA2(root string, rs routeSets, base map[string]bool) int {
	fails, exempt := 0, 0
	for _, face := range []string{"admin", "user", "worker", "open"} {
		spec := map[string]bool{}
		if err := collectSpecRoutes(root, face, spec); err != nil {
			fmt.Println("A2: 解析", face, "契约失败:", err)
			fails++
			continue
		}
		f, e := a2Core(face, rs.paths[face], rs.methoded[face], spec, base)
		fails += f
		exempt += e
	}
	if fails == 0 {
		fmt.Printf("A2 OK 方法级对账无漂移(baseline 豁免存量 %d 条)\n", exempt)
	}
	return fails
}

// a2Core 纯函数判定:spec 有而实现无 → 整路径缺失或方法错位,各按 baseline 前缀豁免;
// 返回 (失败数, 豁免数)——豁免不静默,留可见计数防豁免无限累积。
func a2Core(face string, implPaths, implMethods, specMethods, base map[string]bool) (int, int) {
	var stale, mismatch []string
	exempt := 0
	for key := range specMethods {
		if implMethods[key] {
			continue
		}
		path := key[strings.Index(key, " ")+1:]
		if !implPaths[path] {
			if base["a2:spec:"+face+" "+key] {
				exempt++
				continue
			}
			stale = append(stale, key)
			continue
		}
		if base["a2:method:"+face+" "+key] {
			exempt++
			continue
		}
		mismatch = append(mismatch, key)
	}
	sort.Strings(stale)
	sort.Strings(mismatch)
	for _, k := range stale {
		fmt.Println("A2 FAIL 契约登记但未实现:", face, k)
	}
	for _, k := range mismatch {
		fmt.Println("A2 FAIL 方法错位: 契约", face, k, "实现方法集:", implMethodsFor(implMethods, k[strings.Index(k, " ")+1:]))
	}
	return len(stale) + len(mismatch), exempt
}

// implMethodsFor 列出实现侧某路径的全部方法,供错位提示定位。
func implMethodsFor(implMethods map[string]bool, path string) string {
	var ms []string
	for _, m := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		if implMethods[m+" "+path] {
			ms = append(ms, m)
		}
	}
	if len(ms) == 0 {
		return "[]"
	}
	return "[" + strings.Join(ms, " ") + "]"
}
