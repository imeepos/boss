// bossmcp 是 BOSS 平台的 MCP(Model Context Protocol)server:
// 经 stdio 把用户端(user)/师傅端(worker)/管理端(admin)REST 接口暴露给 agent。
// 认证按端绑定环境变量,三表主体 key 互不通用:
//
//	BOSS_USER_API_KEY    user 端(客户主体)
//	BOSS_WORKER_API_KEY  worker 端(师傅主体)
//	BOSS_ADMIN_API_KEY   admin 端(account 主体;admin 目录按该账号权限过滤,不参与兜底)
//	BOSS_API_KEY         user/worker 两端兜底(仅在端专属 key 缺省时使用)
//	BOSS_SERVER          后端地址(缺省 http://192.168.0.102:28080)
//
// 用法(任一 MCP 客户端配置):
//
//	{"command": "<repo>/bossmcp", "env": {"BOSS_USER_API_KEY": "boss_...", "BOSS_WORKER_API_KEY": "boss_..."}}
//
// 详见 docs/mcp.md。
package main

import (
	"fmt"
	"os"

	"github.com/ymm-001/boss/internal/apiclient"
	"github.com/ymm-001/boss/internal/apiroutes"
)

// bossmcpVersion 构建期经 -ldflags -X 注入。
var bossmcpVersion = "dev"

// portalCfg 单端配置:认证 key 与实际生效的环境变量名(诊断提示用)。
type portalCfg struct {
	Name    string
	Prefix  string
	Client  *apiclient.Client
	EnvKeys []string
}

// requireKey 未配置 key 时拒绝调用,给出配置指引而非放行 401。
func (p *portalCfg) requireKey() error {
	if p.Client.APIKey == "" {
		return fmt.Errorf("%s 端未配置 API key:启动前设置环境变量 %s(签发方式: bossctl apikey create,详见 docs/mcp.md)", p.Name, p.EnvKeys[0])
	}
	return nil
}

// server 聚合三端配置。
type server struct {
	Version string
	Portals map[string]*portalCfg
}

// portal 按名取端配置。
func (s *server) portal(name string) (*portalCfg, error) {
	p, ok := s.Portals[name]
	if !ok {
		return nil, fmt.Errorf("未知端 %q(可用 user/worker/admin)", name)
	}
	return p, nil
}

// newServer 从 getenv 装配三端客户端(注入便于测试)。
func newServer(getenv func(string) string) *server {
	s := &server{Version: bossmcpVersion, Portals: map[string]*portalCfg{}}
	for _, dir := range apiroutes.Portals {
		key, envName := resolveKey(getenv, dir.Name)
		s.Portals[dir.Name] = &portalCfg{
			Name:    dir.Name,
			Prefix:  dir.Prefix,
			Client:  apiclient.New(getenv("BOSS_SERVER"), key),
			EnvKeys: []string{envName},
		}
	}
	return s
}

// resolveKey 端专属 key 优先,BOSS_API_KEY 兜底(user/worker);admin 端仅认 BOSS_ADMIN_API_KEY
// (admin key 是账号级权限面,不参与跨端兜底,防误把业务 key 提权到管理面);
// 返回实际采用(或应配置)的环境变量名。
func resolveKey(getenv func(string) string, portal string) (key, envName string) {
	specific := map[string]string{
		"user":   "BOSS_USER_API_KEY",
		"worker": "BOSS_WORKER_API_KEY",
		"admin":  "BOSS_ADMIN_API_KEY",
	}[portal]
	if v := getenv(specific); v != "" {
		return v, specific
	}
	if portal != "admin" {
		if v := getenv("BOSS_API_KEY"); v != "" {
			return v, "BOSS_API_KEY"
		}
	}
	return "", specific
}

// main 启动 stdio 协议循环;日志只写 stderr(stdout 是协议通道)。
func main() {
	s := newServer(os.Getenv)
	logf("version=%s server=%s user_key=%s worker_key=%s admin_key=%s",
		s.Version,
		s.Portals["user"].Client.Server,
		keyState(s.Portals["user"]), keyState(s.Portals["worker"]), keyState(s.Portals["admin"]))
	if err := serve(os.Stdin, os.Stdout, s); err != nil {
		logf("serve FAILED: %v", err)
		os.Exit(1)
	}
}

// keyState 诊断串:key 来源环境变量或 MISSING。
func keyState(p *portalCfg) string {
	if p.Client.APIKey == "" {
		return "MISSING(" + p.EnvKeys[0] + ")"
	}
	return "set(" + p.EnvKeys[0] + ")"
}

// logf stderr 日志,带 [bossmcp] 前缀便于 grep。
func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[bossmcp] "+format+"\n", args...)
}
