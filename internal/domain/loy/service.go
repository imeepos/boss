package loy

import (
	"context"
	"errors"
)

// ErrNotFound 账本未开通(无流水记录)。
var ErrNotFound = errors.New("loy: not found")

// ErrConflict 积分不足/不可兑换。
var ErrConflict = errors.New("loy: conflict")

// 流水原因。
const (
	ReasonAdjust   = "ADMIN_ADJUST"
	ReasonExchange = "EXCHANGE"
	ReasonReversal = "EXCHANGE_REVERSAL"
)

// Entry 积分流水。
type Entry struct {
	EntryID      int64  `json:"entryId"`
	CustomerID   int64  `json:"customerId"`
	Delta        int64  `json:"delta"`
	BalanceAfter int64  `json:"balanceAfter"`
	Reason       string `json:"reason"`
	RefID        int64  `json:"refId"` // EXCHANGE 时为模板 id
	CreatedAt    string `json:"createdAt"`
}

// PriceOf 模板积分价查询(promotion 注入;0=不可兑换)。
type PriceOf func(ctx context.Context, templateID int64) (int64, error)

// IssueCoupon 兑换发券(promotion 注入;返回券号)。
type IssueCoupon func(ctx context.Context, templateID, customerID int64) (string, error)

// Service 积分域接口。
type Service interface {
	// Balance 余额(无账本视为 0)。
	Balance(ctx context.Context, customerID int64) (int64, error)
	// Entries 流水(近 100 条)。
	Entries(ctx context.Context, customerID int64) ([]Entry, error)
	// Adjust 管理端手动调整(delta 正充负扣);扣减不得为负。
	Adjust(ctx context.Context, customerID int64, delta int64, reason string) (int64, error)
	// Exchange 积分换券:先扣积分(行锁),发券失败补偿回补;返回券号与消耗积分。
	Exchange(ctx context.Context, customerID, templateID int64) (couponID string, cost int64, err error)
}
