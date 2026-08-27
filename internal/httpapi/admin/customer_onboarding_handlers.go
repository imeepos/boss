package adminapi

// 客户注册/审核/实名核验 路由具名 handler(承接 registerCustomerOnboardingRoutes 扁平路由表)。
// 请求体类型 workerReviewReq / workerRealNameReq / workerRealNameVerifyReq 见 worker_onboarding.go。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// customerRegistrationListHandler GET /customer-registrations:审核队列(按状态过滤,空=全部)。
func customerRegistrationListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.CustomerOnboarding.ListRegistrations(c.Request.Context(), c.Query("status"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// customerRegistrationApproveHandler POST /customer-registrations/{id}/approve:审核通过,建 customers 主档 + 回填。
func customerRegistrationApproveHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		customerID, err := a.CustomerOnboarding.Approve(c.Request.Context(), id, claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "customer_registration.approve", "customer_registration", c.Param("id"),
			gin.H{"customerId": customerID})
		respond(c, apitypes.CodeOK, gin.H{"customerId": customerID, "status": customer.RegStatusApproved})
	}
}

// customerRegistrationRejectHandler POST /customer-registrations/{id}/reject:审核驳回,记审核意见。
func customerRegistrationRejectHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req workerReviewReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.CustomerOnboarding.Reject(c.Request.Context(), id, claims.AccountID, req.Note); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "customer_registration.reject", "customer_registration", c.Param("id"),
			gin.H{"note": req.Note})
		respond(c, apitypes.CodeOK, gin.H{"status": customer.RegStatusRejected})
	}
}

// customerSubmitRealNameHandler POST /customers/{id}/real-name:提交实名核验(自动通道 + 人工兜底)。
func customerSubmitRealNameHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req workerRealNameReq
		if !bindRealNameReq(c, &req) {
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
		// 阿里云二要素自动核验;通道未配置时保持 PENDING 走下方人工核验端点。
		result := a.AutoVerifyRealName(c.Request.Context(), customerID, req.RealName, req.IDCardNo)
		if result == customer.RealNamePending {
			emitRealnamePendingTodo(a, c, "customer", customerID, req.RealName)
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id, "result": result})
	}
}

// customerGetRealNameHandler GET /customers/{id}/real-name:取最近一次实名核验记录。
func customerGetRealNameHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		v, err := a.CustomerRealName.GetLatest(c.Request.Context(), customerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, v)
	}
}

// customerVerifyRealNameHandler POST /customers/{id}/real-name/verify:后台核验 PASS/FAIL。
func customerVerifyRealNameHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req workerRealNameVerifyReq
		if !httpx.BindAndValidate(c, &req, func() error {
			if req.Result != customer.RealNamePass && req.Result != customer.RealNameFail {
				return &httpx.ValidationError{Field: "result", Message: "must be PASS or FAIL"}
			}
			return nil
		}) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.CustomerRealName.Verify(c.Request.Context(), customerID, req.Result, req.Reason, claims.Username, claims.AccountID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "customer_realname.verify", "customer_realname", c.Param("id"),
			gin.H{"result": req.Result, "reason": req.Reason})
		resolveRealnameTodo(a, c, "customer", customerID)
		notifyCustomerRealnameResult(a, c, customerID, req.Result, req.Reason)
		respond(c, apitypes.CodeOK, gin.H{"result": req.Result})
	}
}
