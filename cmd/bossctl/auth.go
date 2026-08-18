package main

import (
	"encoding/json"
	"fmt"
)

// login BOSS 账号登录: bossctl login USERNAME PASSWORD
// 登录成功后 JWT 自动保存到 ~/.bossctl/token。
func (c *CLI) login(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("用法: bossctl login USERNAME PASSWORD")
	}
	username, password := args[0], args[1]

	body := map[string]string{"username": username, "password": password}
	resp, err := c.do("POST", "/api/v1/auth/login", body, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("登录失败: code=%d msg=%s", resp.Code, resp.Msg)
	}

	var data struct {
		Token     string `json:"token"`
		AccountID int64  `json:"accountId"`
		RealName  string `json:"realName"`
		RoleName  string `json:"roleName"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return fmt.Errorf("解析登录响应: %w", err)
	}

	if err := saveToken(data.Token); err != nil {
		return fmt.Errorf("保存 token: %w", err)
	}
	// 自动设置 JWT 以便后续命令使用
	c.cfg.JWT = data.Token

	fmt.Printf("登录成功: accountId=%d realName=%s role=%s\n", data.AccountID, data.RealName, data.RoleName)
	fmt.Println("JWT 已保存到 ~/.bossctl/token")
	return nil
}
