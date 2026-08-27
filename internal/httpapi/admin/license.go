// 系统授权页路由:状态查询 + 激活码兑换。挂在 authed(豁免授权门禁),
// 是"没证书也能进"的激活入口(adopted license-gate note)。
package adminapi

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/license"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerLicenseRoutes 注册 /license/* 授权路由;License 为 nil 时返回未配置。
func registerLicenseRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/license/status", licenseStatusHandler(a.License))
	g.POST("/license/activate", licenseActivateHandler(a.License))
}

// licenseStatusHandler GET /license/status:当前授权状态。
// code=0 返回 status(data);授权未激活/失效仍返回 200 envelope(前端据此跳激活页)。
func licenseStatusHandler(svc *license.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			respond(c, apitypes.CodeOK, gin.H{"activated": false, "enabled": false})
			return
		}
		st, err := svc.Check(c.Request.Context())
		if errors.Is(err, license.ErrNoLicense) {
			respond(c, apitypes.CodeOK, gin.H{"activated": false, "enabled": true})
			return
		}
		if err != nil {
			// 证书失效仍可查(前端展示原因并允许重新激活);不暴露签名细节。
			respond(c, apitypes.CodeOK, gin.H{"activated": false, "enabled": true, "reason": err.Error()})
			return
		}
		respond(c, apitypes.CodeOK, st)
	}
}

// activateBody POST /license/activate 请求体(激活码)。
type activateBody struct {
	ActivationCode string `json:"activationCode"`
}

// licenseActivateHandler POST /license/activate:兑码→本地落盘→返回授权状态。
func licenseActivateHandler(svc *license.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			respondErr(c, errors.New("license gate not enabled"))
			return
		}
		var body activateBody
		if err := c.ShouldBindJSON(&body); err != nil || body.ActivationCode == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"message": "activationCode required"})
			return
		}
		st, err := svc.Activate(c.Request.Context(), body.ActivationCode)
		if err != nil {
			// 兑码/取令牌失败属下游(release-platform)错误,映射 502;留可 grep 日志。
			respond(c, apitypes.CodeDownstreamErr, gin.H{"message": "activation failed"})
			return
		}
		respond(c, apitypes.CodeOK, st)
	}
}