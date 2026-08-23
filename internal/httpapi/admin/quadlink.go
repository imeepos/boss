package adminapi

// 四码合一域路由注册(承接 quad.yaml,任一码反查)。
// 全部 handler 实现见 quadlink_handlers.go;此处只保留扁平路由表 + 共享工具。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerQuadlinkRoutes 注册四码合一域路由(承接 quad.yaml,任一码反查)。
func registerQuadlinkRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/quad-links", requirePerm(a.User, "menu:quadlink"), quadlinkCreateLinkHandler(a))
	g.GET("/quad-links", requirePerm(a.User, "menu:quadlink"), quadlinkListLinksHandler(a))
	g.GET("/quad-links/by-asset", requirePerm(a.User, "menu:quadlink"), quadlinkGetByAssetHandler(a))
	g.GET("/quad-links/by-customer", requirePerm(a.User, "menu:quadlink"), quadlinkGetByCustomerHandler(a))
	g.GET("/quad-links/by-port", requirePerm(a.User, "menu:quadlink"), quadlinkGetByPortHandler(a))
	g.GET("/quad-links/by-address", requirePerm(a.User, "menu:quadlink"), quadlinkGetByAddressHandler(a))
}

// requireID 校验必填 ID 查询参数:缺失/非法/非正数时返回 422 并终止请求。
func requireID(c *gin.Context, name string) (int64, bool) {
	v := queryInt64(c, name)
	if v <= 0 {
		respond(c, apitypes.CodeInvalidParam, nil)
		return 0, false
	}
	return v, true
}
