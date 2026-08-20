// bossctl CLI 工具:操作全部 API 接口,支持免登录 API key 认证。
//
// 用法:
//
//	bossctl [--server URL] [--api-key KEY | --as 身份名] <command> [args]
//
// 认证优先级:
//  1. --as <身份名>(身份档案 ~/.bossctl/identities.json,复合场景切换首选)
//  2. --api-key / BOSS_API_KEY 环境变量(免登录)
//  3. --jwt / BOSS_JWT 环境变量
//  4. 登录后缓存的 JWT(~/.bossctl/token)
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
	as := flag.String("as", "", "按身份名切换 API key(见 identity 命令)")
	flag.Parse()

	cfg := &config{
		Server: strings.TrimRight(*server, "/"),
		APIKey: *apiKey,
		JWT:    *jwt,
	}

	cli := &CLI{cfg: cfg}

	if flag.NArg() == 0 {
		printHelp()
		os.Exit(0)
	}

	// --as 身份切换:覆盖 --api-key / env(身份档案优先级最高)
	if *as != "" {
		if err := cli.applyIdentity(*as); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
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
		cli.routes(args)
	case "upload":
		err = cli.upload(args)
	case "apikey":
		err = cli.apikey(args)
	case "identity":
		err = cli.identity(args)
	case "ai":
		err = cli.ai(args)
	default:
		err = fmt.Errorf("未知命令: %s\n运行 bossctl -h 查看帮助", cmd)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

// printHelp 输出帮助文本。
func printHelp() {
	fmt.Fprintln(os.Stderr, "bossctl — BOSS API CLI 工具")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "用法:")
	fmt.Fprintln(os.Stderr, "  bossctl [--server URL] [--api-key KEY | --as 身份名] <command> [args]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "认证:")
	fmt.Fprintln(os.Stderr, "  --as <身份名>               按身份档案切换 API key(复合场景首选)")
	fmt.Fprintln(os.Stderr, "  --api-key / BOSS_API_KEY    免登录密钥")
	fmt.Fprintln(os.Stderr, "  --jwt / BOSS_JWT            JWT 令牌(通过 login 获取)")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "命令:")
	fmt.Fprintln(os.Stderr, "  call METHOD PATH [--data JSON] [--query k=v]  调用任意 API")
	fmt.Fprintln(os.Stderr, "       路径端前缀: user:/orders、worker:/home(缺省 admin 端)")
	fmt.Fprintln(os.Stderr, "  login USERNAME PASSWORD                        登录获取 JWT")
	fmt.Fprintln(os.Stderr, "  me                                                查看当前身份")
	fmt.Fprintln(os.Stderr, "  routes [admin|user|worker]                        列出 API 路由(三端)")
	fmt.Fprintln(os.Stderr, "  upload [--portal admin|user|worker] FILE       附件上传(multipart,32MB)")
	fmt.Fprintln(os.Stderr, "  apikey list                                      列出 API key")
	fmt.Fprintln(os.Stderr, "  apikey create <account|worker|customer>/<id> <name>  为主体创建 API key")
	fmt.Fprintln(os.Stderr, "  apikey revoke <id>                               吊销 API key")
	fmt.Fprintln(os.Stderr, "  identity save <name> --api-key KEY              保存身份档案")
	fmt.Fprintln(os.Stderr, "  identity list                                   列出身份档案")
	fmt.Fprintln(os.Stderr, "  identity remove <name>                          删除身份档案")
	fmt.Fprintln(os.Stderr, "  ai config [--url URL] [--key KEY] [--model M]   查看/更新 OpenAI 集中配置")
	fmt.Fprintln(os.Stderr, "  ai chat [--model M] <文本>                       AI 对话补全")
	fmt.Fprintln(os.Stderr, "  ai embed [--model M] <文本...>                   文本向量化")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "复合场景示例(customer 下单 → admin 派单 → worker 扫码 → admin 管理):")
	fmt.Fprintln(os.Stderr, "  bossctl --as customer call POST /orders --data '{...}'")
	fmt.Fprintln(os.Stderr, "  bossctl --as admin call POST /dispatch/pool/TK-1/assign --data '{\"masterId\":5}'")
	fmt.Fprintln(os.Stderr, "  bossctl --as worker call POST /tickets/TK-1/scan-bind --data '{\"epc\":\"EPC-1\"}'")
	fmt.Fprintln(os.Stderr, "  bossctl --as admin call GET /orders/ORD-1")
}
