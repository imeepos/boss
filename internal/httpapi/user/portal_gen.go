package userapi

// 用户端门户通用小工具(跨 handler 复用):枚举标签、map 取值、地址/产品名解析。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
)

// portalStageTitles 12 环节标题(terms.md §1 权威清单,序号禁止增删改序)。
var portalStageTitles = map[int]string{
	1: "下单", 2: "资源核查", 3: "端口预占", 4: "合同收费", 5: "标签预绑定",
	6: "创建账号", 7: "预下发配置", 8: "派单", 9: "扫码绑定", 10: "激活",
	11: "激活回调", 12: "更新 GIS",
}

// portalOrderStatusLabel 订单状态中文标签(terms.md §3)。
var portalOrderStatusLabel = map[string]string{
	"PENDING": "待核查", "RESERVED": "已预占", "INSTALLING": "装维中",
	"DONE": "已完成", "CANCELLED": "已取消",
}

// portalOrderStageLabel 环节序号 → 环节标题。
func portalOrderStageLabel(stage int8) string {
	if t, ok := portalStageTitles[int(stage)]; ok {
		return t
	}
	return "已归档"
}

// portalOrderSummary 订单读模型 → 契约 OrderSummary。
func portalOrderSummary(o *order.Order, productName, address string, canRate bool) gin.H {
	return gin.H{
		"orderNo": o.OrderNo, "status": o.Status,
		"statusLabel": portalOrderStatusLabel[o.Status],
		"productName": productName, "address": address,
		"stage": o.Stage, "stageLabel": portalOrderStageLabel(o.Stage),
		"estimateFinish": "", "canRate": canRate,
	}
}

// statusFilter 契约标签 → 订单状态;in_progress 含 RESERVED/INSTALLING。
func statusFilter(status string) string {
	switch status {
	case "done":
		return "DONE"
	case "cancelled":
		return "CANCELLED"
	case "in_progress":
		return "INSTALLING"
	default:
		return ""
	}
}

// addressName 地址名:联表地址缺失时回退用户地址簿(user_addresses)。
func addressName(a *app.Application, c *gin.Context, addressID int64, fallback string) string {
	if fallback != "" {
		return fallback
	}
	if a.UserData == nil {
		return ""
	}
	addrs, err := a.UserData.ListUserAddresses(c.Request.Context())
	if err != nil {
		return ""
	}
	for _, row := range addrs {
		if toInt64(row["id"]) == addressID {
			return toStr(row["detail"])
		}
	}
	return ""
}

// productName 产品名:主档产品表寻址,失败回退空串。
func productName(a *app.Application, c *gin.Context, offerID int64) string {
	prods, err := a.Product.ListProducts(c.Request.Context(), 0)
	if err != nil {
		return ""
	}
	for _, p := range prods {
		if p.ID == offerID {
			return p.Name
		}
	}
	return ""
}

// toInt64 map 值转 int64(JSON 数值解码为 float64)。
func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	case int:
		return int64(n)
	}
	return 0
}

// toStr map 值转 string。
func toStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// toBool map 值转 bool(PG bool 与 JSON bool 兼容)。
func toBool(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return b == "true" || b == "t"
	}
	return false
}

// portalID 契约字符串 ID → int64。
func portalID(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}