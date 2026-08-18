package app

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// createAPIKeyReq 创建 API key 请求体。
type createAPIKeyReq struct {
	AccountID int64  `json:"accountId" binding:"required"`
	Name      string `json:"name" binding:"required"`
}

// registerAPIKeyRoutes 注册 API key 管理路由(需 sysadmin 权限)。
func registerAPIKeyRoutes(g *gin.RouterGroup, a *Application) {
	// 使用 requirePerm 确保调用方有 menu:apikey 权限(初始仅 sysadmin 角色持有)。
	ak := g.Group("", requirePerm(a.User, "menu:apikey"))

	ak.GET("/api-keys", func(c *gin.Context) {
		list, err := a.APIKey.List(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	ak.POST("/api-keys", func(c *gin.Context) {
		var req createAPIKeyReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		// 验证目标账号存在
		_, err := a.User.GetProfile(c.Request.Context(), req.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		res, err := a.APIKey.Create(c.Request.Context(), req.AccountID, claims.AccountID, req.Name)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"id":        res.ID,
			"accountId": res.AccountID,
			"name":      res.Name,
			"plainKey":  res.PlainKey,
		})
	})

	ak.DELETE("/api-keys/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.APIKey.Revoke(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	})
}