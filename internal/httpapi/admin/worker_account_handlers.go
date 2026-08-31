package adminapi

// 师傅账号录入(worker.yaml POST /workers + PUT /workers/{workerId}/password)。
// 登录密码 bcrypt 落库(workers.password_hash),师傅端凭手机号+密码登录(worker/auth.yaml mode=password)。

import (
	"fmt"
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
// 负责区域:regionIds 多区域(000175,首位为主区域)或兼容单值 regionId,二者至少其一。
type workerCreateReq struct {
	StaffNo   string  `json:"staffNo"`
	Name      string  `json:"name"`
	GroupID   int64   `json:"groupId"`
	RegionID  int64   `json:"regionId"`
	RegionIDs []int64 `json:"regionIds"`
	Phone     string  `json:"phone"`
	Password  string  `json:"password"`
}

// workerRegionsOf 归一录入请求的负责区域:regionIds 优先,空则回退单值 regionId。
func workerRegionsOf(req workerCreateReq) []int64 {
	if len(req.RegionIDs) > 0 {
		return req.RegionIDs
	}
	if req.RegionID > 0 {
		return []int64{req.RegionID}
	}
	return nil
}

// workerCreateHandler POST /workers:后台录入师傅(主档 + 登录密码 + 负责区域),落 workers 主档。
func workerCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req workerCreateReq
		if !httpx.BindAndValidate(c, &req, func() error {
			regionIDs := workerRegionsOf(req)
			errs := []*httpx.ValidationError{
				httpx.RequireString(req.StaffNo, "staffNo", 32),
				httpx.RequireString(req.Name, "name", 64),
				httpx.RequireString(req.Phone, "phone", 32),
				httpx.RequirePositiveID(req.GroupID, "groupId"),
				httpx.RequireString(req.Password, "password", 128),
			}
			if len(regionIDs) == 0 {
				errs = append(errs, httpx.RequirePositiveID(0, "regionId"))
			}
			return httpx.CollectErrors(append(errs, regionIDListErrors(regionIDs)...)...)
		}) {
			return
		}
		regionIDs := workerRegionsOf(req)
		id, err := a.Worker.CreateWorkerWithPassword(c.Request.Context(), worker.Worker{
			StaffNo: strings.TrimSpace(req.StaffNo), Name: strings.TrimSpace(req.Name),
			GroupID: req.GroupID, RegionID: regionIDs[0], RegionIDs: regionIDs,
			Phone:  strings.TrimSpace(req.Phone),
			Status: 1, JoinedAt: time.Now(),
		}, req.Password)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker.create", "worker", strconv.FormatInt(id, 10),
			gin.H{"staffNo": req.StaffNo, "groupId": req.GroupID, "regionIds": workerRegionsOf(req)})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// workerSetRegionsReq 配置师傅负责区域请求体(regionIds 覆盖式,首位为主区域)。
type workerSetRegionsReq struct {
	RegionIDs []int64 `json:"regionIds"`
}

// workerSetRegionsHandler PUT /workers/{workerId}/regions:配置师傅负责区域(000175;
// 首位为主区域回写 workers.region_id,空集合=仅清空扩展区域主区域保留)。
func workerSetRegionsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, ok := httpx.ParsePathParamInt64(c, "workerId")
		if !ok {
			return
		}
		var req workerSetRegionsReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(regionIDListErrors(req.RegionIDs)...)
		}) {
			return
		}
		if err := a.Worker.SetWorkerRegions(c.Request.Context(), workerID, req.RegionIDs); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker.set-regions", "worker", c.Param("workerId"),
			gin.H{"regionIds": req.RegionIDs})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// regionIDListErrors 负责区域数组校验:元素须为正 ID(去重/归一由域层负责)。
func regionIDListErrors(regionIDs []int64) []*httpx.ValidationError {
	errs := make([]*httpx.ValidationError, 0, len(regionIDs))
	for i, id := range regionIDs {
		if id <= 0 {
			errs = append(errs, &httpx.ValidationError{
				Field: fmt.Sprintf("regionIds[%d]", i), Message: "must be a positive integer"})
		}
	}
	return errs
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
