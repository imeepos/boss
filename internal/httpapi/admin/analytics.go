package adminapi

// 阶段9 经营分析路由:五大指标/热力图/维护清单/自动报告(承接 admin analytics.html、report.html)。

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerAnalyticsRoutes 注册经营分析路由(menu:analytics)。
func registerAnalyticsRoutes(g *gin.RouterGroup, a *app.Application) {
	an := g.Group("", requirePerm(a.User, "menu:analytics"))

	an.GET("/analytics/indicators", func(c *gin.Context) {
		ind, rois, err := a.Analytics.FiveIndicators(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"indicators": ind, "regionROI": rois})
	})

	an.GET("/analytics/heatmap", func(c *gin.Context) {
		cells, err := a.Analytics.Heatmap(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": cells})
	})

	an.GET("/analytics/maintenance", func(c *gin.Context) {
		items, err := a.Analytics.MaintenanceList(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	})
}

// registerReportRoutes 注册自动报告路由(menu:report)。
func registerReportRoutes(g *gin.RouterGroup, a *app.Application) {
	rp := g.Group("", requirePerm(a.User, "menu:report"))

	// 软引用孤儿巡检(db-design-review D7):只读报表,不阻断业务。
	rp.GET("/db-patrol/orphans", func(c *gin.Context) {
		findings, err := a.Report.PatrolOrphans(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": findings})
	})

	// 最新巡检快照(定时循环落 report_snapshots);无快照 404。
	rp.GET("/db-patrol/latest", func(c *gin.Context) {
		p, err := a.Report.LatestPatrol(c.Request.Context())
		if err != nil {
			if errors.Is(err, report.ErrNoSnapshot) {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, p)
	})

	rp.GET("/reports", func(c *gin.Context) {
		list, err := a.Report.List(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	rp.GET("/reports/latest", func(c *gin.Context) {
		period := c.DefaultQuery("period", "daily")
		snap, err := a.Report.Latest(c.Request.Context(), period)
		if err != nil {
			if errors.Is(err, report.ErrNoSnapshot) {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		var body report.Payload
		if err := json.Unmarshal(snap.Payload, &body); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"id": snap.ID, "period": snap.Period,
			"windowStart": snap.WindowStart, "windowEnd": snap.WindowEnd,
			"createdAt": snap.CreatedAt, "payload": body,
		})
	})

	// 报告推送:按 id 取快照 → Push(intel.yaml /reports/{reportId}/send)。
	rp.POST("/reports/:reportId/send", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("reportId"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		snap, err := a.Report.St.SnapshotByID(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, report.ErrNoSnapshot) {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		if err := a.Report.Push(c.Request.Context(), snap); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}
