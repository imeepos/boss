package adminapi

// Q2 四码冲突 4 小时清零率:GET /quad-links/clearance-rate(menu:quadlink)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// quadClearanceRateHandler GET /quad-links/clearance-rate:7 天窗口清零率(验收 ≥95%)。
// QuadLink 未实现窄口(测试 fake/降级部署)时返回 501 语义错误码。
func quadClearanceRateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		rd, ok := a.QuadLink.(quadlink.ClearanceStatReader)
		if !ok {
			respond(c, apitypes.CodeInvalidParam, gin.H{"msg": "clearance stats unsupported"})
			return
		}
		st, err := rd.ClearanceStats(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, st)
	}
}
