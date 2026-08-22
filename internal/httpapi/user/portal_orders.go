package userapi

// 用户端门户 Order 域核心端点:订单列表/下单/详情/时间轴 + 渠道解析。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
)

// portalOrderTimeline 12 环节时间轴:DB 环节日志(StageLog)为基,缺失环节按 PENDING 补齐。
func portalOrderTimeline(stages []order.StageLog) []gin.H {
	byStage := make(map[int8]order.StageLog, len(stages))
	for _, s := range stages {
		byStage[s.Stage] = s
	}
	out := make([]gin.H, 0, len(portalStageTitles))
	for i := 1; i <= len(portalStageTitles); i++ {
		t := int8(i)
		result := "PENDING"
		if s, ok := byStage[t]; ok {
			result = s.Result
		}
		out = append(out, gin.H{"stage": i, "title": portalStageTitles[i], "result": result, "meta": ""})
	}
	return out
}

// portalOrderAddress 订单详情地址:优先复用列表读模型的联表地址(addresses/user_addresses),
// 与 /orders 列表口径一致;联表也取不到时回退地址簿查名。
func portalOrderAddress(a *app.Application, c *gin.Context, cid int64, o *order.Order) string {
	rows, err := a.Order.List(c.Request.Context(), order.OrderQuery{
		Keyword: o.OrderNo, CustomerID: cid, Limit: 1,
	})
	if err == nil && len(rows) > 0 && rows[0].Address != "" {
		return rows[0].Address
	}
	return addressName(a, c, o.AddressID, "")
}

// portalCompletedStage 截至当前已完成的最大环节号。
func portalCompletedStage(stages []order.StageLog) int {
	completed := 0
	for _, s := range stages {
		if s.Result != "PENDING" && int(s.Stage) > completed {
			completed = int(s.Stage)
		}
	}
	return completed
}

// portalChannelID 解析渠道 ID:显式传入按 ID 校验,缺省选 ONLINE 渠道;目录未接入时返回 0。
func portalChannelID(c *gin.Context, a *app.Application, id string) (int64, error) {
	if id != "" {
		return strconv.ParseInt(id, 10, 64)
	}
	if a.Channel == nil {
		return 0, nil
	}
	channels, err := a.Channel.ListChannels(c.Request.Context())
	if err != nil {
		return 0, err
	}
	for _, ch := range channels {
		if ch.Status == "ACTIVE" && (ch.Code == "ONLINE" || ch.ID == 102) {
			return ch.ID, nil
		}
	}
	return 0, nil
}