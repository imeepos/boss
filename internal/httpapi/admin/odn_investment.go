package adminapi

// ODN 网格投资测算读模型路由(P-INFRA-1 W2;纯只读,无写路径)。
// 口径:internal/domain/odn/investment.go + fields.md 1.5.11;契约 api/openapi/admin/odn.yaml。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerODNInvestmentRoutes 注册投资测算路由(门禁 menu:grid-investment,与 menu.def key 同源)。
func registerODNInvestmentRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/odn/grid-investment", requirePerm(a.User, "menu:grid-investment"), odnGridInvestmentHandler(a))
}

// odnGridInvestmentHandler GET /odn/grid-investment:网格投资测算聚合列表。
// settledCost/costPerServed 为 null 表示「未登记」(W1 结算源缺失或该网格无数据),禁止显示 0。
func odnGridInvestmentHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := a.ODN.GridInvestment(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": rows})
	}
}
