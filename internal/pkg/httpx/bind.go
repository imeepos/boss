package httpx

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// BindBody 绑定 JSON 请求体;失败已回 422 envelope,返回 false 供 handler 提前返回。
func BindBody(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		Respond(c, apitypes.CodeInvalidParam, nil)
		return false
	}
	return true
}
