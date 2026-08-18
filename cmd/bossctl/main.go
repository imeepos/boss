// bossctl CLI 工具:操作全部 API 接口,支持免登录 API key 认证。
//
// 用法:
//   bossctl [--server URL] [--api-key KEY] <command> [args]
//
// 资源:
//   docs/skill/bossctl.md   技能使用说明
//
// 认证优先级:
//   1. --api-key / BOSS_API_KEY 环境变量(免登录)
//   2. --jwt / BOSS_JWT 环境变量
//   3. 登录后缓存的 JWT(~/.bossctl/token)
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	server := flag.String("server", "http://localhost:8080", "BOSS server base URL")
	apiKey := flag.String("api-key", os.Getenv("BOSS_API_KEY"), "API key (免登录,优先级高于 JWT)")
	jwt := flag.String("jwt", os.Getenv("BOSS_JWT"), "JWT token (登录后获取)")
	flag.Parse()

	cfg := &config{
		Server: strings.TrimRight(*server, "/"),
		APIKey: *apiKey,
		JWT:    *jwt,
	}

	cli := &CLI{cfg: cfg}

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "bossctl — BOSS API CLI 工具")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "用法:")
		fmt.Fprintln(os.Stderr, "  bossctl [--server URL] [--api-key KEY] [--jwt TOKEN] <command> [args]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "认证:")
		fmt.Fprintln(os.Stderr, "  --api-key / BOSS_API_KEY    免登录(推荐,优先级最高)")
		fmt.Fprintln(os.Stderr, "  --jwt / BOSS_JWT            JWT 令牌(通过 login 获取)")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "命令:")
		fmt.Fprintln(os.Stderr, "  call METHOD PATH [--data JSON] [--query k=v]  调用任意 API")
		fmt.Fprintln(os.Stderr, "  login USERNAME PASSWORD                        登录获取 JWT")
		fmt.Fprintln(os.Stderr, "  me                                                查看当前身份")
		fmt.Fprintln(os.Stderr, "  routes                                            列出 API 路由")
		fmt.Fprintln(os.Stderr, "  apikey list                                      列出 API key")
		fmt.Fprintln(os.Stderr, "  apikey create <accountId> <name>                 创建 API key")
		fmt.Fprintln(os.Stderr, "  apikey revoke <id>                               吊销 API key")
		os.Exit(0)
	}

	cmd := flag.Arg(0)
	args := flag.Args()[1:]

	var err error
	switch cmd {
	case "call":
		err = cli.call(args)
	case "login":
		err = cli.login(args)
	case "me":
		err = cli.me()
	case "routes":
		cli.routes()
	case "apikey":
		err = cli.apikey(args)
	default:
		err = fmt.Errorf("未知命令: %s\n运行 bossctl -h 查看帮助", cmd)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}