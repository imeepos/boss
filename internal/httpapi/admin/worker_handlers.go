package adminapi

// W 师傅域路由 handler 实现(承接 registerWorkerRoutes 的扁平路由表)。
// 班组 / 师傅主档 / 接单设置。

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerListGroupsHandler GET /worker-groups:班组列表。
func workerListGroupsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Worker.ListGroups(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerCreateGroupHandler POST /worker-groups:新建班组(承接 admin 后台基础数据维护;code 公司内唯一)。
func workerCreateGroupHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req workerGroupCreateReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Code, "code", 32),
				httpx.RequireString(req.Name, "name", 64),
				httpx.RequirePositiveID(req.LegalEntityID, "legalEntityId"),
			)
		}) {
			return
		}
		id, err := a.Worker.CreateGroup(c.Request.Context(), worker.Group{
			LegalEntityID: req.LegalEntityID,
			Code:          req.Code,
			Name:          req.Name,
			LeaderID:      req.LeaderID,
			LeaderName:    req.LeaderName,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "worker_group.create", "worker_group", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// workerListWorkersHandler GET /workers:师傅列表(按 groupId/keyword 过滤)。
func workerListWorkersHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Worker.ListWorkers(c.Request.Context(), queryInt64(c, "groupId"), c.Query("keyword"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// workerGetWorkerHandler GET /workers/{workerId}:师傅详情(worker.yaml GET /workers/{workerId})。
func workerGetWorkerHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, ok := httpx.ParsePathParamInt64(c, "workerId")
		if !ok {
			return
		}
		w, err := a.Worker.GetWorker(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, w)
	}
}

// workerUpdateSettingsHandler PUT /workers/{workerId}/settings:修改接单设置(在线/半径/接单类型),师傅1:1 即时生效(worker.yaml /workers/{id}/settings)。
func workerUpdateSettingsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, ok := httpx.ParsePathParamInt64(c, "workerId")
		if !ok {
			return
		}
		if _, err := a.Worker.GetWorker(c.Request.Context(), workerID); err != nil {
			respondErr(c, err)
			return
		}
		var req workerSettingsReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if _, err := a.WorkerLedger.UpsertSettings(c.Request.Context(), worker.Settings{
			WorkerID: workerID, Accepting: req.Online,
			RadiusKm: req.RadiusKm, AcceptTypes: strings.Join(req.AcceptTypes, ","),
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
