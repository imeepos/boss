package adminapi

// GIS 域 handler 实现(gis.go 仅留路由表)。

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/gis"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func gisDrill(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func gisLevelCounts(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		counts, err := a.Gis.LevelCounts(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": counts})
	}
}

func gisResourceDetail(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "resourceId")
		if !ok {
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
	}
}
