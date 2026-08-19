package userapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
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
}

// registerCustomerSelfRegistration 客户自助注册:公开端点(审核前不入 customers 主档)。
func registerCustomerSelfRegistration(pub *gin.RouterGroup, a *app.Application) {
	pub.POST("/customer-registrations", func(c *gin.Context) {
		var req customerRegistrationReq
		if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Phone == "" ||
			req.IDCardNo == "" || req.LegalEntityID <= 0 || req.AddressID <= 0 || req.RegionID <= 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.CustomerOnboarding.Submit(c.Request.Context(), customer.Registration{
			Name: req.Name, Phone: req.Phone, IDCardNo: req.IDCardNo,
			LegalEntityID: req.LegalEntityID, AddressID: req.AddressID, RegionID: req.RegionID,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": customer.RegStatusPending})
	})
}
