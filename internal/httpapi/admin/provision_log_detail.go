package adminapi

// 下发日志详情 handler:GET /provision-logs/{logId}(后台日志详情页数据源)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// provisionLogDetailHandler GET /provision-logs/{logId}:日志本体(完整指令/设备应答)
// + 任务/订单/模板三维上下文,详情页一次取齐。
func provisionLogDetailHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("logId"), 10, 64)
		if err != nil || id <= 0 {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "logId is required"})
			return
		}
		d, err := a.Provision.GetLogDetail(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, d)
	}
}
