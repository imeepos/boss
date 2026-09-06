package adminapi

// 采购域编辑类 handler(P2-W2-T2):供应商编辑/启用、草稿编辑/详情、入库驳回。
// 审计口径:B/C/E 成功后必写(RecordAudit);A/D 为资料编辑与查询,不写审计。

import (
	"errors"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/procurement"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// procurementUpdateSupplier PUT /procurement/suppliers/:id(A):部分更新,编码不可改
// (入参 SupplierUpdate 无 code 字段,多余 JSON 键被忽略,结构性保证)。
func procurementUpdateSupplier(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var in procurement.SupplierUpdate
		if err := c.ShouldBindJSON(&in); err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Procurement.UpdateSupplier(c.Request.Context(), id, in); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// procurementEnableSupplier POST /procurement/suppliers/:id/enable(B):幂等启用,写审计。
func procurementEnableSupplier(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.Procurement.EnableSupplier(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "procurement_supplier", strconv.FormatInt(id, 10),
			map[string]any{"to": "ENABLED"})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// procurementUpdateOrder PUT /procurement/orders/:id(C):仅 DRAFT 可改(非 DRAFT 40900);
// 审计记变更前后键值(before/after 快照,前端与追溯两侧同构可读)。
func procurementUpdateOrder(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var in procurement.OrderDraftUpdate
		if err := c.ShouldBindJSON(&in); err != nil {
			respondErr(c, err)
			return
		}
		before, err := a.Procurement.GetOrderDetail(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Procurement.UpdateOrderDraft(c.Request.Context(), id, in); err != nil {
			respondErr(c, err)
			return
		}
		after, err := a.Procurement.GetOrderDetail(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "procurement_order", strconv.FormatInt(id, 10),
			map[string]any{"before": orderDraftSnapshot(before), "after": orderDraftSnapshot(after)})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// orderDraftSnapshot 审计键值快照:remark/期望日期/总金额/状态/明细行。
func orderDraftSnapshot(o *procurement.Order) map[string]any {
	items := make([]map[string]any, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, map[string]any{
			"materialCode": it.MaterialCode, "quantity": it.Quantity, "unitAmount": it.UnitAmount,
		})
	}
	return map[string]any{
		"remark": o.Remark, "expectedDate": o.ExpectedDate,
		"totalAmount": o.TotalAmount, "status": o.Status, "items": items,
	}
}

// procurementGetOrder GET /procurement/orders/:id(D):单头+明细行+状态;未命中 40400。
func procurementGetOrder(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		o, err := a.Procurement.GetOrderDetail(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"item": o})
	}
}

// procurementRejectReceipt POST /procurement/receipts/:id/reject(E):仅 DRAFT 可驳回
// (CONFIRMED 等 40900);原因限 255 字,空则缺省文案;驳回写审计(状态变更+原因)。
func procurementRejectReceipt(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var in struct {
			Reason string `json:"reason"`
		}
		// 允许空请求体(缺省文案);非 EOF 的绑定错误照常 422。
		if err := c.ShouldBindJSON(&in); err != nil && !errors.Is(err, io.EOF) {
			respondErr(c, err)
			return
		}
		if in.Reason == "" {
			in.Reason = "入库驳回"
		}
		if err := a.Procurement.RejectReceipt(c.Request.Context(), id, in.Reason); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "procurement_receipt", strconv.FormatInt(id, 10),
			map[string]any{"to": "REJECTED", "reason": in.Reason})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}
