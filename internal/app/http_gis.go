package app

// 阶段8 GIS 路由:八级下钻 / 全级数量统计 / 资产实时详情(承接 admin gis.html,menu:gis)。

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/gis"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerGisRoutes 注册 GIS 数字孪生路由。
func registerGisRoutes(g *gin.RouterGroup, a *Application) {
	gi := g.Group("", requirePerm(a.User, "menu:gis"))

	gi.GET("/gis/drill", func(c *gin.Context) {
		level, err := strconv.Atoi(c.Query("level"))
		if err != nil || level < 1 || level > 8 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		nodes, err := a.Gis.Drill(c.Request.Context(), int16(level), queryInt64(c, "parentId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": nodes})
	})

	gi.GET("/gis/levels", func(c *gin.Context) {
		counts, err := a.Gis.LevelCounts(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": counts})
	})

	gi.GET("/gis/resources/:resourceId/detail", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("resourceId"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		d, err := a.Gis.ResourceDetail(c.Request.Context(), id)
		if err != nil {
			if errors.Is(err, gis.ErrNotFound) {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, d)
	})
}
