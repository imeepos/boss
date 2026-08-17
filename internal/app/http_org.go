package app

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

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
func registerOrgRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/legal-entities", requirePerm(a.User, "menu:company"), func(c *gin.Context) {
		list, err := a.User.ListLegalEntities(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	g.GET("/departments", requirePerm(a.User, "menu:department"), func(c *gin.Context) {
		list, err := a.User.ListDepartments(c.Request.Context(), queryInt64(c, "legalEntityId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	g.GET("/posts", requirePerm(a.User, "menu:post"), func(c *gin.Context) {
		list, err := a.User.ListPosts(c.Request.Context(), queryInt64(c, "deptId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	g.GET("/regions", requirePerm(a.User, "menu:region"), func(c *gin.Context) {
		list, err := a.User.ListRegions(c.Request.Context(), c.Query("parentPath"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})
}
