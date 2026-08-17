package user

import (
	"context"
	"time"
)

// AccountOrgHistory 账号组织归属台账(调岗/调部门/调公司的时间段)。
type AccountOrgHistory struct {
	ID                int64
	AccountID         int64
	LegalEntityID     int64 // 0=空
	LegalEntityName   string
	DeptID            int64 // 0=空
	DeptName          string
	PostID            int64 // 0=空
	PostName          string
	Reason            string
	OperatorAccountID int64 // 0=空
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time // nil=至今
}

// OrgLedgerService 账号组织归属台账域服务口(阶段1)。
type OrgLedgerService interface {
	ListAccountOrgHistories(ctx context.Context, accountID int64) ([]AccountOrgHistory, error)
	AppendAccountOrgHistory(ctx context.Context, h AccountOrgHistory) (int64, error)
}
