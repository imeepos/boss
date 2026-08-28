package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// identity 身份档案:复合业务场景测试时快速切换主体。
// 档案存于 ~/.bossctl/identities.json,每条记录一个主体 + 其 API key。
//
// 典型场景(customer 下单 → account 派单 → worker 上门 → account 管理):
//
//	bossctl identity save customer --api-key boss_c1...
//	bossctl identity save admin    --api-key boss_a1...
//	bossctl identity save worker   --api-key boss_w1...
//	bossctl --as customer call POST /orders --data '{...}'
//	bossctl --as admin    call POST /dispatch/pool/TK-1/assign --data '{"masterId":5}'
//	bossctl --as worker   call POST /tickets/TK-1/scan-bind --data '{"epc":"EPC-1"}'
//	bossctl --as admin    call GET /orders/ORD-1

// identityFile 返回身份档案文件路径。
func identityFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".bossctl", "identities.json")
}

// loadIdentities 读取全部身份档案;文件不存在返回空表。
func loadIdentities() (map[string]string, error) {
	b, err := os.ReadFile(identityFile())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("解析身份档案: %w", err)
	}
	return out, nil
}

// saveIdentities 写回身份档案(0600,含密钥)。
func saveIdentities(ids map[string]string) error {
	b, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(identityFile())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(identityFile(), b, 0600)
}

// identity 身份档案管理: save|list|remove
func (c *CLI) identity(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: bossctl identity save <name> --api-key KEY | list | remove <name>")
	}
	ids, err := loadIdentities()
	if err != nil {
		return err
	}
	switch args[0] {
	case "save":
		name, key := identitySaveArgs(args[1:])
		if name == "" || key == "" {
			return fmt.Errorf("用法: bossctl identity save <name> --api-key KEY")
		}
		if err := checkAPIKey(name, key); err != nil {
			return err
		}
		ids[name] = key
		if err := saveIdentities(ids); err != nil {
			return err
		}
		fmt.Printf("身份 %q 已保存(当前共 %d 个身份,用 --as %s 切换)\n", name, len(ids), name)
		return nil
	case "list":
		if len(ids) == 0 {
			fmt.Println("暂无身份档案。创建: bossctl identity save <name> --api-key KEY")
			return nil
		}
		names := make([]string, 0, len(ids))
		for n := range ids {
			names = append(names, n)
		}
		sort.Strings(names)
		fmt.Printf("已保存 %d 个身份(切换: bossctl --as <name> ...):\n", len(ids))
		for _, n := range names {
			k := ids[n]
			prefix := k
			if len(k) > 12 {
				prefix = k[:12] + "..."
			}
			fmt.Printf("  %-16s %s\n", n, prefix)
		}
		return nil
	case "remove":
		if len(args) < 2 {
			return fmt.Errorf("用法: bossctl identity remove <name>")
		}
		if _, ok := ids[args[1]]; !ok {
			return fmt.Errorf("身份 %q 不存在", args[1])
		}
		delete(ids, args[1])
		if err := saveIdentities(ids); err != nil {
			return err
		}
		fmt.Printf("身份 %q 已删除\n", args[1])
		return nil
	}
	return fmt.Errorf("未知 identity 子命令: %s", args[0])
}

// applyIdentity 按 --as 名称加载身份密钥(优先级高于 --api-key/env)。
// 记录 identityName,401 时主程序给出档案过期的修复提示。
func (c *CLI) applyIdentity(name string) error {
	ids, err := loadIdentities()
	if err != nil {
		return err
	}
	key, ok := ids[name]
	if !ok {
		return fmt.Errorf("身份 %q 未保存;先执行: bossctl identity save %s --api-key KEY", name, name)
	}
	if err := checkAPIKey(name, key); err != nil {
		return err
	}
	c.cfg.APIKey = key
	c.identityName = name
	return nil
}

// checkAPIKey 校验档案里的 key 形态(boss_ + 小写十六进制);历史档案曾被粘贴进
// 换行/中文尾巴导致 "invalid header field value",加载时尽早报清楚而不是在 HTTP 层炸。
func checkAPIKey(name, key string) error {
	if !apiKeyPattern.MatchString(key) {
		return fmt.Errorf("身份 %q 的 key 形态非法(应为 boss_+十六进制,疑似粘贴混入多余字符);重新保存: bossctl identity save %s --api-key KEY", name, name)
	}
	return nil
}

var apiKeyPattern = regexp.MustCompile(`^boss_[0-9a-f]+$`)

// identitySaveArgs 从 identity save 子命令参数中取 <name> 与 --api-key 值。
// (全局 flag 解析在首个位置参数处停止,子命令级 flag 需手工解析)
func identitySaveArgs(args []string) (name, key string) {
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--api-key" && i+1 < len(args):
			key = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--api-key="):
			key = strings.TrimPrefix(args[i], "--api-key=")
		default:
			if name == "" {
				name = args[i]
			}
		}
	}
	return name, key
}
