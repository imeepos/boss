package userapi

// 用户端门户积分域:余额/流水/积分换券(000104 最小 LOY)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPortalLoyRoutes 注册积分域路由。
func registerPortalLoyRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/points", portalPoints(a))
	g.POST("/points/exchange", portalPointsExchange(a))
}

// portalPoints GET /points:我的积分(余额 + 近 100 条流水)。
func portalPoints(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		bal, err := a.Points.Balance(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		entries, err := a.Points.Entries(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"balance": bal, "entries": entries})
	}
}

// portalPointsExchange POST /points/exchange {templateId}:积分换券。
func portalPointsExchange(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			TemplateID int64 `json:"templateId" binding:"required,gt=0"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		couponID, cost, err := a.Points.Exchange(c.Request.Context(), cid, req.TemplateID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"couponId": couponID, "cost": cost})
	}
}
