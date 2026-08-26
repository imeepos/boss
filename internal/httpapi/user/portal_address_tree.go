package userapi

// 用户端地址层级树端点:级联子节点 + 全树搜索。
// 复用 admin 同源 domain(user.Service.ListAddresses/SearchAddresses),
// 仅登录门禁(uauth + portalCustomerOnly),无 RBAC;契约 misc.yaml /address-tree。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPortalAddressTreeRoutes /address-tree 系列:children + search。
// 由 registerPortalAddressRoutes 统一挂载(uauth 组内,JWT 客户鉴权)。
func registerPortalAddressTreeRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/address-tree", portalAddressChildren(a))
	g.GET("/address-tree/search", portalAddressTreeSearch(a))
}

// portalAddressChildren GET /address-tree?parentId= 子节点列表;parentId 缺省 0=顶层。
// App 级联选择器逐层懒加载;返回域模型 Address(json tag 即契约字段)。
func portalAddressChildren(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.User == nil {
			respond(c, apitypes.CodeInternal, gin.H{"error": "user service not configured"})
			return
		}
		parentID, err := strconv.ParseInt(c.DefaultQuery("parentId", "0"), 10, 64)
		if err != nil || parentID < 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		items, err := a.User.ListAddresses(c.Request.Context(), parentID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// portalAddressTreeSearch GET /address-tree/search?q= 全树关键字搜索(命中+祖先链)。
// 语义与 admin /addresses/search 一致;App 小区/街道层搜索直达,免逐级翻页。
func portalAddressTreeSearch(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.User == nil {
			respond(c, apitypes.CodeInternal, gin.H{"error": "user service not configured"})
			return
		}
		q := c.Query("q")
		if q == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		hits, err := a.User.SearchAddresses(c.Request.Context(), q)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": hits})
	}
}
