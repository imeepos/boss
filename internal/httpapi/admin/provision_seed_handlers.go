package adminapi

// 后台置备端点 handler 实现(从 provision_seed.go 抽出,registerProvisionSeedRoutes 只剩扁平路由表)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// createResourceHandler POST /provision/resources:环节2/3 前置。
func createResourceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r resource.Resource
		if !httpx.BindAndValidate(c, &r, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(r.AddressID, "addressId"),
			)
		}) {
			return
		}
		id, err := a.Resource.CreateResource(c.Request.Context(), r)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "resource", strItoa(id), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// createPortHandler POST /provision/ports:网络资源端口。
func createPortHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p2 resource.Port
		if !httpx.BindAndValidate(c, &p2, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(p2.ResourceID, "resourceId"),
				httpx.RequirePositiveID(p2.AddressID, "addressId"),
			)
		}) {
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
		httpx.RecordAudit(a, c, "数据变更", "port", strItoa(portID), nil)
		respond(c, apitypes.CodeOK, gin.H{"portId": portID})
	}
}

// createChannelHandler POST /provision/channels:下单前置。
func createChannelHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ch order.Channel
		if !httpx.BindAndValidate(c, &ch, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(ch.Code, "code", 32),
			)
		}) {
			return
		}
		id, err := a.Channel.CreateChannel(c.Request.Context(), ch)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// createAssetBatchHandler POST /provision/asset-batches:环节5 标签预绑定。
func createAssetBatchHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var b asset.AssetBatch
		if !httpx.BindAndValidate(c, &b, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(b.Code, "code", 64),
			)
		}) {
			return
		}
		id, err := a.Asset.CreateBatch(c.Request.Context(), b)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// createTagHandler POST /provision/tags:标签预绑定。
func createTagHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var t asset.Tag
		if !httpx.BindAndValidate(c, &t, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(t.EpcCode, "epcCode", 64),
			)
		}) {
			return
		}
		id, err := a.Asset.CreateTag(c.Request.Context(), t)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// createAssetHandler POST /provision/assets:环节9 扫码前置。
func createAssetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var as asset.Asset
		if !httpx.BindAndValidate(c, &as, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(as.BatchID, "batchId"),
			)
		}) {
			return
		}
		id, err := a.Asset.CreateAsset(c.Request.Context(), as)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// createDispatchTicketHandler POST /provision/dispatch-tickets:环节8 派单后置备工单。
func createDispatchTicketHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var t order.DispatchTicket
		if !httpx.BindAndValidate(c, &t, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(t.TicketNo, "ticketNo", 32),
				httpx.RequirePositiveID(t.OrderID, "orderId"),
			)
		}) {
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
	}
}