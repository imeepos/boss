package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// aaaSummaryHandler 使用数据库侧聚合并套用当前账号数据范围。
func aaaSummaryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := a.Aaa.(aaa.AdminQueryService)
		if !ok {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		scope, err := aaaScope(c, a.User)
		if err != nil {
			respondErr(c, err)
			return
		}
		summary, err := svc.GetAdminSummary(c.Request.Context(), scope)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"summary": summary})
	}
}
