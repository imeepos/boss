package workerapi

// 推送设备上报(师傅端):App 启动/登录后上报 JPush RegistrationID(契约 worker/misc.yaml)。

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	pushdomain "github.com/ymm-001/boss/internal/domain/push"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerPushDeviceRoutes 推送设备路由(wauth 组,师傅鉴权)。
func registerWorkerPushDeviceRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/push/device", func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		var req struct {
			RegistrationID string `json:"registrationId" binding:"required"`
			Vendor         string `json:"vendor"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if a.PushDevices == nil || workerID <= 0 {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		if err := a.PushDevices.RegisterDevice(c.Request.Context(),
			pushdomain.SubjectWorker, workerID, req.RegistrationID, req.Vendor); err != nil {
			if errors.Is(err, pushdomain.ErrInvalidDevice) {
				respond(c, apitypes.CodeInvalidParam, nil)
				return
			}
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}
