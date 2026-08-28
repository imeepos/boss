package userapi

// 用户端地址层级树端点:级联子节点 + 全树搜索。
// 复用 admin 同源 domain(user.Service.ListAddresses/SearchAddresses),
// 仅登录门禁(uauth + portalCustomerOnly),无 RBAC;契约 misc.yaml /address-tree。

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// maxLookupPaths 单次批量反查 path 上限(地址簿分页场景一页足够)。
const maxLookupPaths = 50

// registerPortalAddressTreeRoutes /address-tree 系列:children + search + lookup。
// 由 registerPortalAddressRoutes 统一挂载(uauth 组内,JWT 客户鉴权)。
func registerPortalAddressTreeRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/address-tree", portalAddressChildren(a))
	g.GET("/address-tree/search", portalAddressTreeSearch(a))
	g.GET("/address-tree/lookup", portalAddressLookup(a))
}

// portalAddressLookup GET /address-tree/lookup?paths=a.b.c,x.y.z 按路径批量精确反查。
// 地址簿列表/编辑表单反显面包屑用;缺失路径返回 missing 不整体 404
// (address_path 弱引用,树节点可删,悬挂路径须逐条暴露而非拖垮整页)。
func portalAddressLookup(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.User == nil {
			respond(c, apitypes.CodeInternal, gin.H{"error": "user service not configured"})
			return
		}
		paths := parseLookupPaths(c.Query("paths"))
		if len(paths) == 0 || len(paths) > maxLookupPaths {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		hits, missing, err := a.User.LookupAddresses(c.Request.Context(), paths)
		if err != nil {
			log.Printf("[portal-address-tree] LOOKUP FAILED paths=%v: %v", paths, err)
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(hits))
		for _, h := range hits {
			items = append(items, gin.H{"path": h.Node.Path, "node": h.Node, "ancestors": h.Ancestors})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items, "missing": missing})
	}
}

// parseLookupPaths 逗号分隔 path 清单:去空白、去空段、保序去重。
func parseLookupPaths(raw string) []string {
	out := make([]string, 0, strings.Count(raw, ",")+1)
	seen := map[string]bool{}
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
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
// hasMore=true 表示超 20 条被截断,客户端应提示收紧关键字(菲律宾 San/Santa 前缀常见)。
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
		hits, hasMore, err := a.User.SearchAddresses(c.Request.Context(), q)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": hits, "hasMore": hasMore})
	}
}
