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
	Status          string `json:"status"`      // ACTIVE/SUSPENDED/CLOSED
	BillingMode     string `json:"billingMode"` // PREPAID/POSTPAID(000102);空回退 POSTPAID
}

// 付费模式枚举(terms.md §4):订购关系权威态,预付费不进月度出账。
const (
	BillingModePrepaid  = "PREPAID"
	BillingModePostpaid = "POSTPAID"
)

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

// AuthLog 认证日志(认证成功/失败;失败原因码见 auth.go 常量,000194)。
type AuthLog struct {
	ID         int64     `json:"id"`
	Loid       string    `json:"loid"`
	Result     string    `json:"result"`     // SUCCESS/FAILED
	FailReason string    `json:"failReason"` // 失败原因码,空=成功或存量(000194)
	CreatedAt  time.Time `json:"createdAt"`
}

// AdminScope 管理端 AAA 查询范围；空公司和区域表示全集团。
type AdminScope struct {
	LegalEntityID int64
	RegionScope   string
}

// AdminPage AAA 管理列表分页参数。
type AdminPage struct {
	Page     int
	PageSize int
	Keyword  string
	Status   string
	Loid     string
}

// AdminPageResult 分页数据与总数。
type AdminPageResult[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// AdminSummary AAA 管理总览聚合结果。
type AdminSummary struct {
	Accounts    int `json:"accounts"`
	Active      int `json:"active"`
	Suspended   int `json:"suspended"`
	Closed      int `json:"closed"`
	Cdrs        int `json:"cdrs"`
	Unbilled    int `json:"unbilled"`
	AuthSuccess int `json:"authSuccess"`
	AuthFailed  int `json:"authFailed"`
}

// AdminQueryService 是 AAA 管理端分页查询扩展，不改变三端业务接口。
type AdminQueryService interface {
	GetAdminSummary(ctx context.Context, scope AdminScope) (AdminSummary, error)
	ListLoAccountsPage(ctx context.Context, q AdminPage, scope AdminScope) (AdminPageResult[LoAccount], error)
	ListCdrsPage(ctx context.Context, q AdminPage, scope AdminScope) (AdminPageResult[CdrRecord], error)
	ListAuthLogsPage(ctx context.Context, q AdminPage, scope AdminScope) (AdminPageResult[AuthLog], error)
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

	// SuspendLoAccount/ResumeLoAccount 停复机即时生效(状态迁移,仅合法前置态可迁)。
	SuspendLoAccount(ctx context.Context, loAccountID int64) error
	ResumeLoAccount(ctx context.Context, loAccountID int64) error
}
