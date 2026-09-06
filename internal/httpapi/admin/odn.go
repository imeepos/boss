package adminapi

// ODN 无源物理层路由注册(menu:odn 门禁;契约 api/openapi/admin/odn.yaml)。
// 网格 handler 实现见 odn_handlers.go;设施/光缆/纤芯/局点/设备路由注册沿用各自文件。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
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

	g.GET("/odn/grids", perm, odnListGridsHandler(a))
	g.POST("/odn/grids", perm, odnCreateGridHandler(a))
	g.PUT("/odn/grids/:gridCode", perm, odnUpdateGridHandler(a))
	g.DELETE("/odn/grids/:gridCode", perm, odnRetireGridHandler(a))

	registerODNFacilityRoutes(g, a, perm)
	registerODNCableRoutes(g, a, perm)
	registerODNSiteDeviceRoutes(g, a, perm)
	registerODNCoverageRoutes(g, a, perm)
	registerODNLifecycleRoutes(g, a, perm)
	registerODNConstructionRoutes(g, a, perm)
	registerODNImpactRoutes(g, a, perm)
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
