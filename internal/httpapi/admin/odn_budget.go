package adminapi

// 施工项目预算与里程碑路由(P-INFRA-1 W6/G2,迁移 000218;契约 admin/odn.yaml)。
// 编辑窗口:预算与里程碑清单仅项目 PENDING 可改;里程碑状态标记至 ACCEPTED 前;
// 全部写操作入审计,失败路径留 [odn-budget] 可 grep 日志(域层)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerODNBudgetRoutes 注册预算与里程碑路由(menu:odn 门禁;由 registerODNConstructionRoutes 调)。
func registerODNBudgetRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.PUT("/odn/constructions/:id/budget", perm, odnSetBudgetHandler(a))
	g.GET("/odn/constructions/:id/milestones", perm, odnListMilestonesHandler(a))
	g.POST("/odn/constructions/:id/milestones", perm, odnAddMilestoneHandler(a))
	g.PUT("/odn/milestones/:milestoneId", perm, odnUpdateMilestoneHandler(a))
	g.POST("/odn/milestones/:milestoneId/complete", perm, odnMarkMilestoneHandler(a, "DONE"))
	g.POST("/odn/milestones/:milestoneId/reopen", perm, odnMarkMilestoneHandler(a, "PENDING"))
}

// odnBudgetReq 预算设置请求体(amount=null 清除预算)。
type odnBudgetReq struct {
	BudgetAmount *float64 `json:"budgetAmount"`
}

// odnSetBudgetHandler PUT /odn/constructions/{id}/budget:设置/清除预算(仅 PENDING)。
func odnSetBudgetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req odnBudgetReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if req.BudgetAmount != nil && *req.BudgetAmount < 0 {
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "budgetAmount must be >= 0"})
			return
		}
		if err := a.ODN.SetProjectBudget(c.Request.Context(), id, req.BudgetAmount); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.budget.set", "construction_projects", c.Param("id"),
			map[string]any{"budgetAmount": req.BudgetAmount})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "budgetAmount": req.BudgetAmount})
	}
}

// odnListMilestonesHandler GET /odn/constructions/{id}/milestones:里程碑清单。
func odnListMilestonesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		list, err := a.ODN.ListMilestones(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnMilestoneReq 里程碑创建/编辑请求体(名称必填,计划日可空 YYYY-MM-DD)。
type odnMilestoneReq struct {
	Name        string `json:"name" binding:"required,max=128"`
	PlannedDate string `json:"plannedDate" binding:"omitempty,len=10"`
}

// odnAddMilestoneHandler POST /odn/constructions/{id}/milestones:追加里程碑(仅 PENDING)。
func odnAddMilestoneHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req odnMilestoneReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		m, err := a.ODN.AddMilestone(c.Request.Context(), id, req.Name, req.PlannedDate)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.milestone.create", "construction_milestones",
			strconv.FormatInt(m.ID, 10), map[string]any{"projectId": id, "name": req.Name, "plannedDate": req.PlannedDate})
		respond(c, apitypes.CodeOK, m)
	}
}

// odnUpdateMilestoneHandler PUT /odn/milestones/{mid}:编辑里程碑(项目须 PENDING)。
func odnUpdateMilestoneHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("milestoneId"), 10, 64)
		var req odnMilestoneReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.ODN.UpdateMilestone(c.Request.Context(), id, req.Name, req.PlannedDate); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.milestone.update", "construction_milestones", c.Param("milestoneId"),
			map[string]any{"name": req.Name, "plannedDate": req.PlannedDate})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// odnMarkMilestoneHandler POST /odn/milestones/{mid}/complete|reopen:状态标记(ACCEPTED 后锁定)。
func odnMarkMilestoneHandler(a *app.Application, to string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("milestoneId"), 10, 64)
		if err := a.ODN.MarkMilestone(c.Request.Context(), id, to); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.milestone.mark", "construction_milestones", c.Param("milestoneId"),
			map[string]any{"to": to})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": to})
	}
}
