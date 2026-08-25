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

// gisPoints GET /gis/points?level&parentId&bbox:地图点位(Point 列表)。
// 前端收到 items 后自行拼 GeoJSON FeatureCollection(type/features);后端不耦合 GeoJSON 字段。
// bbox 格式:"minLng,minLat,maxLng,maxLat"(WGS84);空=不限。
func gisPoints(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		level, err := strconv.Atoi(c.Query("level"))
		if err != nil || level < 1 || level > 8 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		pts, err := a.Gis.Points(c.Request.Context(), int16(level), queryInt64(c, "parentId"), c.Query("bbox"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": pts})
	}
}

// gisODNPoints GET /gis/odn-points?entity=facility|site|device&bbox:
// ODN 无源物理层点位(设施/局点/设备自带 lat/lng,直接作地图图层)。
func gisODNPoints(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		pts, err := a.Gis.ODNPoints(c.Request.Context(), c.Query("entity"), c.Query("bbox"))
		if err != nil {
			if errors.Is(err, gis.ErrODNEntityInvalid) {
				respond(c, apitypes.CodeInvalidParam, nil)
				return
			}
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": pts})
	}
}
