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
	ReasonAdjust    = "ADMIN_ADJUST"
	ReasonExchange  = "EXCHANGE"
	ReasonReversal  = "EXCHANGE_REVERSAL"
	ReasonPayEarn   = "PAYMENT_EARN"     // ref=payment_id
	ReasonPayRoll   = "PAYMENT_REVERSAL" // ref=payment_id,退款冲销
	ReasonTask      = "TASK_EARN"        // ref=task_id
	ReasonExpired   = "EXPIRED"          // ref=被过期汇总的最早 entry_id
	ReasonCompensat = "COMPENSATION"     // 跨域失败补偿回补
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

// Level 积分等级:按累计获得积分匹配最高达标档。
type Level struct {
	LevelID   int64  `json:"levelId"`
	Name      string `json:"name" binding:"required"`
	MinPoints int64  `json:"minPoints" binding:"gte=0"`
	Status    string `json:"status"` // ENABLED/DISABLED
}

// Task 积分任务:ONE_TIME 终身一次,DAILY/MONTHLY 按周期幂等。
type Task struct {
	TaskID int64  `json:"taskId"`
	Code   string `json:"code" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Points int64  `json:"points" binding:"gt=0"`
	Period string `json:"period" binding:"required,oneof=ONE_TIME DAILY MONTHLY"`
	Status string `json:"status"` // ENABLED/DISABLED
}

// EarnRule 缴费自动积分规则:每元积分 + 起缴门槛 + 获得积分有效期天数。
type EarnRule struct {
	RuleID        int64  `json:"ruleId"`
	PointsPerYuan int    `json:"pointsPerYuan" binding:"gte=0"`
	MinCents      int64  `json:"minCents" binding:"gte=0"`
	ExpireDays    int    `json:"expireDays" binding:"gte=0"` // 0=永不过期
	Status        string `json:"status"`
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

	// ---- 等级(按累计获得积分匹配) ----
	ListLevels(ctx context.Context) ([]Level, error)
	CreateLevel(ctx context.Context, l Level) (int64, error)
	DisableLevel(ctx context.Context, levelID int64) error
	// TierOf 客户当前等级(累计获得积分最高达标档);无匹配返回 nil。
	TierOf(ctx context.Context, customerID int64) (*Level, error)

	// ---- 任务 ----
	ListTasks(ctx context.Context) ([]Task, error)
	CreateTask(ctx context.Context, t Task) (int64, error)
	DisableTask(ctx context.Context, taskID int64) error
	// CompleteTask 完成任务发放积分;周期内重复完成返回 ErrConflict(幂等不发)。
	CompleteTask(ctx context.Context, customerID, taskID int64) (int64, error)
	// TaskStatus 客户任务完成态(period_key → 完成时间)。
	TaskStatus(ctx context.Context, customerID int64) (map[int64]string, error)

	// ---- 缴费自动积分与退款回滚(以 payment_id 幂等) ----
	// EarnForPayment 缴费成功送积分;命中规则送,未配置/低于门槛返回 0。
	EarnForPayment(ctx context.Context, paymentID, customerID, amountCents int64) (int64, error)
	// RollbackPayment 退款冲销缴费积分;未送过/已冲销返回 0。
	RollbackPayment(ctx context.Context, paymentID, customerID int64) (int64, error)

	// EarnRuleOf 当前生效的缴费送积分规则(nil=未配置)。
	EarnRuleOf(ctx context.Context) (*EarnRule, error)
	SaveEarnRule(ctx context.Context, r EarnRule) (int64, error)

	// ExpireDue 过期清算:到期未消耗获得流水汇总扣减;返回清算客户数。
	ExpireDue(ctx context.Context) (int, error)
}
