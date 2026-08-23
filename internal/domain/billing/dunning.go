package billing

import (
	"context"
	"time"
)

// 欠费催收状态(arrears.status 快照值)。
const (
	ArrearsCollecting = "COLLECTING" // 催收中(账单已逾期,未到停机线)
	ArrearsStopped    = "STOPPED"    // 已停机(逾期超停机线,LO 已 STOP)
)

// OverdueCustomer 欠费客户读模型(dunning run 输入):逾期合计与最早逾期时间。
type OverdueCustomer struct {
	CustomerID int64
	Amount     float64
	OldestAt   time.Time
}

// DunningResult 一次催收批处理结果。
type DunningResult struct {
	OverdueBills int     `json:"overdueBills"` // 本次置 OVERDUE 的账单数
	ArrearsUps   int     `json:"arrearsUps"`   // 更新欠费快照的客户数
	StoppedIDs   []int64 `json:"stoppedIds"`   // 本次自动停机的客户(运营留痕清单)
}

// DunningService 欠费催收域口(Q3 收口):逾期标记 + 欠费清单。
// 停机动作跨域(AAA),编排见 app.RunDunning。
type DunningService interface {
	// MarkOverdueBills 超过宽限仍未缴的账单置 OVERDUE(graceDays 按账单生成时间起算)。
	MarkOverdueBills(ctx context.Context, graceDays int) (int, error)
	// ListOverdueCustomers 逾期客户清单:合计金额 + 最早逾期时间(算欠费天数用)。
	ListOverdueCustomers(ctx context.Context) ([]OverdueCustomer, error)
}
