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
	ID          int64  `json:"id"`
	CustomerID  int64  `json:"customerId"`  // 软引用 customers.id
	LoAccountID int64  `json:"loAccountId"` // 软引用 lo_accounts.id
	Action      string `json:"action"`      // STOP停机/RESUME复机
	Status      string `json:"status"`      // PENDING/DOING/DONE/FAILED
}

// ArrearsItem 欠费列表读模型(联表展示客户名,承接 arrears.html)。
type ArrearsItem struct {
	CustomerID   int64   `json:"customerId"`
	CustomerName string  `json:"customer"`
	Amount       float64 `json:"amount"`
	Days         int32   `json:"days"`
	Status       string  `json:"status"`
}

// ArrearsService 应收信用域服务口(阶段5):欠费快照 + 停复机流水。
type ArrearsService interface {
	GetArrears(ctx context.Context, customerID int64) (*Arrears, error)
	UpsertArrears(ctx context.Context, a Arrears) (int64, error)
	ListArrears(ctx context.Context) ([]ArrearsItem, error)
	ListStopResumeTasks(ctx context.Context, customerID int64) ([]StopResumeTask, error)
	AppendStopResumeTask(ctx context.Context, t StopResumeTask) (int64, error)
}
