package adminapi

// ODN 生命周期状态机路由(P6,迁移 000198;docs/plan/odn-lifecycle-roadmap.md T8)。
// 转移规则/留痕见 internal/domain/odn/lifecycle.go;此处只保留路由表+请求体+handler。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnLifecycleReq 生命周期转移请求体。
type odnLifecycleReq struct {
	LifecycleStatus string `json:"lifecycleStatus" binding:"required,oneof=PLANNED IN_BUILD IN_SERVICE RETIRED"`
}

// registerODNLifecycleRoutes 注册三实体生命周期转移路由(menu:odn 门禁;契约 admin/odn.yaml)。
func registerODNLifecycleRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.PUT("/odn/facilities/:code/lifecycle", perm, odnFacilityLifecycleHandler(a))
	g.PUT("/odn/sites/:siteNo/lifecycle", perm, odnSiteLifecycleHandler(a))
	g.PUT("/odn/devices/:id/lifecycle", perm, odnDeviceLifecycleHandler(a))
}

// odnFacilityLifecycleHandler PUT /odn/facilities/{code}/lifecycle。
func odnFacilityLifecycleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnLifecycleReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if err := a.ODN.SetFacilityLifecycle(c.Request.Context(), c.Param("code"), req.LifecycleStatus); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.lifecycle", "odn_facility", c.Param("code"), map[string]any{"to": req.LifecycleStatus})
		respond(c, apitypes.CodeOK, gin.H{"code": c.Param("code"), "lifecycleStatus": req.LifecycleStatus})
	}
}

// odnSiteLifecycleHandler PUT /odn/sites/{siteNo}/lifecycle。
func odnSiteLifecycleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnLifecycleReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		siteNo64, _ := strconv.ParseInt(c.Param("siteNo"), 10, 16)
		prv := c.Query("prvCode")
		city := c.Query("cityPrefix")
		if err := a.ODN.SetSiteLifecycle(c.Request.Context(), prv, city, int16(siteNo64), req.LifecycleStatus); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.lifecycle", "odn_site", prv+"/"+city+"/"+c.Param("siteNo"), map[string]any{"to": req.LifecycleStatus})
		respond(c, apitypes.CodeOK, gin.H{"siteNo": siteNo64, "lifecycleStatus": req.LifecycleStatus})
	}
}

// odnDeviceLifecycleHandler PUT /odn/devices/{id}/lifecycle。
func odnDeviceLifecycleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnLifecycleReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		if err := a.ODN.SetDeviceLifecycle(c.Request.Context(), id, req.LifecycleStatus); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.lifecycle", "odn_device", c.Param("id"), map[string]any{"to": req.LifecycleStatus})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "lifecycleStatus": req.LifecycleStatus})
	}
}
