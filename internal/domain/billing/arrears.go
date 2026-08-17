package billing

import "context"

// Arrears 欠费态快照(应收信用域,客户1:1)。
type Arrears struct {
	ID         int64
	CustomerID int64
	Amount     float64 // 欠费金额
	Days       int32   // 欠费天数
	Status     string  // 催收中/已停机等
}

// StopResumeTask 停复机任务流水(欠费停机/缴费复机)。
type StopResumeTask struct {
	ID          int64
	CustomerID  int64 // 软引用 customers.id
	LoAccountID int64 // 软引用 lo_accounts.id
	Action      string // STOP停机/RESUME复机
	Status      string // PENDING/DOING/DONE/FAILED
}

// ArrearsService 应收信用域服务口(阶段5):欠费快照 + 停复机流水。
type ArrearsService interface {
	GetArrears(ctx context.Context, customerID int64) (*Arrears, error)
	UpsertArrears(ctx context.Context, a Arrears) (int64, error)
	ListStopResumeTasks(ctx context.Context, customerID int64) ([]StopResumeTask, error)
	AppendStopResumeTask(ctx context.Context, t StopResumeTask) (int64, error)
}
