package workerapi

// 开发模式端点:回显最近一条未过期未消费的验证码(对接 Android 开发模式开关)。
// 注册条件:Server.DevMode=true(由 BOSS_DEV_MODE=true 注入);生产严禁开启,
// 路径只读不写,泄露 phone 维度的验证码等于绕开短信通道做登录,等同关闭凭据。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// devSmsCodeReq 开发模式查询验证码请求体。scene=login(默认)|register(入驻页回填)。
type devSmsCodeReq struct {
	Phone string `json:"phone" binding:"required"`
	Scene string `json:"scene"`
}

// devSmsCodeHandler 查询某 phone+scene 最近一条未过期未消费的验证码。
// scene 白名单 login|register:登录页回填用 login,入驻页回填用 register,两场景码分储不串用。
func devSmsCodeHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req devSmsCodeReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Phone, "phone", 20),
			)
		}) {
			return
		}
		if req.Scene == "" {
			req.Scene = "login"
		}
		if req.Scene != "login" && req.Scene != "register" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		sc, err := a.Portal.LatestSmsCode(c.Request.Context(), req.Phone, req.Scene)
		if err != nil {
			respondErr(c, err)
			return
		}
		if sc == nil {
			respond(c, apitypes.CodeNotFound, gin.H{"hint": "no valid code; call /auth/sms-code first"})
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"code": sc.Code, "issuedAt": sc.IssuedAt})
	}
}

// registerWorkerDevRoutes 注册开发模式路由;DevMode=false 时静默跳过,
// 生产装配不暴露路径(404 由 gin 默认中间件兜底)。
func registerWorkerDevRoutes(g *gin.RouterGroup, a *app.Application) {
	if !config.Load().Server.DevMode {
		return
	}
	g.POST("/auth/dev/sms-code", devSmsCodeHandler(a))
}
