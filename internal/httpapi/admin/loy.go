package adminapi

// 积分域管理端点:余额/流水查询 + 手动调整(000104 最小 LOY)。

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/loy"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerLoyRoutes 注册积分域管理路由(权限挂 menu:userdata 组,与促销域同门)。
func registerLoyRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:userdata")
	g.GET("/points/:customerId", perm, loyGetPoints(a))
	g.POST("/points/:customerId/adjust", perm, loyAdjustPoints(a))

	g.GET("/loy/levels", perm, loyListLevels(a))
	g.POST("/loy/levels", perm, loyCreateLevel(a))
	g.POST("/loy/levels/:id/disable", perm, loyDisableLevel(a))

	g.GET("/loy/tasks", perm, loyListTasks(a))
	g.POST("/loy/tasks", perm, loyCreateTask(a))
	g.POST("/loy/tasks/:id/disable", perm, loyDisableTask(a))

	g.GET("/loy/earn-rule", perm, loyGetEarnRule(a))
	g.PUT("/loy/earn-rule", perm, loySaveEarnRule(a))

	g.POST("/loy/expire/run", perm, loyRunExpire(a))
	g.GET("/loy/points-recon", perm, loyPointsRecon(a))
}

// pointsReconer Points 的可选对账能力(PGStore 实现,窄口断言)。
type pointsReconer interface {
	PointsRecon(ctx context.Context) ([]loy.PointReconRow, loy.PointReconSummary, error)
}

// loyPointsRecon GET /loy/points-recon:积分对账报表(diff=drift 只看差异行)。
func loyPointsRecon(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		rc, ok := a.Points.(pointsReconer)
		if !ok {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		rows, sum, err := rc.PointsRecon(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		if c.Query("diff") == "drift" {
			filtered := make([]loy.PointReconRow, 0, len(rows))
			for _, r := range rows {
				if r.DiffKind != loy.ReconDiffMatch {
					filtered = append(filtered, r)
				}
			}
			rows = filtered
		}
		respond(c, apitypes.CodeOK, gin.H{"rows": rows, "summary": sum})
	}
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

// loyListLevels GET /loy/levels:积分等级列表。
func loyListLevels(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Points.ListLevels(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"levels": list})
	}
}

// loyCreateLevel POST /loy/levels {name,minPoints}:新建等级。
func loyCreateLevel(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loy.Level
		if !httpx.BindBody(c, &req) {
			return
		}
		id, err := a.Points.CreateLevel(c.Request.Context(), req)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"levelId": id})
	}
}

// loyDisableLevel POST /loy/levels/:id/disable:停用等级。
func loyDisableLevel(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "id")
		if !ok {
			return
		}
		if err := a.Points.DisableLevel(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}

// loyListTasks GET /loy/tasks:任务列表。
func loyListTasks(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Points.ListTasks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"tasks": list})
	}
}

// loyCreateTask POST /loy/tasks {code,name,points,period}:新建任务。
func loyCreateTask(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loy.Task
		if !httpx.BindBody(c, &req) {
			return
		}
		id, err := a.Points.CreateTask(c.Request.Context(), req)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"taskId": id})
	}
}

// loyDisableTask POST /loy/tasks/:id/disable:停用任务。
func loyDisableTask(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "id")
		if !ok {
			return
		}
		if err := a.Points.DisableTask(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}

// loyGetEarnRule GET /loy/earn-rule:当前生效缴费送积分规则。
func loyGetEarnRule(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, err := a.Points.EarnRuleOf(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"rule": r})
	}
}

// loySaveEarnRule PUT /loy/earn-rule {pointsPerYuan,minCents,expireDays}:保存规则。
func loySaveEarnRule(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loy.EarnRule
		if !httpx.BindBody(c, &req) {
			return
		}
		id, err := a.Points.SaveEarnRule(c.Request.Context(), req)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ruleId": id})
	}
}

// loyRunExpire POST /loy/expire/run:手动触发积分过期清算(循环兜底之外的运营口)。
func loyRunExpire(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		n, err := a.Points.ExpireDue(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"customers": n})
	}
}
