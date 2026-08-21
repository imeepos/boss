package adminapi

// 网络资源域 handler 实现(从 resource.go 抽出,registerResourceRoutes 只剩扁平路由表)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// listResourcesHandler GET /resources:资源列表。
func listResourcesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Resource.ListResources(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// listPortsHandler GET /ports:端口列表(按 resourceId 过滤)。
func listPortsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Resource.ListPorts(c.Request.Context(), queryInt64(c, "resourceId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// listPortHistoryHandler GET /ports/:portId/change-history:端口变更历史。
func listPortHistoryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "portId")
		if !ok {
			return
		}
		list, err := a.ResourceSub.ListPortHistory(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// releaseReserveHandler POST /reserves/:reserveId/release:释放预留。
func releaseReserveHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "reserveId")
		if !ok {
			return
		}
		if err := a.ResourceSub.ReleaseReserve(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "reserve", c.Param("reserveId"), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

// listReservesHandler GET /reserves:预留记录列表。
func listReservesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.ResourceSub.ListReserveRecords(c.Request.Context(), queryInt64(c, "portId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// createTransferHandler POST /transfers:资源调拨。
func createTransferHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var t resource.Transfer
		if !httpx.BindAndValidate(c, &t, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(t.ResourceID, "resourceId"),
				httpx.RequirePositiveID(t.FromRegionID, "fromRegionId"),
				httpx.RequirePositiveID(t.ToRegionID, "toRegionId"),
			)
		}) {
			return
		}
		if t.TransferNo == "" {
			t.TransferNo = genNo("TRF")
		}
		if t.Status == "" {
			t.Status = "PENDING"
		}
		id, err := a.ResourceSub.CreateTransfer(c.Request.Context(), t)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "transfer", t.TransferNo, map[string]any{"resourceId": t.ResourceID, "toRegionId": t.ToRegionID})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "transferNo": t.TransferNo})
	}
}

// approveTransferHandler POST /transfers/:transferNo/approve:调拨审批通过。
func approveTransferHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		no := c.Param("transferNo")
		if err := a.ResourceSub.ApproveTransfer(c.Request.Context(), no); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "transfer", no, map[string]any{"result": "approved"})
		respond(c, apitypes.CodeOK, nil)
	}
}

// rejectTransferHandler POST /transfers/:transferNo/reject:调拨审批驳回。
func rejectTransferHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		no := c.Param("transferNo")
		if err := a.ResourceSub.RejectTransfer(c.Request.Context(), no); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "transfer", no, map[string]any{"result": "rejected"})
		respond(c, apitypes.CodeOK, nil)
	}
}

// listTransfersHandler GET /transfers:调拨列表。
func listTransfersHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.ResourceSub.ListTransfers(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// listExpansionsHandler GET /expansions:扩容列表。
func listExpansionsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.ResourceSub.ListExpansions(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// createExpansionHandler POST /expansions:扩容申请。
func createExpansionHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var e resource.Expansion
		if !httpx.BindAndValidate(c, &e, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(e.LegalEntityID, "legalEntityId"),
				httpx.RequirePositiveID(e.RegionID, "regionId"),
			)
		}) {
			return
		}
		if e.ExpansionNo == "" {
			e.ExpansionNo = genNo("EXP")
		}
		if e.Status == "" {
			e.Status = "PENDING"
		}
		id, err := a.ResourceSub.CreateExpansion(c.Request.Context(), e)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "expansion", e.ExpansionNo, map[string]any{"regionId": e.RegionID, "expectedPorts": e.ExpectedPorts})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "expansionNo": e.ExpansionNo})
	}
}

// listQosTemplatesHandler GET /expansions/qos-templates:QoS 模板列表。
func listQosTemplatesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.ResourceAssign.ListQosTemplates(c.Request.Context(), queryInt64(c, "legalEntityId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}