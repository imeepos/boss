package adminapi

// registerProvisionSeedRoutes 后台置备端点(CLI 联调/开网全流程模拟用)。
// 口径:工程质量"先可观测后自动化"——先落地置备端点,供管理后台与 CLI 走通 12 环节流程;
// 域复用既有 create 服务口,不重复造轮子。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerProvisionSeedRoutes 注册置备端点:资源/端口、渠道、资产批次/标签/资产、派单工单。
func registerProvisionSeedRoutes(g *gin.RouterGroup, a *app.Application) {
	p := g.Group("/provision", requirePerm(a.User, "menu:provision"))
	// 网络资源(含端口):环节2/3 前置。
	p.POST("/resources", createResourceHandler(a))
	p.POST("/ports", createPortHandler(a))
	// 渠道:下单前置;GET 目录供联调取 channelId(无需翻 DB)。
	p.GET("/channels", listChannelsHandler(a))
	p.POST("/channels", createChannelHandler(a))
	// 资产批次/标签/资产:环节5 标签预绑定与环节9 扫码前置。
	p.POST("/asset-batches", createAssetBatchHandler(a))
	p.POST("/tags", createTagHandler(a))
	p.POST("/assets", createAssetHandler(a))
	// 派单工单:环节8 派单后置备工单,再入池指派师傅。
	p.POST("/dispatch-tickets", createDispatchTicketHandler(a))
}

// strItoa int64 → string(审计记录用,避免引入 fmt)。
func strItoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}