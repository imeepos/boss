package workerapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
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
		if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Phone == "" ||
			req.IDCardNo == "" || req.GroupID <= 0 || req.RegionID <= 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
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
