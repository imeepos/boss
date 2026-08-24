package adminapi

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerSitePublicRoutes 官网匿名只读端点(免登录,仅 PUBLISHED)。
// 沿用 partner 入驻公开提交先例:公开面收窄到最小投影,不透出版本/作者等管理字段。
func registerSitePublicRoutes(api *gin.RouterGroup, a *app.Application) {
	api.GET("/site/posts", sitePublicListHandler(a))
	api.GET("/site/posts/:slug", sitePublicDetailHandler(a))
}

// registerSiteRoutes 管理端文章 CRUD(menu:site 权限)。
func registerSiteRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/site-posts", requirePerm(a.User, "menu:site"), siteListHandler(a))
	g.POST("/site-posts", requirePerm(a.User, "menu:site"), siteCreateHandler(a))
	g.PUT("/site-posts/:postId", requirePerm(a.User, "menu:site"), siteUpdateHandler(a))
	g.DELETE("/site-posts/:postId", requirePerm(a.User, "menu:site"), siteDeleteHandler(a))
}

func siteLimit(c *gin.Context) int {
	n, _ := strconv.Atoi(c.Query("limit"))
	if n <= 0 {
		n = 10
	}
	return n
}
