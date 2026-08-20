package billing

import (
	"context"
	"errors"
	"math"
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

// 对账明细差异种类:MATCH 双边一致 / MISSING_SYSTEM 渠道有系统无 /
// MISSING_CHANNEL 系统有渠道无 / AMOUNT_MISMATCH 双边金额不一致。
const (
	DiffMatch          = "MATCH"
	DiffMissingSystem  = "MISSING_SYSTEM"
	DiffMissingChannel = "MISSING_CHANNEL"
	DiffAmountMismatch = "AMOUNT_MISMATCH"
)

// ReconItem 对账批次行级明细:一行 = 一笔渠道流水的比对结论。
type ReconItem struct {
	ID         int64   `json:"id"`
	BatchID    int64   `json:"batchId"`
	PaymentID  *int64  `json:"paymentId,omitempty"` // 软引用 payments;渠道独有流水为 nil
	ChannelRef string  `json:"channelRef"`
	Amount     float64 `json:"amount"`
	DiffKind   string  `json:"diffKind"`
	Note       string  `json:"note"`
}

// ChannelStatementRow 渠道侧流水行(手工/导入录入,本期不自动拉取)。
type ChannelStatementRow struct {
	ChannelRef string  `json:"channelRef"` // 渠道流水号,与 payments.pay_no 对齐比对
	Amount     float64 `json:"amount"`
}

// PaymentRef 比对用系统侧缴费投影(pay_no 唯一)。
type PaymentRef struct {
	ID     int64
	PayNo  string
	Amount float64
}

// ReconService 渠道对账域服务口:批次列表 + 差异挂起平账 + 行级明细比对。
type ReconService interface {
	ListReconciliations(ctx context.Context) ([]ReconBatch, error)
	AppendReconciliation(ctx context.Context, b ReconBatch) (int64, error)
	SettleReconciliation(ctx context.Context, batchNo string) error
	GetReconciliation(ctx context.Context, batchNo string) (*ReconBatch, error)
	// RecordChannelStatement 录入渠道侧流水并逐行比对生成 items,重算批次总额与状态。
	RecordChannelStatement(ctx context.Context, batchID int64, rows []ChannelStatementRow) error
	ListReconciliationItems(ctx context.Context, batchID int64) ([]ReconItem, error)
}

// CompareStatement 纯比对:渠道流水行 vs 系统侧缴费(按 pay_no=channel_ref 匹配),
// 双边金额按分比较;返回逐行明细(MATCH 行也落,便于对账单审计)。
func CompareStatement(rows []ChannelStatementRow, pays []PaymentRef) []ReconItem {
	byPayNo := make(map[string]PaymentRef, len(pays))
	for _, p := range pays {
		byPayNo[p.PayNo] = p
	}
	seen := make(map[string]bool, len(rows))
	items := make([]ReconItem, 0, len(rows)+len(pays))
	for _, r := range rows {
		seen[r.ChannelRef] = true
		items = append(items, compareRow(r, byPayNo[r.ChannelRef]))
	}
	for _, p := range pays {
		if !seen[p.PayNo] {
			items = append(items, ReconItem{PaymentID: &p.ID, ChannelRef: p.PayNo,
				Amount: p.Amount, DiffKind: DiffMissingChannel})
		}
	}
	return items
}

// compareRow 单行判定:无系统缴费=MISSING_SYSTEM;金额一致=MATCH;否则 AMOUNT_MISMATCH。
func compareRow(r ChannelStatementRow, p PaymentRef) ReconItem {
	it := ReconItem{ChannelRef: r.ChannelRef, Amount: r.Amount}
	if p.ID == 0 {
		it.DiffKind = DiffMissingSystem
		return it
	}
	it.PaymentID = &p.ID
	if cents(r.Amount) == cents(p.Amount) {
		it.DiffKind = DiffMatch
	} else {
		it.DiffKind = DiffAmountMismatch
		it.Note = "system=" + p.PayNo
	}
	return it
}

// cents 元转分,规避浮点直比误差。
func cents(v float64) float64 { return math.Round(v * 100) }
