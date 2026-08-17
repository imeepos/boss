package aaa

import (
	"context"
	"time"
)

// LoAccount LO 认证账号(客户 1:1,宽带认证与 QoS 生效载体)。
type LoAccount struct {
	ID              int64  `json:"id"`
	Loid            string `json:"loid"`
	CustomerID      int64  `json:"customerId"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	RegionID        int64  `json:"regionId"`
	RegionName      string `json:"regionName"`
	RegionPath      string `json:"regionPath"` // 区域 ltree 路径快照,空=未挂区域(承接区域调价覆盖)
	OfferID         int64  `json:"offerId"`
	QosTemplateID   int64  `json:"qosTemplateId"`
	Status          string `json:"status"` // ACTIVE/SUSPENDED/CLOSED
}

// CdrRecord 话单(计费原始记录,RADIUS Accounting 产出)。
type CdrRecord struct {
	ID            int64     `json:"id"`
	Loid          string    `json:"loid"`
	Username      string    `json:"username"`   // 空=无
	AcctStatus    int16     `json:"acctStatus"` // 1开始/2停止/3中间
	SessionID     string    `json:"sessionId"`
	SessionTime   int32     `json:"sessionTime"` // 秒
	InputOctets   int64     `json:"inputOctets"`
	OutputOctets  int64     `json:"outputOctets"`
	NasIP         string    `json:"nasIp"`
	BillingStatus string    `json:"billingStatus"` // UNBILLED/BILLED
	StartedAt     time.Time `json:"startedAt"`
}

// AuthLog 认证日志(认证成功/失败)。
type AuthLog struct {
	ID        int64     `json:"id"`
	Loid      string    `json:"loid"`
	Result    string    `json:"result"` // SUCCESS/FAILED
	CreatedAt time.Time `json:"createdAt"`
}

// AaaService AAA 认证计费域服务口(阶段7):LO 账号/话单/认证日志。
type AaaService interface {
	ListLoAccounts(ctx context.Context) ([]LoAccount, error)
	CreateLoAccount(ctx context.Context, a LoAccount) (int64, error)
	GetLoAccountByLoid(ctx context.Context, loid string) (*LoAccount, error)
	GetLoAccountByCustomer(ctx context.Context, customerID int64) (*LoAccount, error)

	AppendCdr(ctx context.Context, c CdrRecord) (int64, error)
	ListCdrs(ctx context.Context, loid string) ([]CdrRecord, error)

	AppendAuthLog(ctx context.Context, l AuthLog) (int64, error)
	ListAuthLogs(ctx context.Context, loid string) ([]AuthLog, error)
}
