package userapi

// 推送设备上报(用户端):App 启动/登录后上报 JPush RegistrationID(契约 user/misc.yaml)。

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	pushdomain "github.com/ymm-001/boss/internal/domain/push"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPortalPushDeviceRoutes 推送设备路由(uauth 组,客户鉴权)。
func registerPortalPushDeviceRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/push/device", func(c *gin.Context) {
		customerID, ok := requireCustomer(c)
		if !ok {
			return
		}
		var req struct {
			RegistrationID string `json:"registrationId" binding:"required"`
			Vendor         string `json:"vendor"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		if a.PushDevices == nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		if err := a.PushDevices.RegisterDevice(c.Request.Context(),
			pushdomain.SubjectUser, customerID, req.RegistrationID, req.Vendor); err != nil {
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
