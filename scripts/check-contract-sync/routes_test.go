// 契约对账 A 项门禁的回归测试:确保正则能同时识别顶层 $ref 转发与子文件直定义
// 两种 path 形态,且不被 method/tags/schemas 误伤。修路由 specPathRe 后回归必备。
package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile 测试辅助:覆盖 os.WriteFile 的导入,简化测试体。
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestScanSpecFileMatchesTopLevelAndSub(t *testing.T) {
	root := "../../" // 仓库根相对 scripts/check-contract-sync
	abs, _ := filepath.Abs(root)
	for _, face := range []string{"admin", "user", "worker", "open"} {
		got := map[string]bool{}
		if err := collectSpecPaths(abs, face, got); err != nil {
			t.Fatalf("%s: collect err: %v", face, err)
		}
		// 各 face 至少应有一条 path;具体条数随契约演化,不锁死总量。
		if len(got) == 0 {
			t.Fatalf("%s: 未扫描到任何 path", face)
		}
		// 顶层 admin.yaml 的 $ref 转发路径也应可见:
		if face == "admin" {
			if !got["/auth/login"] {
				t.Fatalf("admin: 漏扫顶层 $ref 转发的 /auth/login")
			}
			if !got["/ops/notify-emit"] {
				t.Fatalf("admin: 漏扫新增 /ops/notify-emit")
			}
		} else if got["/ops/notify-emit"] {
			t.Fatalf("%s: 不应混入 admin /ops/notify-emit", face)
		}
	}
}

func TestScanSpecFileIgnoresMethodLines(t *testing.T) {
	tmp := t.TempDir()
	// 构造一份最小 yaml:顶层 components + 一个 path + 一个 method 块;
	// 仅 path 项应进 specPaths,method/tags 不进。
	path := filepath.Join(tmp, "test.yaml")
	yaml := "openapi: 3.0.0\n" +
		"info:\n  title: t\n" +
		"paths:\n" +
		"  /v1/things:\n" +
		"    get:\n" +
		"      tags: [Foo]\n" +
		"      summary: list\n" +
		"  /v1/things/{id}:\n" +
		"    delete:\n" +
		"      summary: del\n"
	if err := writeFile(path, yaml); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	if err := scanSpecFile(path, got); err != nil {
		t.Fatal(err)
	}
	if !got["/v1/things"] || !got["/v1/things/{id}"] {
		t.Fatalf("missing paths, got=%v", got)
	}
	for k := range got {
		if k == "/v1/things" || k == "/v1/things/{id}" {
			continue
		}
		t.Fatalf("non-path captured: %q", k)
	}
}
