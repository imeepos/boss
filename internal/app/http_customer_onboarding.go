package app

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerCustomerOnboardingRoutes 注册客户注册 / 审核 / 实名认证 子域路由(迁移 000051)。
// 注册申请为公开端点,已在 RegisterRoutes 的 api 组注册;此处为审核队列 + 实名核验(均走 menu:customer)。
func registerCustomerOnboardingRoutes(g *gin.RouterGroup, a *Application) {
	// 审核队列:按状态列出(空=全部)。
	g.GET("/customer-registrations", requirePerm(a.User, "menu:customer"), func(c *gin.Context) {
		list, err := a.CustomerOnboarding.ListRegistrations(c.Request.Context(), c.Query("status"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 审核通过:建 customers 主档 + 回填。
	g.POST("/customer-registrations/:id/approve", requirePerm(a.User, "menu:customer"), func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		customerID, err := a.CustomerOnboarding.Approve(c.Request.Context(), id, claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "customer_registration.approve", "customer_registration", c.Param("id"),
			gin.H{"customerId": customerID})
		respond(c, apitypes.CodeOK, gin.H{"customerId": customerID, "status": customer.RegStatusApproved})
	})

	// 审核驳回:记审核意见。
	g.POST("/customer-registrations/:id/reject", requirePerm(a.User, "menu:customer"), func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req workerReviewReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.CustomerOnboarding.Reject(c.Request.Context(), id, claims.AccountID, req.Note); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "customer_registration.reject", "customer_registration", c.Param("id"),
			gin.H{"note": req.Note})
		respond(c, apitypes.CodeOK, gin.H{"status": customer.RegStatusRejected})
	})

	// 客户实名核验相关(customer 主体 1:1)。
	g.POST("/customers/:id/real-name", requirePerm(a.User, "menu:customer"), func(c *gin.Context) {
		customerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req workerRealNameReq
		if err := c.ShouldBindJSON(&req); err != nil || req.RealName == "" || req.IDCardNo == "" || req.Method == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.CustomerRealName.SubmitRealName(c.Request.Context(), customer.CustomerRealNameVerification{
			CustomerID: customerID,
			Method:     req.Method,
			RealName:   req.RealName,
			IDCardNo:   req.IDCardNo,
			Result:     customer.RealNamePending,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id, "result": customer.RealNamePending})
	})

	g.GET("/customers/:id/real-name", requirePerm(a.User, "menu:customer"), func(c *gin.Context) {
		customerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		v, err := a.CustomerRealName.GetLatest(c.Request.Context(), customerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, v)
	})

	// 后台核验:PASS / FAIL。
	g.POST("/customers/:id/real-name/verify", requirePerm(a.User, "menu:customer"), func(c *gin.Context) {
		customerID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req workerRealNameVerifyReq
		if err := c.ShouldBindJSON(&req); err != nil ||
			(req.Result != customer.RealNamePass && req.Result != customer.RealNameFail) {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.CustomerRealName.Verify(c.Request.Context(), customerID, req.Result, claims.Username, claims.AccountID); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "customer_realname.verify", "customer_realname", c.Param("id"),
			gin.H{"result": req.Result})
		respond(c, apitypes.CodeOK, gin.H{"result": req.Result})
	})
}

// customerRegistrationReq 客户注册申请请求体。
type customerRegistrationReq struct {
	Name          string `json:"name" binding:"required"`
	Phone         string `json:"phone" binding:"required"`
	IDCardNo      string `json:"idCardNo" binding:"required"`
	LegalEntityID int64  `json:"legalEntityId" binding:"required"`
	AddressID     int64  `json:"addressId" binding:"required"`
	RegionID      int64  `json:"regionId" binding:"required"`
}