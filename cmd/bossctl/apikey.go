package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// apikey 管理 API key: apikey list|create|revoke
// create 支持三类主体: account/<id> | worker/<id> | customer/<id>
func (c *CLI) apikey(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: bossctl apikey list|create <account|worker|customer>/<id> <name>|revoke <id>")
	}
	switch args[0] {
	case "list":
		return c.apiKeyList()
	case "create":
		if len(args) < 3 {
			return fmt.Errorf("用法: bossctl apikey create <account|worker|customer>/<id> <name>\n示例: bossctl apikey create worker/5 field-test")
		}
		return c.apiKeyCreate(args[1], args[2])
	case "revoke":
		if len(args) < 2 {
			return fmt.Errorf("用法: bossctl apikey revoke <id>")
		}
		return c.apiKeyRevoke(args[1])
	default:
		return fmt.Errorf("未知 API key 命令: %s", args[0])
	}
}

// parseSubject 解析 account/7 形式的主体标识。
func parseSubject(s string) (string, int64, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("主体格式应为 <account|worker|customer>/<id>,如 worker/5,收到 %q", s)
	}
	var ref int64
	if _, err := fmt.Sscanf(parts[1], "%d", &ref); err != nil || ref <= 0 {
		return "", 0, fmt.Errorf("主体 ID 非法: %q", parts[1])
	}
	switch parts[0] {
	case "account", "worker", "customer":
		return parts[0], ref, nil
	}
	return "", 0, fmt.Errorf("主体类型非法: %q(应为 account|worker|customer)", parts[0])
}

// apiKeyList 列出所有 API key。
func (c *CLI) apiKeyList() error {
	resp, err := c.do("GET", "/api/admin/v1/api-keys", nil, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("获取 API key 列表失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	printJSON(resp.Data)
	return nil
}

// apiKeyCreate 为指定主体创建 API key,返回完整密钥(仅此一次)。
func (c *CLI) apiKeyCreate(subject, name string) error {
	subjType, ref, err := parseSubject(subject)
	if err != nil {
		return err
	}
	body := map[string]any{"subjectType": subjType, "subjectRef": ref, "name": name}
	resp, err := c.do("POST", "/api/admin/v1/api-keys", body, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("创建 API key 失败: code=%d msg=%s", resp.Code, resp.Msg)
	}

	var data struct {
		ID         int64  `json:"id"`
		PlainKey   string `json:"plainKey"`
		SubjectRef int64  `json:"subjectRef"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("解析创建响应: %w", err)
	}
	fmt.Printf("创建成功(%s 主体 #%d),请立即保存密钥(仅此一次返回):\n", subjType, data.SubjectRef)
	fmt.Printf("  %s\n", data.PlainKey)
	fmt.Printf("保存为身份: bossctl identity save %s-test --api-key %s\n", subjType, data.PlainKey)
	fmt.Printf("使用身份:   bossctl --as %s-test me\n", subjType)
	return nil
}

// apiKeyRevoke 吊销(停用)API key。
func (c *CLI) apiKeyRevoke(id string) error {
	resp, err := c.do("DELETE", "/api/admin/v1/api-keys/"+id, nil, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("吊销 API key 失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	fmt.Printf("API key %s 已吊销\n", id)
	return nil
}
