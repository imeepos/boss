package main

import (
	"encoding/json"
	"fmt"
)

// apikey 管理 API key: apikey list|create|revoke
func (c *CLI) apikey(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: bossctl apikey list|create <accountId> <name>|revoke <id>")
	}
	switch args[0] {
	case "list":
		return c.apiKeyList()
	case "create":
		if len(args) < 3 {
			return fmt.Errorf("用法: bossctl apikey create <accountId> <name>")
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

// apiKeyList 列出所有 API key。
func (c *CLI) apiKeyList() error {
	resp, err := c.do("GET", "/api/v1/api-keys", nil, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("获取 API key 列表失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	printJSON(resp.Data)
	return nil
}

// apiKeyCreate 创建 API key,返回完整密钥(仅此一次)。
func (c *CLI) apiKeyCreate(accountID, name string) error {
	body := map[string]any{"accountId": accountID, "name": name}
	resp, err := c.do("POST", "/api/v1/api-keys", body, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("创建 API key 失败: code=%d msg=%s", resp.Code, resp.Msg)
	}

	var data struct {
		PlainKey string `json:"plainKey"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("解析创建响应: %w", err)
	}
	fmt.Println("创建成功,请立即保存密钥(仅此一次返回):")
	fmt.Printf("  %s\n", data.PlainKey)
	fmt.Println("使用: bossctl --api-key " + data.PlainKey + " me")
	return nil
}

// apiKeyRevoke 吊销(停用)API key。
func (c *CLI) apiKeyRevoke(id string) error {
	resp, err := c.do("DELETE", "/api/v1/api-keys/"+id, nil, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("吊销 API key 失败: code=%d msg=%s", resp.Code, resp.Msg)
	}
	fmt.Printf("API key %s 已吊销\n", id)
	return nil
}