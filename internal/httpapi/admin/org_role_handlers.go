// 自定义角色域 handler(migrations/000100):角色详情/权限码清单/派生角色改删。
// 模板复用 = 前端以 GET /role-details 的内置角色权限集为初始勾选,POST 全集落库。
package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// orgListPermissionsHandler GET /permissions:全量权限码清单(角色编辑抽屉数据源)。
func orgListPermissionsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.User.ListPermissions(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// orgListRoleDetailsHandler GET /role-details:角色详情全集(内置标记 + 权限码聚合)。
func orgListRoleDetailsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.User.ListRoleDetails(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// orgCreateRoleHandler POST /roles:新建派生角色(权限码全集)。
func orgCreateRoleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req roleReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Name, "name", 64),
			)
		}) {
			return
		}
		role, err := a.User.CreateCustomRole(c.Request.Context(), req.Name, req.PermissionCodes)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "权限变更", "role", role.Code, map[string]any{
			"name": req.Name, "perms": len(req.PermissionCodes), "op": "create",
		})
		respond(c, apitypes.CodeOK, role)
	}
}

// orgUpdateRoleHandler PUT /roles/{roleId}:编辑派生角色(改名 + 权限集全量替换);内置拒改。
func orgUpdateRoleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "roleId")
		if !ok {
			return
		}
		var req roleReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Name, "name", 64),
			)
		}) {
			return
		}
		if err := a.User.UpdateCustomRole(c.Request.Context(), id, req.Name, req.PermissionCodes); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "权限变更", "role", c.Param("roleId"), map[string]any{
			"name": req.Name, "perms": len(req.PermissionCodes), "op": "update",
		})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// orgDeleteRoleHandler DELETE /roles/{roleId}:删除派生角色;被账号/岗位引用时 40900 拒。
func orgDeleteRoleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "roleId")
		if !ok {
			return
		}
		if err := a.User.DeleteCustomRole(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "权限变更", "role", c.Param("roleId"), map[string]any{"op": "delete"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
