// 检查 C:非 test 的 .go 文件不得超过 300 行(AGENTS.md 红线),存量超标走 baseline 豁免。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

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

// baselinePath baseline 与本目录源文件同目录(与运行 cwd 无关)。
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
