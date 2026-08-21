package workerapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerRegistrationReq 师傅自助注册请求体(admin 审核队列在 adminapi/worker_onboarding.go)。
type workerRegistrationReq struct {
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	IDCardNo string `json:"idCardNo" binding:"required"`
	GroupID  int64  `json:"groupId" binding:"required"`
	RegionID int64  `json:"regionId" binding:"required"`
}

// registerWorkerSelfRegistration 师傅自助注册:公开端点(师傅端尚未登录,对标客户自助建档)。
func registerWorkerSelfRegistration(pub *gin.RouterGroup, a *app.Application) {
	pub.POST("/worker-registrations", func(c *gin.Context) {
		var req workerRegistrationReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Name, "name", 64),
				httpx.RequirePhone(req.Phone, "phone"),
				httpx.RequireString(req.IDCardNo, "idCardNo", 32),
				httpx.RequirePositiveID(req.GroupID, "groupId"),
				httpx.RequirePositiveID(req.RegionID, "regionId"),
			)
		}) {
			return
		}
		id, err := a.WorkerOnboarding.Submit(c.Request.Context(), worker.Registration{
			Name: req.Name, Phone: req.Phone, IDCardNo: req.IDCardNo,
			GroupID: req.GroupID, RegionID: req.RegionID,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": worker.RegStatusPending})
	})
}
