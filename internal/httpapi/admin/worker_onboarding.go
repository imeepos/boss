package adminapi

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

// registerWorkerOnboardingRoutes 注册师傅注册 / 审核 / 实名认证 子域路由(迁移 000050)。
// 注册申请为公开端点,已在 RegisterRoutes 的 api 组注册;此处为审核队列 + 实名核验(均走 menu:dispatch)。
func registerWorkerOnboardingRoutes(g *gin.RouterGroup, a *app.Application) {
	// 审核队列:按状态列出(空=全部)。
	g.GET("/worker-registrations", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		list, err := a.WorkerOnboarding.ListRegistrations(c.Request.Context(), c.Query("status"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 审核通过:建 workers 主档 + 回填。
	g.POST("/worker-registrations/:id/approve", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		workerID, err := a.WorkerOnboarding.Approve(c.Request.Context(), id, claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_registration.approve", "worker_registration", c.Param("id"),
			gin.H{"workerId": workerID})
		respond(c, apitypes.CodeOK, gin.H{"workerId": workerID, "status": worker.RegStatusApproved})
	})

	// 审核驳回:记审核意见。
	g.POST("/worker-registrations/:id/reject", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req workerReviewReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.WorkerOnboarding.Reject(c.Request.Context(), id, claims.AccountID, req.Note); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_registration.reject", "worker_registration", c.Param("id"),
			gin.H{"note": req.Note})
		respond(c, apitypes.CodeOK, gin.H{"status": worker.RegStatusRejected})
	})

	// 师傅实名核验相关(worker 主体 1:1)。
	g.POST("/workers/:workerId/real-name", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		workerID, _ := strconv.ParseInt(c.Param("workerId"), 10, 64)
		var req workerRealNameReq
		if err := c.ShouldBindJSON(&req); err != nil || req.RealName == "" || req.IDCardNo == "" || req.Method == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
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
		respond(c, apitypes.CodeOK, gin.H{"id": id, "result": worker.RealNamePending})
	})

	g.GET("/workers/:workerId/real-name", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		workerID, _ := strconv.ParseInt(c.Param("workerId"), 10, 64)
		v, err := a.WorkerRealName.GetLatest(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, v)
	})

	// 后台核验:PASS / FAIL。
	g.POST("/workers/:workerId/real-name/verify", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		workerID, _ := strconv.ParseInt(c.Param("workerId"), 10, 64)
		var req workerRealNameVerifyReq
		if err := c.ShouldBindJSON(&req); err != nil ||
			(req.Result != worker.RealNamePass && req.Result != worker.RealNameFail) {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.WorkerRealName.Verify(c.Request.Context(), workerID, req.Result, claims.Username, claims.AccountID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_realname.verify", "worker_realname", c.Param("workerId"),
			gin.H{"result": req.Result})
		respond(c, apitypes.CodeOK, gin.H{"result": req.Result})
	})
}

// workerGroupCreateReq 新建班组请求体(legalEntityId+code+name 必填)。
type workerGroupCreateReq struct {
	LegalEntityID int64  `json:"legalEntityId" binding:"required"`
	Code          string `json:"code" binding:"required"`
	Name          string `json:"name" binding:"required"`
	LeaderID      int64  `json:"leaderId"`
	LeaderName    string `json:"leaderName"`
}

// workerRegistrationReq 师傅注册申请请求体。

// workerReviewReq 审核驳回请求体(note 选填,建议必填)。
type workerReviewReq struct {
	Note string `json:"note"`
}

// workerRealNameReq 师傅实名核验提交请求体。
type workerRealNameReq struct {
	RealName string `json:"realName" binding:"required"`
	IDCardNo string `json:"idCardNo" binding:"required"`
	Method   string `json:"method" binding:"required"`
}

// workerRealNameVerifyReq 实名核验请求体;reason 仅客户域 FAIL 时落 verifications.reject_reason。
type workerRealNameVerifyReq struct {
	Result string `json:"result" binding:"required"` // PASS / FAIL
	Reason string `json:"reason"`                    // 驳回原因(FAIL 时建议必填)
}
