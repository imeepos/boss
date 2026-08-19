package app

// registerProvisionSeedRoutes 后台置备端点(CLI 联调/开网全流程模拟用)。
// 口径:工程质量"先可观测后自动化"——先落地置备端点,供管理后台与 CLI 走通 12 环节流程;
// 域复用既有 create 服务口,不重复造轮子。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerProvisionSeedRoutes 注册置备端点:资源/端口、渠道、资产批次/标签/资产、派单工单。
func registerProvisionSeedRoutes(g *gin.RouterGroup, a *Application) {
	p := g.Group("/provision", requirePerm(a.User, "menu:provision"))

	// 网络资源(含端口):环节2/3 前置。
	p.POST("/resources", func(c *gin.Context) {
		var r resource.Resource
		if err := c.ShouldBindJSON(&r); err != nil || r.AddressID == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.Resource.CreateResource(c.Request.Context(), r)
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "数据变更", "resource", strItoa(id), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	p.POST("/ports", func(c *gin.Context) {
		var p2 resource.Port
		if err := c.ShouldBindJSON(&p2); err != nil || p2.ResourceID == 0 || p2.AddressID == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if p2.Status == "" {
			p2.Status = "IDLE"
		}
		portID, err := a.Resource.CreatePort(c.Request.Context(), p2)
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "数据变更", "port", strItoa(portID), nil)
		respond(c, apitypes.CodeOK, gin.H{"portId": portID})
	})

	// 渠道:下单前置。
	p.POST("/channels", func(c *gin.Context) {
		var ch order.Channel
		if err := c.ShouldBindJSON(&ch); err != nil || ch.Code == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.Channel.CreateChannel(c.Request.Context(), ch)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	// 资产批次/标签/资产:环节5 标签预绑定与环节9 扫码前置。
	p.POST("/asset-batches", func(c *gin.Context) {
		var b asset.AssetBatch
		if err := c.ShouldBindJSON(&b); err != nil || b.Code == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.Asset.CreateBatch(c.Request.Context(), b)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	p.POST("/tags", func(c *gin.Context) {
		var t asset.Tag
		if err := c.ShouldBindJSON(&t); err != nil || t.EpcCode == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.Asset.CreateTag(c.Request.Context(), t)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	p.POST("/assets", func(c *gin.Context) {
		var as asset.Asset
		if err := c.ShouldBindJSON(&as); err != nil || as.BatchID == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.Asset.CreateAsset(c.Request.Context(), as)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	// 派单工单:环节8 派单后置备工单,再入池指派师傅。
	p.POST("/dispatch-tickets", func(c *gin.Context) {
		var t order.DispatchTicket
		if err := c.ShouldBindJSON(&t); err != nil || t.TicketNo == "" || t.OrderID == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if t.Status == "" {
			t.Status = "PENDING"
		}
		id, err := a.WorkOrder.CreateDispatchTicket(c.Request.Context(), t)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})
}

// strItoa int64 → string(审计记录用,避免引入 fmt)。
func strItoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
