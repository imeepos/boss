package adminapi

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// genNo 生成台账单号(缺省时):前缀-日期-纳秒尾(12 位熵,防跨轮持久库唯一号撞车)。
func genNo(prefix string) string {
	now := time.Now()
	return fmt.Sprintf("%s-%s-%012d", prefix, now.Format("20060102"), now.UnixNano()%1e12)
}

// registerResourceRoutes 注册网络资源域路由(承接 api/openapi/admin/oss.yaml)。
func registerResourceRoutes(g *gin.RouterGroup, a *app.Application) {
	res := g.Group("", requirePerm(a.User, "menu:resource"))
	res.GET("/resources", listResourcesHandler(a))
	res.GET("/resources/capacity", capacityHandler(a))
	res.POST("/resources/capacity/alert-scan", capacityAlertScanHandler(a))
	res.GET("/ports", listPortsHandler(a))
	res.GET("/ports/:portId/change-history", listPortHistoryHandler(a))
	res.GET("/ports/:portId/path", portPathHandler(a))
	res.POST("/reserves/:reserveId/release", releaseReserveHandler(a))
	res.GET("/reserves", listReservesHandler(a))

	tr := g.Group("", requirePerm(a.User, "menu:transfer"))
	tr.POST("/transfers", createTransferHandler(a))
	tr.POST("/transfers/:transferNo/approve", approveTransferHandler(a))
	tr.POST("/transfers/:transferNo/reject", rejectTransferHandler(a))
	tr.GET("/transfers", listTransfersHandler(a))
	tr.GET("/expansions", listExpansionsHandler(a))
	tr.POST("/expansions", createExpansionHandler(a))
	tr.GET("/expansions/qos-templates", listQosTemplatesHandler(a))
}
