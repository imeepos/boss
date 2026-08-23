// 沙箱样例数据(M4):sandbox=true 的开放应用在 /api/open/v1 只能读到这里的
// 固定样例(回放 fixture),与生产订单数据隔离。样例与
// scripts/openplat-selftest.mjs、docs/integration/open-platform.md 三处同源,
// 修改必须同步(契约样例,纳入 M5 契约测试覆盖)。
package openplat

import "encoding/json"

// SandboxOrderNoPrefix 沙箱样例订单号前缀。
const SandboxOrderNoPrefix = "SBX-"

// sandboxOrders 样例订单集(覆盖 PENDING/RESERVED/INSTALLING/DONE 四态 + 12 环节梯度)。
var sandboxOrders = []openOrderView{
	{OrderNo: "SBX-ORD-0001", Status: "PENDING", Stage: 1, OfferID: 101, CreatedAt: "2026-08-01T08:00:00Z"},
	{OrderNo: "SBX-ORD-0002", Status: "RESERVED", Stage: 6, OfferID: 102, CreatedAt: "2026-08-02T09:30:00Z"},
	{OrderNo: "SBX-ORD-0003", Status: "INSTALLING", Stage: 9, OfferID: 101, CreatedAt: "2026-08-03T14:00:00Z"},
	{OrderNo: "SBX-ORD-0004", Status: "DONE", Stage: 12, OfferID: 103, CreatedAt: "2026-08-04T16:45:00Z"},
}

// openOrderView 与 internal/httpapi/open 的订单只读投影同形(避免域依赖倒置,
// 此处以独立结构维护,M5 契约测试对齐两者字段)。
type openOrderView struct {
	OrderNo   string `json:"orderNo"`
	Status    string `json:"status"`
	Stage     int8   `json:"stage"`
	OfferID   int64  `json:"offerId"`
	CreatedAt string `json:"createdAt"`
}

// SandboxOrderByNo 取沙箱样例订单;不存在返回 false。
func SandboxOrderByNo(orderNo string) (json.RawMessage, bool) {
	for _, o := range sandboxOrders {
		if o.OrderNo == orderNo {
			b, _ := json.Marshal(o)
			return b, true
		}
	}
	return nil, false
}

// SandboxOrderNos 全部样例订单号(自助验收工具遍历用)。
func SandboxOrderNos() []string {
	out := make([]string, 0, len(sandboxOrders))
	for _, o := range sandboxOrders {
		out = append(out, o.OrderNo)
	}
	return out
}

// IsSandboxOrderNo 判定订单号是否沙箱样例(生产应用查 SBX-* 一律 404)。
func IsSandboxOrderNo(orderNo string) bool {
	return len(orderNo) > len(SandboxOrderNoPrefix) && orderNo[:len(SandboxOrderNoPrefix)] == SandboxOrderNoPrefix
}

// WebhookReplaySample Webhook 验签练习样例:集成方用自己 Secret 对
// timestamp+body 复算 v1 签名,与 expectedV1 比对(算法见 SignPayload)。
// 固定 secret="ops_sandbox_demo" 仅用于练习,不用于任何真实应用。
type WebhookReplaySample struct {
	EventID    string `json:"eventId"`
	EventType  string `json:"eventType"`
	Timestamp  string `json:"timestamp"`
	Body       string `json:"body"`
	ExpectedV1 string `json:"expectedV1"`
}

// SandboxWebhookSample 生成回放样例(当前时间戳,每次调用可复算)。
func SandboxWebhookSample() WebhookReplaySample {
	const demoSecret = "ops_sandbox_demo"
	body := `{"type":"order.stage.done","orderNo":"SBX-ORD-0003","stage":9,"status":"INSTALLING"}`
	ts := "1755859200" // 2025-08-22T12:00:00Z,固定值保证可复现
	return WebhookReplaySample{
		EventID:    "SBX-ORD-0003:stage:9",
		EventType:  "order.stage.done",
		Timestamp:  ts,
		Body:       body,
		ExpectedV1: SignPayload(demoSecret, ts, []byte(body)),
	}
}
