package adminapi

// ODN 逻辑-物理绑定路由(P3,T13,迁移 000201;售后与 GIS 反查数据源)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnBindingReq 绑定请求体。
type odnBindingReq struct {
	PortID         int64  `json:"portId" binding:"required,min=1"`
	OrderID        int64  `json:"orderId" binding:"required,min=1"`
	ResourcePortID int64  `json:"resourcePortId"`
	Note           string `json:"note"`
}

// registerODNBindingRoutes 注册绑定路由(menu:odn 门禁;契约 admin/odn.yaml)。
func registerODNBindingRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.POST("/odn/bindings", perm, odnBindPortHandler(a))
	g.GET("/odn/bindings", perm, odnListBindingsHandler(a))
	g.DELETE("/odn/bindings/port/:portId", perm, odnUnbindHandler(a))
}

// odnBindPortHandler POST /odn/bindings:写绑定事实(端口须 IN_SERVICE)。
func odnBindPortHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnBindingReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		b := odn.ODNBinding{PortID: req.PortID, OrderID: req.OrderID, ResourcePortID: req.ResourcePortID, Note: req.Note}
		if err := a.ODN.BindPort(c.Request.Context(), b); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.binding.bind", "odn_port", strconv.FormatInt(req.PortID, 10), map[string]any{"order": req.OrderID})
		respond(c, apitypes.CodeOK, gin.H{"portId": req.PortID, "orderId": req.OrderID})
	}
}

// odnListBindingsHandler GET /odn/bindings?portId=|orderId=:按口或按单反查。
func odnListBindingsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		qPort := c.Query("portId")
		qOrder := c.Query("orderId")
		switch {
		case qPort != "":
			id, _ := strconv.ParseInt(qPort, 10, 64)
			list, err := a.ODN.ListBindingsByPort(c.Request.Context(), id)
			if err != nil {
				respondErr(c, err)
				return
			}
			respond(c, apitypes.CodeOK, list)
		case qOrder != "":
			id, _ := strconv.ParseInt(qOrder, 10, 64)
			list, err := a.ODN.ListBindingsByOrder(c.Request.Context(), id)
			if err != nil {
				respondErr(c, err)
				return
			}
			respond(c, apitypes.CodeOK, list)
		default:
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "portId or orderId required"})
		}
	}
}

// odnUnbindHandler DELETE /odn/bindings/port/{portId}:解绑。
func odnUnbindHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		portID, _ := strconv.ParseInt(c.Param("portId"), 10, 64)
		if err := a.ODN.UnbindPort(c.Request.Context(), portID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.binding.unbind", "odn_port", c.Param("portId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"portId": portID})
	}
}
