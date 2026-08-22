package adminapi

// Q2 补偿任务中心:GET /compensation-tasks(跨域可回放补偿任务,menu:report)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// compensationTasksHandler GET /compensation-tasks:六类补偿任务聚合清单,
// 每项带可回放 admin POST 路径;调对应域 retry/settle 接口完成回放。
func compensationTasksHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tasks, err := a.Report.CompensationTasks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": tasks, "total": len(tasks)})
	}
}
