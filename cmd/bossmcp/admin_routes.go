package main

// admin 端目录按账号权限过滤(硬性过关线:目录面 = 当前登录账号拥有的接口面)。
// 数据链:服务端 requirePerm 装配 → genrouteperms 静态投影(apiroutes.AdminRoutePerms)
// → /auth/me 取账号权限码全集(apiroutes.Profile) → 子集过滤;与 Authz 语义严格同源。

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ymm-001/boss/internal/apiclient"
	"github.com/ymm-001/boss/internal/apiroutes"
)

// adminProfile /auth/me 账号主体载荷(目录过滤只需权限码与身份标识)。
type adminProfile struct {
	AccountID       int64    `json:"accountId"`
	Username        string   `json:"username"`
	RealName        string   `json:"realName"`
	RoleCode        string   `json:"roleCode"`
	TemplateCode    string   `json:"templateCode"`
	PermissionCodes []string `json:"permissionCodes"`
}

// fetchAdminProfile 用 admin key 查 /auth/me,取账号身份与权限码全集。
// worker/customer 主体误配时 AccountID=0 且无权限码:目录仅剩认证自服务/公开面(与服务端 403 一致)。
func fetchAdminProfile(p *portalCfg) (*adminProfile, error) {
	res, err := p.Client.Call("GET", p.Prefix+whoamiPath[p.Name], nil, nil)
	if err != nil {
		return nil, fmt.Errorf("权限查询失败(GET /auth/me): %v", err)
	}
	if res.Envelope == nil || res.Envelope.Code != 0 {
		return nil, fmt.Errorf("无法确认账号权限(GET /auth/me HTTP %d): %s;admin 目录拒发,请检查 BOSS_ADMIN_API_KEY",
			res.StatusCode, apiclient.Truncate(res.Body))
	}
	var prof adminProfile
	if err := json.Unmarshal(res.Envelope.Data, &prof); err != nil {
		return nil, fmt.Errorf("/auth/me 载荷解析失败: %v", err)
	}
	return &prof, nil
}

// adminRoutes admin 目录:按当前账号权限过滤后再暴露;拒绝在权限查询失败时放行全量目录。
func (s *server) adminRoutes(p *portalCfg, filter string) (string, error) {
	if err := p.requireKey(); err != nil {
		return "", err
	}
	prof, err := fetchAdminProfile(p)
	if err != nil {
		return "", err
	}
	perms := map[string]bool{}
	for _, c := range prof.PermissionCodes {
		perms[c] = true
	}
	dir := apiroutes.ByName(p.Name)
	var b strings.Builder
	fmt.Fprintf(&b, "[admin 端 · 前缀 %s · 认证 %s]\n", p.Prefix, p.EnvKeys[0])
	fmt.Fprintf(&b, "账号=%s(%s, role=%s", prof.Username, prof.RealName, prof.RoleCode)
	if prof.TemplateCode != "" {
		fmt.Fprintf(&b, ", 受限模板=%s", prof.TemplateCode)
	}
	fmt.Fprintf(&b, ") 持有权限码 %d 个;目录已按账号权限过滤,权限外接口调用将被服务端 403 拒绝\n",
		len(prof.PermissionCodes))
	kept := apiroutes.FilterByPerms(dir, dir.Routes, perms)
	n := 0
	for _, r := range kept {
		if filter != "" && !routeMatch(r, filter) {
			continue
		}
		fmt.Fprintf(&b, "%-6s %s    %s\n", r.Method, r.Path, r.Desc)
		n++
	}
	fmt.Fprintf(&b, "共 %d/%d 条(账号可用 %d 条);调用接口: boss_call(portal=\"admin\", method, path)。",
		n, len(dir.Routes), len(kept))
	return b.String(), nil
}
