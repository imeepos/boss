package apiroutes

// admin 端目录按账号权限过滤的手写辅助(生成文件 routes_gen.go / admin_perms_gen.go 勿改)。
// 过滤语义与服务端 requirePerm(Authz)严格一致:路由须持有其全部权限码;无码路由仅认证即可。

// RouteKey 权限映射键:方法 空格 服务端全路径({param} 占位)。
func RouteKey(portal *Portal, r Route) string {
	return r.Method + " " + portal.Prefix + r.Path
}

// RoutePerms 路由的权限码全集;user/worker 端恒空(无菜单门禁)。
func RoutePerms(portal *Portal, r Route) []string {
	if portal.Name != "admin" {
		return nil
	}
	return AdminRoutePerms[RouteKey(portal, r)]
}

// AllowedByPerms 判定路由对持给定权限码集的账号可用。
// 空码集路由(认证自服务/公开)恒可用;admin 之外不受限。
func AllowedByPerms(portal *Portal, r Route, perms map[string]bool) bool {
	for _, code := range RoutePerms(portal, r) {
		if !perms[code] {
			return false
		}
	}
	return true
}

// FilterByPerms 过滤目录,保留该账号可用的路由(保持原顺序)。
func FilterByPerms(portal *Portal, routes []Route, perms map[string]bool) []Route {
	out := make([]Route, 0, len(routes))
	for _, r := range routes {
		if AllowedByPerms(portal, r, perms) {
			out = append(out, r)
		}
	}
	return out
}
