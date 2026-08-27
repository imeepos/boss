// 事件目录:订阅中心可订阅事件类型的权威清单,前端事件选择器数据源。
// 登记规则:事件在 emit 侧落地(如 order.StageEventType 广播点)后在此登记,
// 未登记事件不进目录、不可被集成方订阅。
package openplat

import "strings"

// MaxSubEventsPerRequest 单次批量订阅事件数上限。
const MaxSubEventsPerRequest = 32

// EventType 目录项。description 为英文基准文案,多语言展示由 admin 前端 i18n 承担。
type EventType struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// eventCatalog 可订阅事件目录。
var eventCatalog = []EventType{
	{Type: "order.stage.done", Description: "Order stage advanced (installation progress)"},
}

// EventCatalog 返回事件目录只读副本。
func EventCatalog() []EventType {
	out := make([]EventType, len(eventCatalog))
	copy(out, eventCatalog)
	return out
}

// NormalizeEventTypes 事件类型清单 trim、去空、去重(保序)。
func NormalizeEventTypes(types []string) []string {
	seen := make(map[string]struct{}, len(types))
	out := make([]string, 0, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
