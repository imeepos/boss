// Command check-contract-sync 契约同步机械门禁。
// 三项检查,任一 fail 即退出码 1(CI/Makefile 接入点):
//
//	A. 路由对账在 routes.go;B. json tag 在 jsontags.go;C. 行数红线在 filelen.go。
//	已登记差异走 check-contract-sync.baseline 豁免,新增差异即 fail。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

func unquote(s string) string { return strings.Trim(s, "`\"") }
