package workerapi

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
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

// registerWorkerSelfRegistration 师傅自助注册:公开端点(师傅端尚未登录,对标客户自助建档)。
func registerWorkerSelfRegistration(pub *gin.RouterGroup, a *app.Application) {
	pub.POST("/worker-registrations", func(c *gin.Context) {
		var req workerRegistrationReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Name, "name", 64),
				httpx.RequirePhone(req.Phone, "phone"),
				httpx.RequireString(req.IDCardNo, "idCardNo", 32),
				requireNonNegativeID(req.GroupID, "groupId"),
				requireNonNegativeID(req.RegionID, "regionId"),
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
		// 后台待办提醒:师傅注册待审核(docs/plan/admin-notify-center.md §4)。
		if a.Notify != nil {
			_ = a.Notify.Emit(c.Request.Context(), notify.Input{
				Category: notify.CategoryTodo, Level: notify.LevelWarn,
				Title: "师傅注册待审核:" + req.Name, RefType: "worker_reg",
				RefID: strconv.FormatInt(id, 10), Link: "/boss/worker-reg",
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": worker.RegStatusPending})
	})
}
