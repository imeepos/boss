package main

import (
	"fmt"
	"strings"
)

// call 调用任意 API 端点: bossctl call METHOD PATH [--data JSON] [--query k=v]
func (c *CLI) call(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("用法: bossctl call METHOD PATH [--data JSON] [--query k=v]")
	}

	data, query, positional := queryFromArgs(args)
	method := strings.ToUpper(positional[0])
	path := positional[1]

	// 自动补全 /api/admin/v1 前缀
	if !strings.HasPrefix(path, "/api/") {
		path = "/api/admin/v1" + path
	}

	resp, err := c.do(method, path, data, query)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		fmt.Printf("请求失败: code=%d msg=%s\n", resp.Code, resp.Msg)
		return nil
	}
	printJSON(resp.Data)
	return nil
}

// me 查看当前身份: bossctl me
func (c *CLI) me() error {
	resp, err := c.do("GET", "/api/admin/v1/auth/me", nil, nil)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		fmt.Println("未认证。使用 --api-key 或 --jwt 指定认证信息,或运行 bossctl login")
		return nil
	}
	printJSON(resp.Data)
	return nil
}
