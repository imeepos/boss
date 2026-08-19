package app

// 用户数据域缺口路由:契约 admin.yaml 汇总中尚未落地的端点(worker.yaml faqs/
// device-maintenances/hall-items/service-messages、intel.yaml POST /reports、
// sys.yaml DELETE /accounts)。接线(挂 authed 组)由父会话在 http.go 完成。

import (
	"time"

	"github.com/gin-gonic/gin"

	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerUserdataGapRoutes 注册用户数据域缺口路由。
func registerUserdataGapRoutes(g *gin.RouterGroup, a *Application) {
	registerGapFaqRoutes(g, a)
	registerGapDeviceRoutes(g, a)
	registerGapMessageRoutes(g, a)
	registerGapReportRoutes(g, a)
	registerGapAccountRoutes(g, a)
}

// registerGapFaqRoutes FAQ 知识库(worker.yaml /faqs 族):复用 user_faqs 表,
// 契约字段 title/summary 映射 question/answer。
func registerGapFaqRoutes(g *gin.RouterGroup, a *Application) {
	ud := a.UserData
	udList(g, a, "/faqs", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListUserFaqs(c.Request.Context())
	})
	g.POST("/faqs", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var req struct {
			Title    string `json:"title" binding:"required"`
			Summary  string `json:"summary"`
			Category string `json:"category"`
		}
		if !bindBody(c, &req) {
			return
		}
		f := udcustomer.UserFaq{
			Category: defaultGapStr(req.Category, "general"),
			Question: req.Title, Answer: req.Summary, Active: true,
		}
		if err := ud.CreateUserFaq(c.Request.Context(), f); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
	g.PUT("/faqs/:faqId/toggle", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		if err := ud.ToggleUserFaq(c.Request.Context(), c.Param("faqId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}

// registerGapDeviceRoutes 设备健康观察 + 抢单池。
func registerGapDeviceRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/device-maintenances", requirePerm(a.User, "menu:alarm"), func(c *gin.Context) {
		list, err := a.Device.ListMaintenances(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	// 需商议DB改动:orders 无师傅指派字段,亦无应急/预告附加任务表,暂返回空列表。
	g.GET("/hall-items", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{"items": []map[string]any{}})
	})
}

// registerGapMessageRoutes 调度会话记录(worker.yaml /service-messages):
// 复用 worker_messages 台账。
func registerGapMessageRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/service-messages", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		list, err := a.WorkerLedger.ListMessages(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	g.POST("/service-messages", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		var req struct {
			WorkerID int64  `json:"workerId" binding:"required"`
			Content  string `json:"content" binding:"required"`
			Title    string `json:"title"`
			Level    string `json:"level"`
		}
		if !bindBody(c, &req) {
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
	})
}

// registerGapReportRoutes 生成经营分析报告(intel.yaml POST /reports):
// 复用 ReportService.Generate(幂等:同窗口覆盖)。
func registerGapReportRoutes(g *gin.RouterGroup, a *Application) {
	g.POST("/reports", requirePerm(a.User, "menu:report"), func(c *gin.Context) {
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
	})
}

// registerGapAccountRoutes 删除账号(sys.yaml DELETE /accounts/{accountId})。
// 用户域无硬删服务:复用 ListAccounts+UpdateAccount 做"取行回写停用"(status=0),
// 保留审计线索;真删需商议DB/域改动。
func registerGapAccountRoutes(g *gin.RouterGroup, a *Application) {
	g.DELETE("/accounts/:accountId", requirePerm(a.User, "menu:account"), func(c *gin.Context) {
		id, ok := pathInt64(c, "accountId")
		if !ok {
			return
		}
		if !gapDisableAccount(c, a, id) {
			return
		}
		a.recordAudit(c, "权限变更", "account", c.Param("accountId"),
			map[string]any{"op": "disable"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}

// gapDisableAccount 取原行回写停用;返回 false 表示已响应(未命中/写失败)。
func gapDisableAccount(c *gin.Context, a *Application, id int64) bool {
	rows, err := a.User.ListAccounts(c.Request.Context())
	if err != nil {
		respondErr(c, err)
		return false
	}
	var row *user.AccountRow
	for i := range rows {
		if rows[i].ID == id {
			row = &rows[i]
			break
		}
	}
	if row == nil {
		respond(c, apitypes.CodeNotFound, nil)
		return false
	}
	return gapWriteDisabled(c, a, row)
}

// gapWriteDisabled 按原行构造 AccountInput,仅翻转 status=0。
func gapWriteDisabled(c *gin.Context, a *Application, row *user.AccountRow) bool {
	off := int16(0)
	in := user.AccountInput{
		Username: row.Username, RealName: row.RealName, Phone: row.Phone,
		RoleCode: row.RoleCode, Status: &off,
	}
	if row.LegalEntityID != 0 {
		in.LegalEntityID = &row.LegalEntityID
	}
	if row.DeptID != 0 {
		in.DeptID = &row.DeptID
	}
	if row.PostID != 0 {
		in.PostID = &row.PostID
	}
	if row.RegionScope != "" {
		in.RegionScope = &row.RegionScope
	}
	if err := a.User.UpdateAccount(c.Request.Context(), row.ID, in); err != nil {
		respondErr(c, err)
		return false
	}
	return true
}

// defaultGapStr 空串兜底。
func defaultGapStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
