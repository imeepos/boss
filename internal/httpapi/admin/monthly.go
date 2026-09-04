package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/monthly"
)

// registerMonthlyRoutes 月度填报事实域路由(menu:monthly,迁移 000181,前端页 T20)。
// 路径段=域表标识:user-revenue / network-delivery / finance-cost。
func registerMonthlyRoutes(g *gin.RouterGroup, a *app.Application) {
	m := g.Group("", requirePerm(a.User, "menu:monthly"))

	m.GET("/monthly/regions", monthlyRegionsHandler(a))
	m.GET("/monthly/summary", monthlySummaryHandler(a))

	m.GET("/monthly/user-revenue", monthlyListHandler(a, monthly.TableUserRevenue))
	m.PUT("/monthly/user-revenue", monthlyUpsertHandler(a, monthly.TableUserRevenue))
	m.POST("/monthly/user-revenue/import", monthlyImportHandler(a, monthly.TableUserRevenue))
	m.GET("/monthly/user-revenue/export", monthlyExportHandler(a, monthly.TableUserRevenue))

	m.GET("/monthly/network-delivery", monthlyListHandler(a, monthly.TableNetworkDelivery))
	m.PUT("/monthly/network-delivery", monthlyUpsertHandler(a, monthly.TableNetworkDelivery))
	m.POST("/monthly/network-delivery/import", monthlyImportHandler(a, monthly.TableNetworkDelivery))
	m.GET("/monthly/network-delivery/export", monthlyExportHandler(a, monthly.TableNetworkDelivery))

	m.GET("/monthly/finance-cost", monthlyListHandler(a, monthly.TableFinanceCost))
	m.PUT("/monthly/finance-cost", monthlyUpsertHandler(a, monthly.TableFinanceCost))
	m.POST("/monthly/finance-cost/import", monthlyImportHandler(a, monthly.TableFinanceCost))
	m.GET("/monthly/finance-cost/export", monthlyExportHandler(a, monthly.TableFinanceCost))
}
