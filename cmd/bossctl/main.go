// bossctl CLI 工具:操作全部 API 接口,支持免登录 API key 认证。
//
// 用法:
//
//	bossctl [--server URL] [--api-key KEY | --as 身份名] <command> [args]
//
// 全局 flag 必须放在子命令之前(flag 包在首个位置参数处停止解析)。
//
// 认证优先级:
//  1. --as <身份名>(身份档案 ~/.bossctl/identities.json,复合场景切换首选)
//  2. --api-key / BOSS_API_KEY 环境变量(免登录)
//  3. --jwt / BOSS_JWT 环境变量
//  4. 登录后缓存的 JWT(~/.bossctl/token)
//
// 服务端地址优先级: --server > BOSS_SERVER 环境变量 > 缺省 102 部署环境。
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// bossctlVersion 构建时由 ldflags -X 注入 git describe(未注入显示 dev,
// 便于识别"源码 go run"与"发布构建");报障请附带。
var bossctlVersion = "dev"

// defaultServer 缺省服务端:项目约定的 102 部署环境(不要本机起服务)。
const defaultServer = "http://192.168.0.102:28080"

// resolveServer 服务端地址优先级: --server 显式指定 > BOSS_SERVER 环境变量 > 缺省 102 部署环境。
func resolveServer(explicit string) string {
	if explicit != "" {
		return strings.TrimRight(explicit, "/")
	}
	if env := os.Getenv("BOSS_SERVER"); env != "" {
		return strings.TrimRight(env, "/")
	}
	return defaultServer
}

func main() {
	server := flag.String("server", "", "BOSS server base URL(缺省读 BOSS_SERVER 环境变量)")
	apiKey := flag.String("api-key", os.Getenv("BOSS_API_KEY"), "API key (免登录,优先级高于 JWT)")
	jwt := flag.String("jwt", os.Getenv("BOSS_JWT"), "JWT token (登录后获取)")
	as := flag.String("as", "", "按身份名切换 API key(见 identity 命令)")
	showVersion := flag.Bool("version", false, "打印版本号")
	flag.Parse()

	cfg := &config{
		Server: resolveServer(*server),
		APIKey: *apiKey,
		JWT:    *jwt,
	}

	cli := &CLI{cfg: cfg}

	if *showVersion {
		fmt.Printf("bossctl %s\n", bossctlVersion)
		return
	}

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
	case "logout":
		err = cli.logout()
	case "me":
		err = cli.me()
	case "routes":
		cli.routes(args)
	case "upload":
		err = cli.upload(args)
	case "release":
		err = cli.release(args)
	case "apikey":
		err = cli.apikey(args)
	case "identity":
		err = cli.identity(args)
	case "ai":
		err = cli.ai(args)
	case "version":
		fmt.Printf("bossctl %s\n", bossctlVersion)
	default:
		err = fmt.Errorf("未知命令: %s\n运行 bossctl -h 查看帮助", cmd)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		if hint := authHint(err, cli.identityName); hint != "" {
			fmt.Fprintf(os.Stderr, "提示: %s\n", hint)
		}
		os.Exit(1)
	}
}

// authHint 认证失败(401)时的修复提示:--as 身份提示档案重存,
// 否则提示登录途径;非 401 错误返回空串不打扰。
func authHint(err error, identityName string) string {
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		return ""
	}
	if identityName != "" {
		return fmt.Sprintf("身份 %q 档案里的 key 可能已吊销或主体被重建,重新保存: bossctl identity save %s --api-key NEW_KEY", identityName, identityName)
	}
	return "未认证。使用 --api-key / --jwt 指定认证信息,或运行 bossctl login 获取 JWT"
}

// printHelp 输出帮助文本。
func printHelp() {
	out := os.Stderr
	fmt.Fprintln(out, "bossctl — BOSS API CLI 工具")
	fmt.Fprintf(out, "版本 %s · 缺省服务端 %s(可用 BOSS_SERVER 覆盖)\n", bossctlVersion, defaultServer)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "用法:")
	fmt.Fprintln(out, "  bossctl [--server URL] [--api-key KEY | --as 身份名] <command> [args]")
	fmt.Fprintln(out, "  (全局 flag 必须放在子命令之前)")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "认证:")
	fmt.Fprintln(out, "  --as <身份名>               按身份档案切换 API key(复合场景首选)")
	fmt.Fprintln(out, "  --api-key / BOSS_API_KEY    免登录密钥")
	fmt.Fprintln(out, "  --jwt / BOSS_JWT            JWT 令牌(通过 login 获取)")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "命令:")
	fmt.Fprintln(out, "  call METHOD PATH [--data JSON|@file] [--query k=v]  调用任意 API")
	fmt.Fprintln(out, "       路径端前缀: user:/orders、worker:/home(缺省 admin 端)")
	fmt.Fprintln(out, "  login USERNAME PASSWORD                        登录获取 JWT")
	fmt.Fprintln(out, "  logout                                         退出登录并清除本地 JWT")
	fmt.Fprintln(out, "  me                                             查看当前身份")
	fmt.Fprintln(out, "  routes [admin|user|worker]                     列出 API 路由(三端)")
	fmt.Fprintln(out, "  upload [--portal admin|user|worker] FILE       附件上传(multipart,32MB)")
	fmt.Fprintln(out, "  apikey list                                    列出 API key")
	fmt.Fprintln(out, "  apikey create <account|worker|customer>/<id> <name>  为主体创建 API key")
	fmt.Fprintln(out, "  apikey revoke <id>                             吊销 API key")
	fmt.Fprintln(out, "  identity save <name> --api-key KEY             保存身份档案")
	fmt.Fprintln(out, "  identity list                                  列出身份档案")
	fmt.Fprintln(out, "  identity remove <name>                         删除身份档案")
	fmt.Fprintln(out, "  release upload --app user|worker --version 1.2.0 --code 12 FILE.apk")
	fmt.Fprintln(out, "  release patch <id> --status GRAY --rollout 20  发版状态/灰度编辑")
	fmt.Fprintln(out, "  release list [--app user|worker]               发版列表")
	fmt.Fprintln(out, "  ai config [--url URL] [--key KEY] [--model M]  查看/更新 OpenAI 集中配置")
	fmt.Fprintln(out, "  ai chat [--model M] <文本>                     AI 对话补全")
	fmt.Fprintln(out, "  ai embed [--model M] <文本...>                 文本向量化")
	fmt.Fprintln(out, "  version                                        打印版本号")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "退出码: 0 成功;1 失败(HTTP 错误或业务 code!=0),CI/脚本可据此判断")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "复合场景示例(customer 下单 → admin 派单 → worker 扫码 → admin 管理):")
	fmt.Fprintln(out, "  bossctl --as customer call POST user:/orders --data '{...}'")
	fmt.Fprintln(out, "  bossctl --as admin call POST /dispatch/pool/TK-1/assign --data '{\"masterId\":5}'")
	fmt.Fprintln(out, "  bossctl --as worker call POST worker:/tickets/TK-1/scan-bind --data '{\"epc\":\"EPC-1\"}'")
	fmt.Fprintln(out, "  bossctl --as admin call GET /orders/ORD-1")
}
