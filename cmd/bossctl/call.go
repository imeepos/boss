package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// callTarget 从位置参数取 METHOD 与补全后的 PATH。
// 不足两个位置参数报用法错(此前直接索引 positional[1] 会 panic)。
func callTarget(positional []string) (method, path string, err error) {
	if len(positional) < 2 {
		return "", "", fmt.Errorf("用法: bossctl call METHOD PATH [--data JSON|@file] [--query k=v]")
	}
	return strings.ToUpper(positional[0]), resolvePath(positional[1]), nil
}

// call 调用任意 API 端点: bossctl call METHOD PATH [--data JSON|@file] [--query k=v]
func (c *CLI) call(args []string) error {
	data, query, positional, err := queryFromArgs(args)
	if err != nil {
		return err
	}
	method, path, err := callTarget(positional)
	if err != nil {
		return err
	}

	resp, err := c.do(method, path, data, query)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return resp.errBiz("请求")
	}
	printJSON(resp.Data)
	return nil
}

// resolvePath 路径补全:完整路径原样;user:/x / worker:/x 换对应端前缀;裸路径补 admin 前缀。
func resolvePath(p string) string {
	for _, portal := range portalPrefixes {
		if strings.HasPrefix(p, portal.Name+":") {
			return portal.Prefix + strings.TrimPrefix(p, portal.Name+":")
		}
	}
	if !strings.HasPrefix(p, "/api/") {
		return "/api/admin/v1" + p
	}
	return p
}

// me 查看当前身份: bossctl me
// 服务端对 account/worker/customer 三类 API key 主体都返回各自身份视图。
func (c *CLI) me() error {
	resp, err := c.do("GET", "/api/admin/v1/auth/me", nil, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("未认证(code=%d msg=%s)。使用 --api-key 或 --jwt 指定认证信息,或运行 bossctl login", resp.Code, resp.Msg)
	}
	printJSON(resp.Data)
	return nil
}

// logout 退出登录:调用服务端注销并清除本地缓存 JWT。
func (c *CLI) logout() error {
	resp, err := c.do("POST", "/api/admin/v1/auth/logout", nil, nil)
	if err != nil {
		return err
	}
	clearToken()
	if resp.Code != 0 {
		// 服务端注销失败(如 token 已过期)不影响本地清理,提示即可
		fmt.Printf("本地 JWT 已清除(服务端注销返回 code=%d msg=%s,可能早已过期)\n", resp.Code, resp.Msg)
		return nil
	}
	fmt.Println("已退出登录,本地 JWT 已清除(~/.bossctl/token)")
	return nil
}

// login BOSS 账号登录: bossctl login USERNAME PASSWORD
// 登录成功后 JWT 自动保存到 ~/.bossctl/token。
func (c *CLI) login(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("用法: bossctl login USERNAME PASSWORD")
	}
	username, password := args[0], args[1]

	body := map[string]string{"username": username, "password": password}
	resp, err := c.do("POST", "/api/admin/v1/auth/login", body, nil)
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
