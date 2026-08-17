// Package audit 关键操作审计:操作人/时间/内容,按月分区表,可按人/时间/类型查询。
// 阶段1验收项:关键操作(数据变更/状态变更/权限变更)记录可查、可追溯。
package audit

import (
	"context"
	"time"
)

// Event 一条待写入的审计事件(来源侧构造,不感知存储)。
type Event struct {
	AccountID  int64          // 操作人账号 id
	Action     string         // 数据变更/状态变更/权限变更
	TargetType string         // 目标对象类型,如 order/bill/port
	TargetID   string         // 目标对象 id
	Detail     map[string]any // 操作详情(变更前后值),JSONB
	IP         string         // 来源 IP
}

// Query 审计查询条件。
type Query struct {
	AccountID  int64  // 0=全部
	Action     string // 空=全部
	TargetType string // 空=全部
	Limit      int    // <=0 视为不限
	Offset     int
}

// Entry 一条已落库的审计记录。
type Entry struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"accountId"`
	Action     string    `json:"action"`
	TargetType string    `json:"targetType"`
	TargetID   string    `json:"targetId"`
	Detail     string    `json:"detail"` // JSON 文本
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Writer 审计写入/查询口;实现异步或同步由具体类型决定(接口不承诺时序)。
type Writer interface {
	Write(ctx context.Context, e Event) error
	List(ctx context.Context, q Query) ([]Entry, error)
}
