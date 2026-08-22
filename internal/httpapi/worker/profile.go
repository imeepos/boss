package workerapi

// W 师傅端门户个人域(worker/profile.yaml):我的/绩效/排期/打卡/接单设置/满意度。

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerPortalProfileRoutes 个人域路由(wauth 组)。
func registerWorkerPortalProfileRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/profile", workerProfileHandler(a))
	g.GET("/performance", workerPerformanceHandler(a))
	g.GET("/schedule", workerScheduleHandler(a))
	g.POST("/schedule/clock", workerClockHandler(a))
	g.GET("/settings", workerSettingsGetHandler(a))
	g.PUT("/settings", workerSettingsPutHandler(a))
	g.GET("/feedbacks", workerFeedbacksHandler(a))
}

// workerProfileHandler 我的:档案 + 本月累计(绩效行按月过滤)。
func workerProfileHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		w, err := a.Worker.GetWorker(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		perfs, _ := a.WorkerFact.ListPerformances(c.Request.Context(), workerID)
		finished, onTime, score := 0, 0, 0.0
		period := clock.Now().Format("2006-01")
		for _, p := range perfs {
			if p.Period == period {
				finished += int(p.Finished)
				onTime = int(p.OnTimeRate)
				score = p.Score
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"workerId": w.ID, "name": w.Name, "groupName": "", "staffNo": w.StaffNo,
			"phoneMasked": httpx.MaskPhone(w.Phone), "online": true, "serveYears": 0,
			"month": gin.H{"finished": finished, "onTimeRate": onTime, "score": score},
		})
	}
}

// workerPerformanceHandler 绩效:汇总 + 佣金明细。
func workerPerformanceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		period := c.Query("period")
		if period == "" {
			period = clock.Now().Format("2006-01")
		}
		perfs, err := a.WorkerFact.ListPerformances(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		comms, err := a.WorkerFact.ListCommissions(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"period": period, "summary": portalPerfSummary(perfs, period),
			"commissions": portalCommissions(comms, period), "totalAmount": 0,
			"ranking": []gin.H{},
		})
	}
}

// portalPerfSummary 按月聚合绩效摘要。
func portalPerfSummary(perfs []worker.Performance, period string) gin.H {
	finished, onTime, score := 0, 0, 0.0
	for _, p := range perfs {
		if p.Period == period {
			finished += int(p.Finished)
			onTime = int(p.OnTimeRate)
			score = p.Score
		}
	}
	return gin.H{"finished": finished, "onTimeRate": onTime, "score": score}
}

// portalCommissions 按月过滤佣金明细视图。
func portalCommissions(comms []worker.Commission, period string) []gin.H {
	out := make([]gin.H, 0)
	for _, cm := range comms {
		if cm.Period == period {
			out = append(out, gin.H{
				"name": "提成", "formula": cm.Formula, "amount": cm.Amount,
			})
		}
	}
	return out
}

// workerScheduleHandler 排期日历:月度排期行 → busyDays。
func workerScheduleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		month := c.Query("month")
		if month == "" {
			month = clock.Now().Format("2006-01")
		}
		rows, err := a.WorkerFact.ListSchedules(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		busy := make([]int16, 0)
		for _, s := range rows {
			if s.Month == month && s.BusyDays > 0 {
				busy = append(busy, s.BusyDays)
			}
		}
		clocks, err := a.WorkerLedger.ListClocks(c.Request.Context(), workerID, clock.Now())
		if err != nil {
			respondErr(c, err)
			return
		}
		today := make([]gin.H, 0, len(clocks))
		for _, a := range clocks {
			today = append(today, gin.H{"type": a.ClockType, "clockedAt": a.ClockedAt.In(clock.Location()).Format("15:04")})
		}
		respond(c, apitypes.CodeOK, gin.H{"month": month, "busyDays": busy, "today": today})
	}
}

// workerClockHandler 工时打卡:落 worker_attendance 流水,回执当前时间与当日记录。
func workerClockHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		var req workerClockReq
		if !httpx.BindAndValidate(c, &req, func() error {
			if req.Type != "IN" && req.Type != "OUT" {
				return &httpx.ValidationError{Field: "type", Message: "must be IN or OUT"}
			}
			return nil
		}) {
			return
		}
		now := clock.Now()
		if _, err := a.WorkerLedger.AppendClock(c.Request.Context(), worker.Attendance{
			WorkerID: workerID, ClockType: req.Type, ClockedAt: now,
		}); err != nil {
			respondErr(c, err)
			return
		}
		clocks, err := a.WorkerLedger.ListClocks(c.Request.Context(), workerID, now)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"clockedAt": now.Format("15:04"), "today": clockToday(clocks)})
	}
}

// clockToday 当日打卡记录渲染(HH:MM)。
func clockToday(clocks []worker.Attendance) []gin.H {
	today := make([]gin.H, 0, len(clocks))
	for _, a := range clocks {
		today = append(today, gin.H{"type": a.ClockType, "clockedAt": a.ClockedAt.In(clock.Location()).Format("15:04")})
	}
	return today
}

// workerClockReq 打卡请求体;type 取契约枚举 IN/OUT。
type workerClockReq struct {
	Type string `json:"type"`
}

// workerSettingsGetHandler 接单设置查询;未设置返回默认。
func workerSettingsGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		s, err := a.WorkerLedger.GetSettings(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if s == nil {
			respond(c, apitypes.CodeOK, gin.H{"online": true, "radiusKm": 5, "acceptTypes": []string{}})
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"online": s.Accepting, "radiusKm": s.RadiusKm,
			"acceptTypes": strings.Split(s.AcceptTypes, ","),
		})
	}
}

// workerSettingsPutHandler 保存接单设置。
func workerSettingsPutHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req workerSettingsReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		workerID, _ := portalWorker(c)
		_, err := a.WorkerLedger.UpsertSettings(c.Request.Context(), workerSettingsOf(workerID, &req))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerFeedbacksHandler 满意度:评分聚合 + 明细(低分 needReview)。
func workerFeedbacksHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		list, err := a.WorkerEvent.ListFeedbacks(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(list))
		total := 0
		for _, f := range list {
			total += int(f.Score)
			items = append(items, gin.H{
				"customerName": f.CustomerName, "score": f.Score,
				"comment": "", "needReview": f.NeedReview,
			})
		}
		avg := 0.0
		if len(list) > 0 {
			avg = float64(total) / float64(len(list))
		}
		respond(c, apitypes.CodeOK, gin.H{
			"latest": nil, "monthAvgScore": avg, "replyRate": 0, "items": items,
		})
	}
}

// workerSettingsOf 请求 → 域设置(acceptTypes 逗号连接存储)。
func workerSettingsOf(workerID int64, req *workerSettingsReq) worker.Settings {
	return worker.Settings{
		WorkerID: workerID, Accepting: req.Online, RadiusKm: req.RadiusKm,
		AcceptTypes: strings.Join(req.AcceptTypes, ","),
	}
}
