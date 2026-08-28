package userapi

// 用户端门户积分域:余额/流水/积分换券(000104 最小 LOY)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPortalLoyRoutes 注册积分域路由。
func registerPortalLoyRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/points", portalPoints(a))
	g.GET("/points/exchange-offers", portalPointsExchangeOffers(a))
	g.POST("/points/exchange", portalPointsExchange(a))
	g.GET("/points/tier", portalPointsTier(a))
	g.GET("/points/tasks", portalPointsTasks(a))
	g.POST("/points/tasks/:id/complete", portalPointsTaskComplete(a))
}

// portalPointsExchangeOffers GET /points/exchange-offers:积分可兑换券模板列表
// (promotion 域 ENABLED 且 points_price>0;契约 loy.yaml)。
func portalPointsExchangeOffers(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.Promotion == nil {
			respond(c, apitypes.CodeOK, gin.H{"items": []gin.H{}})
			return
		}
		offers, err := a.Promotion.ListExchangeOffers(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(offers))
		for _, o := range offers {
			items = append(items, gin.H{
				"templateId": o.TemplateID, "name": o.Name, "type": o.Type,
				"faceValue": o.FaceValue, "threshold": o.Threshold, "pointsPrice": o.PointsPrice,
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// portalPointsTier GET /points/tier:我的等级(累计获得积分匹配,可为 null)。
func portalPointsTier(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		tier, err := a.Points.TierOf(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"tier": tier})
	}
}

// portalPointsTasks GET /points/tasks:任务列表 + 当前周期完成态。
func portalPointsTasks(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		tasks, err := a.Points.ListTasks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		done, err := a.Points.TaskStatus(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(tasks))
		for _, t := range tasks {
			items = append(items, gin.H{
				"taskId": t.TaskID, "code": t.Code, "name": t.Name,
				"points": t.Points, "period": t.Period,
				"status": t.Status, "completedAt": done[t.TaskID],
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"tasks": items})
	}
}

// portalPointsTaskComplete POST /points/tasks/:id/complete:完成任务领积分。
func portalPointsTaskComplete(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		bal, err := a.Points.CompleteTask(c.Request.Context(), cid, id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"balance": bal})
	}
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
