package workerapi

// 师傅自助注册 handler 实现(self_registration.go 仅留路由表与请求体)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func workerSelfRegister(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		id, err := workerSubmitRegistration(c, a, req)
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
	}
}

func workerSubmitRegistration(c *gin.Context, a *app.Application, req workerRegistrationReq) (int64, error) {
	return a.WorkerOnboarding.Submit(c.Request.Context(), worker.Registration{
		Name: req.Name, Phone: req.Phone, IDCardNo: req.IDCardNo,
		GroupID: req.GroupID, RegionID: req.RegionID,
	})
}
