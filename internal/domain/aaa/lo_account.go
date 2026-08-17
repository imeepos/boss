package aaa

import (
	"context"
	"time"
)

// LoAccount LO 认证账号(客户 1:1,宽带认证与 QoS 生效载体)。
type LoAccount struct {
	ID              int64
	Loid            string
	CustomerID      int64
	LegalEntityID   int64
	LegalEntityName string
	RegionID        int64
	RegionName      string
	OfferID         int64
	QosTemplateID   int64
	Status          string // ACTIVE/SUSPENDED/CLOSED
}

// CdrRecord 话单(计费原始记录,RADIUS Accounting 产出)。
type CdrRecord struct {
	ID            int64
	Loid          string
	Username      string // 空=无
	AcctStatus    int16  // 1开始/2停止/3中间
	SessionID     string
	SessionTime   int32 // 秒
	InputOctets   int64
	OutputOctets  int64
	NasIP         string
	BillingStatus string // UNBILLED/BILLED
	StartedAt     time.Time
}

// AuthLog 认证日志(认证成功/失败)。
type AuthLog struct {
	ID        int64
	Loid      string
	Result    string // SUCCESS/FAILED
	CreatedAt time.Time
}

// AaaService AAA 认证计费域服务口(阶段7):LO 账号/话单/认证日志。
type AaaService interface {
	ListLoAccounts(ctx context.Context) ([]LoAccount, error)
	CreateLoAccount(ctx context.Context, a LoAccount) (int64, error)
	GetLoAccountByLoid(ctx context.Context, loid string) (*LoAccount, error)

	AppendCdr(ctx context.Context, c CdrRecord) (int64, error)
	ListCdrs(ctx context.Context, loid string) ([]CdrRecord, error)

	AppendAuthLog(ctx context.Context, l AuthLog) (int64, error)
	ListAuthLogs(ctx context.Context, loid string) ([]AuthLog, error)
}
