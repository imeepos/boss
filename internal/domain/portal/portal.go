// Package portal 用户端/师傅端门户状态域:验证码、门户账号、偏好、站内消息、钱包、单号。
// 落库为唯一事实源(取代 handler 进程内 map);短信通道接入属基础设施配置,存储先行。
package portal

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound 目标记录不存在(账号/偏好等)。
var ErrNotFound = errors.New("portal: not found")

// Account 门户账号:手机号登录,与 customers 主档(或合成 ID)1:1。
type Account struct {
	Phone             string
	CustomerID        int64
	PasswordHash      string
	PasswordUpdatedAt time.Time
}

// Prefs 通知偏好与语言。
type Prefs struct {
	Notify   map[string]any
	Language string
}

// Message 站内消息:payload 由端契约定义,read 为已读标记。
type Message struct {
	ID         int64
	CustomerID int64
	Payload    map[string]any
	Read       bool
	CreatedAt  time.Time
}

// Service 门户状态服务接口(handler 依赖此抽象,测试以内存实现替身)。
type Service interface {
	// IssueSms 签发验证码(5 分钟有效);ConsumeSms 一次性校验消费。
	IssueSms(ctx context.Context, phone, scene string) error
	ConsumeSms(ctx context.Context, phone, scene, code string) (bool, error)

	// NextSyntheticCustomerID 未关联主档的注册账号发隔离空间合成 ID(负数段,见 000057 迁移)。
	NextSyntheticCustomerID(ctx context.Context) (int64, error)

	// UpsertAccount 创建/更新账号(password 明文入参,内部 bcrypt);改密同一入口。
	UpsertAccount(ctx context.Context, phone, password string, customerID int64) (*Account, error)
	AccountByPhone(ctx context.Context, phone string) (*Account, error)
	AccountByCustomer(ctx context.Context, customerID int64) (*Account, error)
	VerifyPassword(ctx context.Context, phone, password string) (bool, error)
	RebindPhone(ctx context.Context, customerID int64, newPhone string) error

	GetPrefs(ctx context.Context, customerID int64) (*Prefs, error)
	SavePrefs(ctx context.Context, customerID int64, notify map[string]any, language string) error

	Messages(ctx context.Context, customerID int64) ([]Message, error)
	PutMessage(ctx context.Context, customerID int64, payload map[string]any) error
	MarkAllRead(ctx context.Context, customerID int64) error
	HasUnread(ctx context.Context, customerID int64) (bool, error)

	Balance(ctx context.Context, customerID int64) (float64, error)
	AdjustBalance(ctx context.Context, customerID int64, delta float64) error

	// AutoPay/SetAutoPay 自动缴费开通状态(portal_billing_prefs 落库)。
	AutoPay(ctx context.Context, customerID int64) (bool, error)
	SetAutoPay(ctx context.Context, customerID int64, enabled bool) error

	// NextNo 单号:kind ∈ PAY/CHG/TKT/MSG,返回 "<kind>-<自增>"。
	NextNo(ctx context.Context, kind string) (string, error)
}
