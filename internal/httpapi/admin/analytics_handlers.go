package adminapi

// 经营分析/自动报告路由 handler 实现(承接 registerAnalyticsRoutes / registerReportRoutes)。

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// analyticsFiveIndicatorsHandler GET /analytics/indicators:五大指标 + 区域 ROI。
func analyticsFiveIndicatorsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ind, rois, err := a.Analytics.FiveIndicators(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"indicators": ind, "regionROI": rois})
	}
}

// analyticsHeatmapHandler GET /analytics/heatmap:需求热力图。
func analyticsHeatmapHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cells, err := a.Analytics.Heatmap(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": cells})
	}
}

// analyticsMaintenanceHandler GET /analytics/maintenance:维护清单。
func analyticsMaintenanceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := a.Analytics.MaintenanceList(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// reportPatrolOrphansHandler GET /db-patrol/orphans:软引用孤儿巡检(db-design-review D7):只读报表,不阻断业务。
func reportPatrolOrphansHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		findings, err := a.Report.PatrolOrphans(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": findings})
	}
}

// reportLatestPatrolHandler GET /db-patrol/latest:最新巡检快照(定时循环落 report_snapshots);无快照 404。
func reportLatestPatrolHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// reportListHandler GET /reports:报告列表。
func reportListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Report.List(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// reportLatestHandler GET /reports/latest:按周期取最新快照。
func reportLatestHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// reportSendHandler POST /reports/{reportId}/send:报告推送(按 id 取快照 → Push,intel.yaml /reports/{reportId}/send)。
func reportSendHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "reportId")
		if !ok {
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
		emitTask(c.Request.Context(), a, refReport, strconv.FormatInt(snap.ID, 10),
			"报告已推送:"+snap.Period, linkReport, false)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// reportHistoryHandler GET /reports/history?period&limit:历史快照(新→旧,trend 用)。
// limit 默认 12,上限 90(handler 端再夹一次,服务层兜底)。
func reportHistoryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		period := c.DefaultQuery("period", "daily")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))
		if limit <= 0 {
			limit = 12
		}
		if limit > 90 {
			limit = 90
		}
		snaps, err := a.Report.History(c.Request.Context(), period, limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		// 每份快照解出 Payload(与 reportLatestHandler 同形,字段展开给前端免二次解码)。
		out := make([]gin.H, 0, len(snaps))
		for _, s := range snaps {
			var body report.Payload
			if err := json.Unmarshal(s.Payload, &body); err != nil {
				respondErr(c, err)
				return
			}
			out = append(out, gin.H{
				"id": s.ID, "period": s.Period,
				"windowStart": s.WindowStart, "windowEnd": s.WindowEnd,
				"createdAt": s.CreatedAt, "payload": body,
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": out})
	}
}