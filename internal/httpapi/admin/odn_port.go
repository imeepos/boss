package adminapi

// ODN 物理端口路由(P2,T11,迁移 000200;下单门控数据基础)。
// 状态机/CAS/留痕见 internal/domain/odn/pg_port.go;按地址分配=覆盖关联兑现。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnPortAllocReq 端口分配请求体。
type odnPortAllocReq struct {
	OrderID int64 `json:"orderId" binding:"required,min=1"`
}

// odnAddrAllocReq 按地址分配请求体。
type odnAddrAllocReq struct {
	AddressID int64 `json:"addressId" binding:"required,min=1"`
	OrderID   int64 `json:"orderId" binding:"required,min=1"`
}

// registerODNPortRoutes 注册物理端口路由(menu:odn 门禁;契约 admin/odn.yaml)。
func registerODNPortRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.GET("/odn/devices/:id/ports", perm, odnListPortsHandler(a))
	g.POST("/odn/devices/:id/ports/allocate", perm, odnAllocatePortHandler(a))
	g.POST("/odn/ports/allocate-for-address", perm, odnAllocateForAddressHandler(a))
	g.POST("/odn/ports/:portId/release", perm, odnReleasePortHandler(a))
	g.POST("/odn/ports/:portId/activate", perm, odnActivatePortHandler(a))
}

// odnListPortsHandler GET /odn/devices/{id}/ports:端口列表。
func odnListPortsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		list, err := a.ODN.ListPorts(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnAllocatePortHandler POST /odn/devices/{id}/ports/allocate:设备内分配空闲端口(IDLE→RESERVED)。
func odnAllocatePortHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnPortAllocReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		p, err := a.ODN.AllocatePort(c.Request.Context(), id, req.OrderID)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.port.allocate", "odn_port", c.Param("id"), map[string]any{"portNo": p.PortNo, "order": req.OrderID})
		respond(c, apitypes.CodeOK, p)
	}
}

// odnAllocateForAddressHandler POST /odn/ports/allocate-for-address:覆盖关联兑现,地址直达物理口。
func odnAllocateForAddressHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnAddrAllocReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		p, err := a.ODN.AllocateForAddress(c.Request.Context(), req.AddressID, req.OrderID)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.port.allocate-for-address", "odn_port", strconv.FormatInt(p.ID, 10), map[string]any{"address": req.AddressID, "order": req.OrderID})
		respond(c, apitypes.CodeOK, p)
	}
}

// odnReleasePortHandler POST /odn/ports/{portId}/release:释放端口(RESERVED/IN_SERVICE→IDLE)。
func odnReleasePortHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("portId"), 10, 64)
		if err := a.ODN.ReleasePort(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.port.release", "odn_port", c.Param("portId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": "IDLE"})
	}
}

// odnActivatePortHandler POST /odn/ports/{portId}/activate:端口开通(RESERVED→IN_SERVICE)。
func odnActivatePortHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("portId"), 10, 64)
		if err := a.ODN.ActivatePort(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.port.activate", "odn_port", c.Param("portId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": "IN_SERVICE"})
	}
}
