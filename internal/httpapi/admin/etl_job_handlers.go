package adminapi

import (
	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/metric"
	"github.com/ymm-001/boss/pkg/apitypes"
	"strings"
	"time"
)

func registerETLRoutes(g *gin.RouterGroup, a *app.Application) {
	m := g.Group("", requirePerm(a.User, "menu:report"))
	m.GET("/etl-jobs", etlList(a))
	m.GET("/etl-jobs/:jobKey", etlGet(a))
	m.POST("/etl-jobs", etlUpsert(a))
	m.POST("/etl-jobs/:jobKey/disable", etlDisable(a))
	m.POST("/etl-jobs/:jobKey/runs", etlRun(a))
	m.GET("/etl-jobs/:jobKey/runs", etlRuns(a))
	m.GET("/etl-freshness", etlFresh(a))
}
func etlList(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, e := a.ETL.ListJobs(c)
		if e != nil {
			metricRespondErr(c, e)
			return
		}
		respond(c, apitypes.CodeOK, v)
	}
}
func etlGet(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, e := a.ETL.GetJob(c, strings.TrimSpace(c.Param("jobKey")))
		if e != nil {
			metricRespondErr(c, e)
			return
		}
		respond(c, apitypes.CodeOK, v)
	}
}
func etlUpsert(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var v metric.ETLJob
		if e := c.ShouldBindJSON(&v); e != nil {
			respond(c, apitypes.CodeInvalidParam, e.Error())
			return
		}
		v.JobKey = strings.TrimSpace(v.JobKey)
		x, e := a.ETL.UpsertJob(c, v)
		if e != nil {
			metricRespondErr(c, e)
			return
		}
		respond(c, apitypes.CodeOK, x)
	}
}
func etlDisable(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		e := a.ETL.DisableJob(c, strings.TrimSpace(c.Param("jobKey")))
		if e != nil {
			metricRespondErr(c, e)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}
func etlRun(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var v metric.ETLJobRun
		if e := c.ShouldBindJSON(&v); e != nil {
			respond(c, apitypes.CodeInvalidParam, e.Error())
			return
		}
		v.JobKey = strings.TrimSpace(c.Param("jobKey"))
		if v.StartedAt.IsZero() {
			v.StartedAt = time.Now()
		}
		if e := a.ETL.RecordRun(c, v); e != nil {
			metricRespondErr(c, e)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}
func etlRuns(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, e := a.ETL.ListRuns(c, strings.TrimSpace(c.Param("jobKey")))
		if e != nil {
			metricRespondErr(c, e)
			return
		}
		respond(c, apitypes.CodeOK, v)
	}
}
func etlFresh(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, e := a.ETL.ListFreshness(c)
		if e != nil {
			metricRespondErr(c, e)
			return
		}
		respond(c, apitypes.CodeOK, v)
	}
}
