// Package notify 提供后台提醒中心:admin 通知(todo 待办)+ 后台任务结果(task)。
//
// 模型(docs/plan/admin-notify-center.md):
//   - 广播+读回执:admin_notifications 面向角色广播,admin_notification_reads 按账号记已读;
//   - resolved(来源域驱动的 todo 生命周期)与 read(账号视角)分离;
//   - Emit 以 (refType, refID, category) 幂等,任务类通知只发一次。
package notify

import (
	"context"
	"errors"
)

// ErrInvalidInput Emit 入参非法。
var ErrInvalidInput = errors.New("notify: invalid input")

// category 枚举。
const (
	CategoryTodo = "todo"
	CategoryTask = "task"
)

// level 枚举(复用 docs/contract/terms.md 消息 level)。
const (
	LevelInfo   = "INFO"
	LevelWarn   = "WARN"
	LevelUrgent = "URGENT"
)

// Input Emit 入参;RefType/RefID/Category 构成幂等键。
type Input struct {
	Category   string // todo | task
	Level      string // INFO | WARN | URGENT
	Title      string
	Content    string
	Link       string // 前端路由
	RefType    string // 来源域标识,如 importer
	RefID      string // 来源域主键
	TargetRole string // 空=全部后台角色
}

// Item 列表条目(含当前账号读状态)。
type Item struct {
	ID         int64  `json:"id"`
	Category   string `json:"category"`
	Level      string `json:"level"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Link       string `json:"link"`
	RefType    string `json:"refType"`
	RefID      string `json:"refId"`
	Resolved   bool   `json:"resolved"`
	CreatedAt  string `json:"createdAt"`
	Read       bool   `json:"read"`
	TargetRole string `json:"-"` // 服务端过滤用,不下发
}

// Filter 列表过滤;零值=不过滤。
type Filter struct {
	Category     string
	Level        string
	Unread       bool // true=仅未读
	HideResolved bool // true=隐藏已 resolved(todo 已办)
	Limit        int
	Offset       int
}

// Service 后台提醒中心接口。
type Service interface {
	// Emit 发一条提醒;(refType, refID, category) 已存在时幂等跳过。
	Emit(ctx context.Context, in Input) error
	// Resolve 把来源域指定的未办 todo 批量置 resolved(工单办结/审核完成回调)。
	Resolve(ctx context.Context, refType, refID string) error
	// List 按角色过滤清单(role 匹配 target_role IN ('', role)),含账号读状态;返回条目+总数。
	List(ctx context.Context, role string, accountID int64, f Filter) ([]Item, int, error)
	// UnreadCount 当前账号可见的未读数(resolved 未读不计)。
	UnreadCount(ctx context.Context, role string, accountID int64) (int, error)
	// MarkRead 批量已读;ids 为空=当前角色可见全部已读。
	MarkRead(ctx context.Context, role string, accountID int64, ids []int64) error
}

// Validate 校验入参(level 空合法,由 NormalizeDefaults 补默认);非法返回 false。
func (in Input) Valid() bool {
	if in.Category != CategoryTodo && in.Category != CategoryTask {
		return false
	}
	if in.Level != "" && in.Level != LevelInfo && in.Level != LevelWarn && in.Level != LevelUrgent {
		return false
	}
	return in.Title != "" && in.RefType != "" && in.RefID != ""
}

// NormalizeDefaults 补默认 level:task=INFO,todo=WARN。
func (in Input) NormalizeDefaults() Input {
	if in.Level == "" {
		if in.Category == CategoryTodo {
			in.Level = LevelWarn
		} else {
			in.Level = LevelInfo
		}
	}
	return in
}
