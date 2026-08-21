package adminapi

// W 师傅注册 / 审核 / 实名认证 子域路由注册(迁移 000050)。
// 注册申请为公开端点,已在 RegisterRoutes 的 api 组注册;此处为审核队列 + 实名核验(均走 menu:dispatch)。
// 全部 handler 实现见 worker_onboarding_handlers.go;此处只保留扁平路由表 + 请求体类型。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerWorkerOnboardingRoutes 注册师傅注册 / 审核 / 实名认证 子域路由(迁移 000050)。
func registerWorkerOnboardingRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/worker-registrations", requirePerm(a.User, "menu:dispatch"), workerListRegistrationsHandler(a))
	g.POST("/worker-registrations/:id/approve", requirePerm(a.User, "menu:dispatch"), workerApproveRegistrationHandler(a))
	g.POST("/worker-registrations/:id/reject", requirePerm(a.User, "menu:dispatch"), workerRejectRegistrationHandler(a))

	g.POST("/workers/:workerId/real-name", requirePerm(a.User, "menu:dispatch"), workerSubmitRealNameHandler(a))
	g.GET("/workers/:workerId/real-name", requirePerm(a.User, "menu:dispatch"), workerGetRealNameHandler(a))
	g.POST("/workers/:workerId/real-name/verify", requirePerm(a.User, "menu:dispatch"), workerVerifyRealNameHandler(a))
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

// workerApproveReq 审核通过请求体(groupId/regionId 由审核员显式指定/纠正师傅登记空值)。
type workerApproveReq struct {
	GroupID  int64 `json:"groupId" binding:"required"`
	RegionID int64 `json:"regionId" binding:"required"`
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
