package adminapi

// 产品/套餐 ↔ 下发模板绑定 handler(方案B)。
// 挂 /products/{id}/provision-binding:读 menu:product、写 menu:product-write(见 customer.go 路由表)。
// 绑定缺失时 GetOfferBinding 返回空绑定,前端展示"未绑定"。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// productGetProvisionBindingHandler GET /products/{id}/provision-binding:查询套餐已绑模板。
func productGetProvisionBindingHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		b, err := a.Provision.GetOfferBinding(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		if b == nil {
			respond(c, apitypes.CodeOK, gin.H{"offerId": id, "templateId": 0})
			return
		}
		respond(c, apitypes.CodeOK, b)
	}
}

// provisionListOfferBindingsHandler GET /provision-bindings:全部套餐绑定(产品页列状态)。
func provisionListOfferBindingsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Provision.ListOfferBindings(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// upsertProvisionBindingReq 绑定请求体。
type upsertProvisionBindingReq struct {
	TemplateID int64  `json:"templateId"`
	Remark     string `json:"remark"`
}

// productUpsertProvisionBindingHandler PUT /products/{id}/provision-binding:绑定/改绑套餐→模板。
func productUpsertProvisionBindingHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req upsertProvisionBindingReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(httpx.RequirePositiveID(req.TemplateID, "templateId"))
		}) {
			return
		}
		bid, err := a.Provision.UpsertOfferBinding(c.Request.Context(), id, req.TemplateID, req.Remark)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "offer_provision_binding", fmt.Sprint(id),
			map[string]any{"templateId": req.TemplateID})
		respond(c, apitypes.CodeOK, gin.H{"id": bid})
	}
}

// productDeleteProvisionBindingHandler DELETE /products/{id}/provision-binding:解绑(幂等)。
func productDeleteProvisionBindingHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.Provision.DeleteOfferBinding(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据删除", "offer_provision_binding", fmt.Sprint(id), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
