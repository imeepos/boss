package billing

import (
	"context"
	"errors"
	"time"
)

// ErrIllegalReconTransition 非法对账状态迁移(如已平账再平账)。
var ErrIllegalReconTransition = errors.New("billing: illegal reconciliation transition")

// ReconBatch 缴费渠道对账批次(渠道侧金额 vs 系统侧金额)。
type ReconBatch struct {
	ID            int64      `json:"id"`
	BatchNo       string     `json:"batchNo"`
	Channel       string     `json:"channel"`
	ChannelAmount float64    `json:"channelAmount"`
	SystemAmount  float64    `json:"systemAmount"`
	Diff          float64    `json:"diff"`   // 渠道-系统
	Status        string     `json:"status"` // DIFF_PENDING 差异挂起 / SETTLED 已平账
	CreatedAt     time.Time  `json:"createdAt"`
	SettledAt     *time.Time `json:"settledAt,omitempty"` // nil=未平账
}

// ReconService 渠道对账域服务口:批次列表 + 差异挂起平账。
type ReconService interface {
	ListReconciliations(ctx context.Context) ([]ReconBatch, error)
	AppendReconciliation(ctx context.Context, b ReconBatch) (int64, error)
	SettleReconciliation(ctx context.Context, batchNo string) error
}
