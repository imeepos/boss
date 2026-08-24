package main

// 检查 E:web/admin menu.def 每个页面 key 必须有对应 menu:<key> 权限码登记在
// 某个迁移里(先例:000039 geo_menu / 000135 cms_menu);漏登 = 102 回放 403 才发现
// (2026-08-28 menu:site 踩坑)。复用他人权限码的存量页面走 baseline 豁免登记。

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	menuKeyRe  = regexp.MustCompile(`key:\s*'([a-z0-9-]+)'`)
	menuPermRe = regexp.MustCompile(`menu:([a-z0-9-]+)`)
)

func checkMenuPerms(root string) int {
	defPath := filepath.Join(root, "web", "admin", "src", "router", "menu.def.ts")
	defSrc, err := os.ReadFile(defPath)
	if err != nil {
		fmt.Println("E FAIL 无法读取 menu.def.ts:", err)
		return 1
	}
	defKeys := map[string]bool{}
	for _, m := range menuKeyRe.FindAllStringSubmatch(string(defSrc), -1) {
		defKeys[m[1]] = true
	}

	permCodes := map[string]bool{}
	migDir := filepath.Join(root, "migrations")
	entries, _ := os.ReadDir(migDir)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(migDir, e.Name()))
		if err != nil {
			continue
		}
		for _, m := range menuPermRe.FindAllStringSubmatch(string(b), -1) {
			permCodes[m[1]] = true
		}
	}

	baseline := loadBaseline()
	var missing []string
	for k := range defKeys {
		if permCodes[k] {
			continue
		}
		if baseline["menu:"+k] {
			continue
		}
		missing = append(missing, k)
	}
	if len(missing) == 0 {
		fmt.Println("E OK menu.def 页面 key 权限码全部登记(或 baseline 豁免)")
		return 0
	}
	for _, k := range missing {
		fmt.Printf("E FAIL menu.def key %q 无 menu:%s 权限码迁移(也未 baseline 豁免):新建迁移登记 permissions+role_permissions,或复用他人权限码则进 baseline 登记理由\n", k, k)
	}
	return 1
}
