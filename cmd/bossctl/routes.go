package main

import (
	"fmt"
	"os"
)

// routeEntry 路由目录项。
type routeEntry struct {
	Method string
	Path   string
	Desc   string
}

// routeCatalog 全部路由目录合集,仅用于总量校验(单端目录见 portalPrefixes;
// 由 api/openapi 生成,再生成: node scripts/gen-bossctl-routes.mjs)。
var routeCatalog = append(append(append([]routeEntry{}, adminRoutes...), userRoutes...), workerRoutes...)

// portalPrefixes 各端 API 前缀(call 路径补全与目录展示共用)。
var portalPrefixes = []struct {
	Name   string
	Prefix string
	Routes []routeEntry
}{
	{"admin", "/api/admin/v1", adminRoutes},
	{"user", "/api/user/v1", userRoutes},
	{"worker", "/api/worker/v1", workerRoutes},
}

// routes 列出三端 API 路由目录: bossctl routes [admin|user|worker]
func (c *CLI) routes(args []string) {
	if len(args) > 0 {
		for _, p := range portalPrefixes {
			if p.Name == args[0] {
				printPortalRoutes(p.Name, p.Prefix, p.Routes)
				return
			}
		}
		fmt.Fprintf(os.Stderr, "未知端: %s(可用 admin/user/worker)\n", args[0])
		os.Exit(1)
	}
	for _, p := range portalPrefixes {
		printPortalRoutes(p.Name, p.Prefix, p.Routes)
	}
	fmt.Println("用法: bossctl call METHOD PATH [--data JSON|@file] [--query k=v]")
	fmt.Println("路径可带端前缀: user:/orders / worker:/home(缺省 admin);bossctl upload FILE 同理")
}

// printPortalRoutes 打印单个端的路由目录。
func printPortalRoutes(name, prefix string, routes []routeEntry) {
	fmt.Printf("\n[%s 端 · 前缀 %s · %d 条]\n", name, prefix, len(routes))
	for _, r := range routes {
		fmt.Printf("  %-6s %s    %s\n", r.Method, r.Path, r.Desc)
	}
}
