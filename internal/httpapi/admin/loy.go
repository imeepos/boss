package adminapi

// 积分域管理端点:余额/流水查询 + 手动调整(000104 最小 LOY)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerLoyRoutes 注册积分域管理路由(权限挂 menu:userdata 组,与促销域同门)。
func registerLoyRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:userdata")
	g.GET("/points/:customerId", perm, loyGetPoints(a))
	g.POST("/points/:customerId/adjust", perm, loyAdjustPoints(a))
}

// loyGetPoints GET /points/:customerId:客户积分余额与流水。
func loyGetPoints(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, ok := pathIDValid(c, "customerId")
		if !ok {
			return
		}
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

// loyAdjustPoints POST /points/:customerId/adjust {delta,reason}:手动调整。
func loyAdjustPoints(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, ok := pathIDValid(c, "customerId")
		if !ok {
			return
		}
		var req struct {
			Delta  int64  `json:"delta" binding:"required,ne=0"`
			Reason string `json:"reason"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if req.Reason == "" {
			req.Reason = "ADMIN_ADJUST"
		}
		bal, err := a.Points.Adjust(c.Request.Context(), cid, req.Delta, req.Reason)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"balance": bal})
	}
}
