package userapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// customerRegistrationReq 客户自助注册请求体(admin 审核队列在 adminapi/customer_onboarding.go)。
type customerRegistrationReq struct {
	Name          string `json:"name" binding:"required"`
	Phone         string `json:"phone" binding:"required"`
	IDCardNo      string `json:"idCardNo" binding:"required"`
	LegalEntityID int64  `json:"legalEntityId" binding:"required"`
	AddressID     int64  `json:"addressId" binding:"required"`
	RegionID      int64  `json:"regionId" binding:"required"`
	Source        string `json:"source"` // 注册来源(获客追踪,可空,≤32)
}

// registerCustomerSelfRegistration 客户自助注册:公开端点(审核前不入 customers 主档)。
func registerCustomerSelfRegistration(pub *gin.RouterGroup, a *app.Application) {
	pub.POST("/customer-registrations", func(c *gin.Context) {
		var req customerRegistrationReq
		if !httpx.BindAndValidate(c, &req, func() error {
			var sourceErr *httpx.ValidationError
			if len(req.Source) > 32 {
				sourceErr = &httpx.ValidationError{Field: "source", Message: "source 过长(≤32)"}
			}
			return httpx.CollectErrors(
				httpx.RequireString(req.Name, "name", 64),
				httpx.RequirePhone(req.Phone, "phone"),
				httpx.RequireString(req.IDCardNo, "idCardNo", 32),
				httpx.RequirePositiveID(req.LegalEntityID, "legalEntityId"),
				httpx.RequirePositiveID(req.AddressID, "addressId"),
				httpx.RequirePositiveID(req.RegionID, "regionId"),
				sourceErr,
			)
		}) {
			return
		}
		id, err := a.CustomerOnboarding.Submit(c.Request.Context(), customer.Registration{
			Name: req.Name, Phone: req.Phone, IDCardNo: req.IDCardNo,
			LegalEntityID: req.LegalEntityID, AddressID: req.AddressID, RegionID: req.RegionID,
			Source: req.Source,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": customer.RegStatusPending})
	})
}
