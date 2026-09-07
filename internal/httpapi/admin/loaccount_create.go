package adminapi

// T2 存量开户导入配套建号接口:POST /lo-accounts(AAA 建号,menu:loaccount)。
// 契约:loid 全局唯一 40900;customer/offer/qos 必填(offer 须 PUBLISHED,qos 软引用存在);
// billing_mode 可选缺省 POSTPAID;法人缺省平台总公司、区域缺省 0(导入裁定,设计 §3)。
// 域侧校验在 aaa.LoAccountAdminService(PGStore 扩展),环节 6 既有 CreateLoAccount 不受影响。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// loAccountPayload POST /lo-accounts 请求体;status 固定 ACTIVE 不收。
type loAccountPayload struct {
	Loid            string `json:"loid"`
	CustomerID      int64  `json:"customerId"`
	OfferID         int64  `json:"offerId"`
	QosTemplateID   int64  `json:"qosTemplateId"`
	BillingMode     string `json:"billingMode"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	RegionID        int64  `json:"regionId"`
	RegionName      string `json:"regionName"`
	RegionPath      string `json:"regionPath"`
}

// loAccountCreateHandler POST /lo-accounts:AAA 建号(menu:loaccount)。
func loAccountCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := a.Aaa.(aaa.LoAccountAdminService)
		if !ok {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		var p loAccountPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			respondBadRequest(c, "invalid payload")
			return
		}
		if p.Loid == "" || p.CustomerID <= 0 || p.OfferID <= 0 || p.QosTemplateID <= 0 {
			respondBadRequest(c, "loid/customerId/offerId/qosTemplateId required")
			return
		}
		mode := p.BillingMode
		if mode == "" {
			mode = aaa.BillingModePostpaid
		}
		if mode != aaa.BillingModePrepaid && mode != aaa.BillingModePostpaid {
			respondBadRequest(c, "billingMode must be PREPAID or POSTPAID")
			return
		}
		id, err := svc.CreateLoAccountChecked(c.Request.Context(), aaa.LoAccount{
			Loid: p.Loid, CustomerID: p.CustomerID, OfferID: p.OfferID, QosTemplateID: p.QosTemplateID,
			LegalEntityID: p.LegalEntityID, LegalEntityName: p.LegalEntityName,
			RegionID: p.RegionID, RegionName: p.RegionName, RegionPath: p.RegionPath,
			Status: string(aaa.StatusActive), BillingMode: mode,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "新建", "lo_account", p.Loid, map[string]any{"customerId": p.CustomerID, "offerId": p.OfferID, "billingMode": mode})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "loid": p.Loid})
	}
}
