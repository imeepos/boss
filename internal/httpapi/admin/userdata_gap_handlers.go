package adminapi

// 用户数据域缺口路由的 handler 实现(userdata_gap.go 仅留路由表)。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// gapCreateFaq POST /faqs:契约字段 title/summary 映射 question/answer。
func gapCreateFaq(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Title    string `json:"title" binding:"required"`
			Summary  string `json:"summary"`
			Category string `json:"category"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		f := udcustomer.UserFaq{
			Category: defaultGapStr(req.Category, "general"),
			Question: req.Title, Answer: req.Summary, Active: true,
		}
		if err := a.UserData.CreateUserFaq(c.Request.Context(), f); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func gapToggleFaq(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := a.UserData.ToggleUserFaq(c.Request.Context(), c.Param("faqId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func gapListMaintenances(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Device.ListMaintenances(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// gapListHallItems 需商议DB改动:orders 无师傅指派字段,亦无应急/预告附加任务表,暂返回空列表。
func gapListHallItems(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{"items": []map[string]any{}})
	}
}

func gapListServiceMessages(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerLedger.ListMessages(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func gapSendServiceMessage(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			WorkerID int64  `json:"workerId" binding:"required"`
			Content  string `json:"content" binding:"required"`
			Title    string `json:"title"`
			Level    string `json:"level"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		m := worker.Message{
			WorkerID: req.WorkerID, Level: defaultGapStr(req.Level, "INFO"),
			Title: defaultGapStr(req.Title, "调度会话"), Content: req.Content, SentAt: time.Now(),
		}
		id, err := a.WorkerLedger.SendMessage(c.Request.Context(), m)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// gapGenerateReport POST /reports:复用 ReportService.Generate(幂等:同窗口覆盖)。
func gapGenerateReport(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Period string `json:"period"`
		}
		_ = c.ShouldBindJSON(&req) // 契约允许空体
		period := defaultGapStr(req.Period, "daily")
		snap, err := a.Report.Generate(c.Request.Context(), period, time.Now())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": snap.ID, "period": snap.Period})
	}
}

// gapDeleteAccount 复用 ListAccounts+UpdateAccount 做"取行回写停用"(status=0),
// 保留审计线索;真删需商议DB/域改动。
func gapDeleteAccount(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "accountId")
		if !ok {
			return
		}
		if !gapDisableAccount(c, a, id) {
			return
		}
		httpx.RecordAudit(a, c, "权限变更", "account", c.Param("accountId"),
			map[string]any{"op": "disable"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
