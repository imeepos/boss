package adminapi

// W 组织域路由注册(W1 模板:Authn→Authz→域查询→统一 envelope)。
// 全部 handler 实现见 org_handlers.go;此处只保留扁平路由表 + 请求体类型 + 共享工具。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// legalEntityReq 法人新建/编辑请求体(code/name 必填)。
type legalEntityReq struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// departmentReq 部门新建/编辑请求体(挂靠子公司,名称必填)。
type departmentReq struct {
	LegalEntityID int64  `json:"legalEntityId" binding:"required"`
	Name          string `json:"name" binding:"required"`
}

// postReq 岗位新建/编辑请求体(部门内 code 唯一;roles 为功能角色码,可空)。
type postReq struct {
	DeptID int64    `json:"deptId" binding:"required"`
	Code   string   `json:"code" binding:"required"`
	Name   string   `json:"name" binding:"required"`
	Roles  []string `json:"roles"`
}

// roleReq 自定义角色新建/编辑请求体(名称必填;permissionCodes 为权限码全集,空=清空)。
type roleReq struct {
	Name            string   `json:"name" binding:"required"`
	PermissionCodes []string `json:"permissionCodes"`
}

// requirePerm 返回 RBAC 中间件:账号需持有 permCode 才可访问。
// 列表端点按「菜单可见性」权限码门禁(menu:*),与 000003 种子对齐。
func requirePerm(svc user.Service, permCode string) gin.HandlerFunc {
	return middleware.Authz(svc.HasPermission, permCode)
}

// queryInt64 解析整型 query 参数;缺省/非法返回 0(域层按 0=全部 处理)。
func queryInt64(c *gin.Context, name string) int64 {
	v, _ := strconv.ParseInt(c.Query(name), 10, 64)
	return v
}

// registerOrgRoutes 注册组织域只读路由(W1 模板:Authn→Authz→域查询→统一 envelope)。
func registerOrgRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/legal-entities", requirePerm(a.User, "menu:company"), orgListLegalEntitiesHandler(a))
	g.POST("/legal-entities", requirePerm(a.User, "menu:company"), orgCreateLegalEntityHandler(a))
	g.PUT("/legal-entities/:legalEntityId", requirePerm(a.User, "menu:company"), orgUpdateLegalEntityHandler(a))

	g.GET("/accounts", requirePerm(a.User, "menu:account"), orgListAccountsHandler(a))
	g.POST("/accounts", requirePerm(a.User, "menu:account"), orgCreateAccountHandler(a))
	g.PUT("/accounts/:accountId", requirePerm(a.User, "menu:account"), orgUpdateAccountHandler(a))

	g.GET("/roles", requirePerm(a.User, "menu:account"), orgListRolesHandler(a))
	g.GET("/permissions", requirePerm(a.User, "menu:menuperm"), orgListPermissionsHandler(a))
	g.GET("/role-details", requirePerm(a.User, "menu:menuperm"), orgListRoleDetailsHandler(a))
	g.POST("/roles", requirePerm(a.User, "menu:menuperm"), orgCreateRoleHandler(a))
	g.PUT("/roles/:roleId", requirePerm(a.User, "menu:menuperm"), orgUpdateRoleHandler(a))
	g.DELETE("/roles/:roleId", requirePerm(a.User, "menu:menuperm"), orgDeleteRoleHandler(a))
	g.GET("/menu-perms", requirePerm(a.User, "menu:menuperm"), orgListMenuPermsHandler(a))
	g.GET("/data-scopes", requirePerm(a.User, "menu:datascope"), orgListDataScopesHandler(a))

	g.GET("/departments", requirePerm(a.User, "menu:department"), orgListDepartmentsHandler(a))
	g.POST("/departments", requirePerm(a.User, "menu:department"), orgCreateDepartmentHandler(a))
	g.PUT("/departments/:deptId", requirePerm(a.User, "menu:department"), orgUpdateDepartmentHandler(a))
	g.DELETE("/departments/:deptId", requirePerm(a.User, "menu:department"), orgDeleteDepartmentHandler(a))

	g.GET("/posts", requirePerm(a.User, "menu:post"), orgListPostsHandler(a))
	g.POST("/posts", requirePerm(a.User, "menu:post"), orgCreatePostHandler(a))
	g.PUT("/posts/:postId", requirePerm(a.User, "menu:post"), orgUpdatePostHandler(a))
	g.DELETE("/posts/:postId", requirePerm(a.User, "menu:post"), orgDeletePostHandler(a))
}
