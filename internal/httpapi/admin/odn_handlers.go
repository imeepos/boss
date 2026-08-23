package adminapi

// ODN 网格分区路由 handler 实现(承接 registerODNRoutes)。
// 设施/光缆/纤芯/局点/设备路由 handler 分别见 odn_facility_handlers.go、odn_device_handlers.go。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnListGridsHandler GET /odn/grids:网格列表。
func odnListGridsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.ODN.ListGrids(c.Request.Context(), c.Query("prvCode"), c.Query("cityPrefix"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnCreateGridHandler POST /odn/grids:新建网格(fields.md 1.5.3)。
func odnCreateGridHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnGridReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		gr := odn.Grid{PrvCode: c.Query("prvCode"), CityPrefix: c.Query("cityPrefix"),
			GridCode: req.GridCode, Name: req.Name, Coverage: req.Coverage, Status: req.Status}
		if gr.PrvCode == "" || gr.CityPrefix == "" {
			httpx.RespondValidationError(c, "prvCode/cityPrefix", "query params prvCode and cityPrefix are required")
			return
		}
		if err := a.ODN.CreateGrid(c.Request.Context(), gr); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"gridCode": gr.GridCode})
	}
}

// odnUpdateGridHandler PUT /odn/grids/{gridCode}:编辑网格。
func odnUpdateGridHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		gridCode, ok := odnPathParamInt(c, "gridCode")
		if !ok {
			return
		}
		var req odnGridReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		gr := odn.Grid{Name: req.Name, Coverage: req.Coverage, Status: req.Status}
		err := a.ODN.UpdateGrid(c.Request.Context(), c.Query("prvCode"), c.Query("cityPrefix"), gridCode, gr)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}

// odnRetireGridHandler DELETE /odn/grids/{gridCode}:网格作废。
func odnRetireGridHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		gridCode, ok := odnPathParamInt(c, "gridCode")
		if !ok {
			return
		}
		if err := a.ODN.RetireGrid(c.Request.Context(),
			c.Query("prvCode"), c.Query("cityPrefix"), gridCode); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}
