package adminapi

// A1:LOID 接入密码重置(随机生成;明文仅本次响应返回一次,库内只有密文)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// aaaResetPasswordHandler POST /lo-accounts/{loid}/reset-password(menu:loaccount)。
// 重置成功同时清除该 LOID 的防爆破锁定计数(新密码从零计)。
func aaaResetPasswordHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		loid := c.Param("loid")
		password, err := a.Aaa.ResetLoPassword(c.Request.Context(), loid)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"loid": loid, "password": password})
	}
}
