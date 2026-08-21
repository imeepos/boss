package adminapi

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnGridReq 网格新建/编辑请求体(fields.md 1.5.3)。
type odnGridReq struct {
	GridCode int16  `json:"gridCode" binding:"required,min=1,max=99"`
	Name     string `json:"name"`
	Coverage string `json:"coverage"`
	Status   string `json:"status" binding:"required,oneof=ACTIVE RESERVED"`
}

// odnFacilityReq 设施新建请求体。
type odnFacilityReq struct {
	Code       string  `json:"code" binding:"required"`
	Kind       string  `json:"kind" binding:"required,oneof=P MH TW CLS TBX"`
	PrvCode    string  `json:"prvCode" binding:"required,len=6"`
	CityPrefix string  `json:"cityPrefix" binding:"required,min=3,max=5"`
	GridCode   int16   `json:"gridCode" binding:"min=0,max=99"`
	Name       string  `json:"name"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
}

// odnSegmentReq 光缆段落新建请求体(端点顺序任意,服务端按规范 5.2 定向)。
type odnSegmentReq struct {
	Endpoint1 string `json:"endpoint1" binding:"required"`
	Endpoint2 string `json:"endpoint2" binding:"required"`
	Name      string `json:"name"`
}

// odnFiberReq 纤芯新增请求体。
type odnFiberReq struct {
	GNo  int16  `json:"gNo" binding:"required,min=1,max=99"`
	Kind string `json:"kind"`
}

// registerODNRoutes 注册 ODN 无源物理层路由(menu:odn 门禁;契约 api/openapi/admin/odn.yaml)。
func registerODNRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:odn")

	// 网格分区。
	g.GET("/odn/grids", perm, func(c *gin.Context) {
		list, err := a.ODN.ListGrids(c.Request.Context(), c.Query("prvCode"), c.Query("cityPrefix"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})
	g.POST("/odn/grids", perm, func(c *gin.Context) {
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
	})
	g.PUT("/odn/grids/:gridCode", perm, func(c *gin.Context) {
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
	})
	g.DELETE("/odn/grids/:gridCode", perm, func(c *gin.Context) {
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
	})

	registerODNFacilityRoutes(g, a, perm)
	registerODNCableRoutes(g, a, perm)
	registerODNSiteDeviceRoutes(g, a, perm)
}

// odnPathParamInt 路径参数转 int16,失败已回 400。
func odnPathParamInt(c *gin.Context, name string) (int16, bool) {
	v, err := strconv.Atoi(c.Param(name))
	if err != nil || v < 1 || v > 99 {
		respond(c, apitypes.CodeInvalidParam, nil)
		return 0, false
	}
	return int16(v), true
}
