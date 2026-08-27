package adminapi

// W 师傅注册 / 审核 / 实名认证子域路由 handler 实现(承接 registerWorkerOnboardingRoutes 的扁平路由表)。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerListRegistrationsHandler GET /worker-registrations:审核队列(按状态列出,空=全部)。
func workerListRegistrationsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkerOnboarding.ListRegistrations(c.Request.Context(), c.Query("status"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerApproveRegistrationHandler POST /worker-registrations/{id}/approve:审核通过,建 workers 主档 + 回填。
func workerApproveRegistrationHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req workerApproveReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.GroupID, "groupId"),
				httpx.RequirePositiveID(req.RegionID, "regionId"),
			)
		}) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		workerID, err := a.WorkerOnboarding.Approve(c.Request.Context(), id, claims.AccountID, req.GroupID, req.RegionID)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_registration.approve", "worker_registration", c.Param("id"),
			gin.H{"workerId": workerID, "groupId": req.GroupID, "regionId": req.RegionID})
		resolveTodo(c.Request.Context(), a, refWorkerReg, strconv.FormatInt(id, 10))
		respond(c, apitypes.CodeOK, gin.H{"workerId": workerID, "status": worker.RegStatusApproved})
	}
}

// workerRejectRegistrationHandler POST /worker-registrations/{id}/reject:审核驳回,记审核意见。
func workerRejectRegistrationHandler(a *app.Application) gin.HandlerFunc {
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
		if err := a.WorkerOnboarding.Reject(c.Request.Context(), id, claims.AccountID, req.Note); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_registration.reject", "worker_registration", c.Param("id"),
			gin.H{"note": req.Note})
		resolveTodo(c.Request.Context(), a, refWorkerReg, strconv.FormatInt(id, 10))
		respond(c, apitypes.CodeOK, gin.H{"status": worker.RegStatusRejected})
	}
}

// workerSubmitRealNameHandler POST /workers/{workerId}/real-name:师傅实名核验提交(worker 主体 1:1)。
func workerSubmitRealNameHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, ok := httpx.ParsePathParamInt64(c, "workerId")
		if !ok {
			return
		}
		var req workerRealNameReq
		if !bindRealNameReq(c, &req) {
			return
		}
		id, err := a.WorkerRealName.SubmitRealName(c.Request.Context(), worker.WorkerRealNameVerification{
			WorkerID:   workerID,
			Method:     req.Method,
			RealName:   req.RealName,
			IDCardNo:   req.IDCardNo,
			Result:     worker.RealNamePending,
			VerifiedAt: time.Now(),
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		emitRealnamePendingTodo(a, c, "worker", workerID, req.RealName)
		respond(c, apitypes.CodeOK, gin.H{"id": id, "result": worker.RealNamePending})
	}
}

// workerGetRealNameHandler GET /workers/{workerId}/real-name:获取最新实名核验记录。
func workerGetRealNameHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, ok := httpx.ParsePathParamInt64(c, "workerId")
		if !ok {
			return
		}
		v, err := a.WorkerRealName.GetLatest(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, v)
	}
}

// workerVerifyRealNameHandler POST /workers/{workerId}/real-name/verify:后台核验 PASS / FAIL。
func workerVerifyRealNameHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, ok := httpx.ParsePathParamInt64(c, "workerId")
		if !ok {
			return
		}
		var req workerRealNameVerifyReq
		if !httpx.BindAndValidate(c, &req, func() error {
			if req.Result != worker.RealNamePass && req.Result != worker.RealNameFail {
				return &httpx.ValidationError{Field: "result", Message: "must be PASS or FAIL"}
			}
			return nil
		}) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.WorkerRealName.Verify(c.Request.Context(), workerID, req.Result, claims.Username, claims.AccountID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_realname.verify", "worker_realname", c.Param("workerId"),
			gin.H{"result": req.Result})
		resolveRealnameTodo(a, c, "worker", workerID)
		notifyWorkerRealnameResult(a, c, workerID, req.Result)
		respond(c, apitypes.CodeOK, gin.H{"result": req.Result})
	}
}

// bindRealNameReq 实名提交共用绑定校验(realName/idCardNo/method);失败已回写响应。
func bindRealNameReq(c *gin.Context, req *workerRealNameReq) bool {
	return httpx.BindAndValidate(c, req, func() error {
		return httpx.CollectErrors(
			httpx.RequireString(req.RealName, "realName", 64),
			httpx.RequireString(req.IDCardNo, "idCardNo", 32),
			httpx.RequireString(req.Method, "method", 32),
		)
	})
}
