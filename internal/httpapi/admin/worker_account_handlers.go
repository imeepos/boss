package adminapi

// 师傅账号录入(worker.yaml POST /workers + PUT /workers/{workerId}/password)。
// 登录密码 bcrypt 落库(workers.password_hash),师傅端凭手机号+密码登录(worker/auth.yaml mode=password)。

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerCreateReq 新增师傅请求体(admin 录入,password 为师傅端登录密码)。
type workerCreateReq struct {
	StaffNo  string `json:"staffNo"`
	Name     string `json:"name"`
	GroupID  int64  `json:"groupId"`
	RegionID int64  `json:"regionId"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// workerCreateHandler POST /workers:后台录入师傅(主档 + 登录密码),落 workers 主档。
func workerCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req workerCreateReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.StaffNo, "staffNo", 32),
				httpx.RequireString(req.Name, "name", 64),
				httpx.RequireString(req.Phone, "phone", 32),
				httpx.RequirePositiveID(req.GroupID, "groupId"),
				httpx.RequirePositiveID(req.RegionID, "regionId"),
				httpx.RequireString(req.Password, "password", 128),
			)
		}) {
			return
		}
		id, err := a.Worker.CreateWorkerWithPassword(c.Request.Context(), worker.Worker{
			StaffNo: strings.TrimSpace(req.StaffNo), Name: strings.TrimSpace(req.Name),
			GroupID: req.GroupID, RegionID: req.RegionID, Phone: strings.TrimSpace(req.Phone),
			Status: 1, JoinedAt: time.Now(),
		}, req.Password)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker.create", "worker", strconv.FormatInt(id, 10),
			gin.H{"staffNo": req.StaffNo, "groupId": req.GroupID, "regionId": req.RegionID})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// workerSetPasswordReq 重置师傅登录密码请求体。
type workerSetPasswordReq struct {
	Password string `json:"password"`
}

// workerSetPasswordHandler PUT /workers/{workerId}/password:重置师傅登录密码。
func workerSetPasswordHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, ok := httpx.ParsePathParamInt64(c, "workerId")
		if !ok {
			return
		}
		var req workerSetPasswordReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(httpx.RequireString(req.Password, "password", 128))
		}) {
			return
		}
		if err := a.Worker.SetPassword(c.Request.Context(), workerID, req.Password); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker.set-password", "worker", c.Param("workerId"), nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
