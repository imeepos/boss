package adminapi

// 扩容单写侧 handler(写侧补齐):执行(目标设备批量建端口)+ 驳回;契约 oss.yaml /expansions。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// executeExpansionReq 扩容执行请求体。
type executeExpansionReq struct {
	ResourceID int64 `json:"resourceId" binding:"required,min=1"`
}

// executeExpansionHandler POST /expansions/:expansionNo/execute:执行扩容。
func executeExpansionHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req executeExpansionReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		no := c.Param("expansionNo")
		res, err := a.ResourceSub.ExecuteExpansion(c.Request.Context(), no, req.ResourceID)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "expansion", no,
			map[string]any{"result": "executed", "resourceId": req.ResourceID, "created": res.Created, "total": res.TotalPorts})
		respond(c, apitypes.CodeOK, res)
	}
}

// rejectExpansionHandler POST /expansions/:expansionNo/reject:扩容驳回(PENDING→DONE)。
func rejectExpansionHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		no := c.Param("expansionNo")
		if err := a.ResourceSub.RejectExpansion(c.Request.Context(), no); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "expansion", no, map[string]any{"result": "rejected"})
		respond(c, apitypes.CodeOK, nil)
	}
}
