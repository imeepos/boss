package workerapi

// 师傅自助注册:公开端点(师傅端尚未登录,对标客户自助建档)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

// workerRegistrationReq 师傅自助注册请求体(admin 审核队列在 adminapi/worker_onboarding.go)。
// 班组/区域允许为 0,留待 admin 审核时补齐/纠正(adopted docs/notes/adopted/2026-XX-XX-worker-self-registration-loose-foreign-key.md)。
type workerRegistrationReq struct {
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	IDCardNo string `json:"idCardNo" binding:"required"`
	GroupID  int64  `json:"groupId"`
	RegionID int64  `json:"regionId"`
}

// requireNonNegativeID 校验 ID ≥ 0;师傅自助注册允许 groupId/regionId=0 由后台补正。
func requireNonNegativeID(value int64, field string) *httpx.ValidationError {
	if value < 0 {
		return &httpx.ValidationError{Field: field, Message: "must not be negative"}
	}
	return nil
}

// registerWorkerSelfRegistration 师傅自助注册路由。
func registerWorkerSelfRegistration(pub *gin.RouterGroup, a *app.Application) {
	pub.POST("/worker-registrations", workerSelfRegister(a))
}
