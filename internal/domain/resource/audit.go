package resource

// 资源台账稽核(P5-W3,对标 NRM C7 库存质量稽核):三类只读检查,只报不修,
// 不改状态机语义。口径权威:docs/contract/fields.md §4.2 + terms.md §4 状态枚举。
// 出口:GET /inventory-audit(menu:resource)+ 每日快照(report_snapshots/oss-audit-daily)。

import (
	"context"
	"time"
)

// 稽核类别标识(fields.md §4.2 登记,API ?category= 过滤值)。
const (
	AuditCatOwnership = "ownership" // 归属断裂:引用链断裂
	AuditCatState     = "state"     // 状态机违例:状态与关联事实矛盾
	AuditCatCoding    = "coding"    // 编码规范违例:编码不符合命名规则
)

// AuditDefaultStaleHours RESERVED 超时阈值缺省(48 小时;biz_params
// resource.audit.reservedStaleHours 可热调,见迁移 000193)。
const AuditDefaultStaleHours = 48

// AuditSampleLimit 每检查明细样本上限;计数为全量,明细截断防大结果面。
const AuditSampleLimit = 50

// AuditOptions 稽核入参:Category 空=全部三类;ReservedStaleHours<=0 回退默认。
type AuditOptions struct {
	Category           string
	ReservedStaleHours int
}

// AuditItem 一条稽核明细(违规行样本)。
type AuditItem struct {
	Check    string `json:"check"`    // 检查码,如 PORT_SPLITTER_MISSING
	Category string `json:"category"` // ownership/state/coding
	ID       int64  `json:"id"`       // 违规行主键(端口/设备 id)
	Code     string `json:"code"`     // 端口码/设备编码
	Detail   string `json:"detail"`   // 上下文(引用目标/状态起始时间等)
}

// AuditCategoryCount 单类别违规计数。
type AuditCategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// AuditReport 一次稽核结果:三类计数 + 明细。
type AuditReport struct {
	GeneratedAt time.Time            `json:"generatedAt"`
	StaleHours  int                  `json:"staleHours"` // RESERVED 阈值(本次生效值)
	Counts      []AuditCategoryCount `json:"counts"`
	Items       []AuditItem          `json:"items"`
	Total       int64                `json:"total"`
}

// InventoryAuditor 资源台账稽核可选能力(OrphanPatroller 模式窄口断言,
// 不扩 ResourceService 主接口,便于测试 fake 降级)。
type InventoryAuditor interface {
	AuditInventory(ctx context.Context, opts AuditOptions) (*AuditReport, error)
}
