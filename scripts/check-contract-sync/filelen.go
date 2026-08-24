// 检查 C:非 test 源文件不得超过 300 行(AGENTS.md 红线),存量超标走 baseline 豁免。
// Go 侧覆盖 internal/pkg/cmd;Kotlin 侧覆盖 mobile/*/android/app/src/main(手写页面红线)。
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
	fails += walkLen(root, []string{"internal", "pkg", "cmd"}, ".go", func(p string) bool {
		return strings.HasSuffix(p, "_test.go") || isGenerated(p)
	})
	for _, d := range []string{"mobile/worker/android", "mobile/user/android"} {
		fails += walkLen(root, []string{filepath.Join(d, "app/src/main")}, ".kt", func(string) bool { return false })
	}
	if fails == 0 {
		fmt.Println("C OK 非测试文件均 <=300 行(豁免除外)")
	}
	return fails
}

func walkLen(root string, dirs []string, ext string, skip func(string) bool) int {
	fails := 0
	base := loadBaseline()
	for _, dir := range dirs {
		filepath.Walk(filepath.Join(root, dir), func(p string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() || !strings.HasSuffix(p, ext) || skip(p) {
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
	return fails
}

func countLines(p string) int {
	b, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	return strings.Count(string(b), "\n") + 1
}

// isGenerated 按 Go 惯例识别生成文件:头部含 "Code generated"。
func isGenerated(p string) bool {
	b, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	head := string(b)
	if i := strings.IndexByte(head, '\n'); i >= 0 {
		head = head[:min(i, 512)]
	}
	return strings.Contains(head, "Code generated")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
