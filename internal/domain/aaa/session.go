package aaa

import (
	"context"
	"errors"
	"time"
)

// ErrSessionNotFound 在线会话不存在(admin 按 id 强制下线未命中)。
var ErrSessionNotFound = errors.New("aaa: session not found")

// 在线会话状态枚举(迁移 000194 CHECK 约束;fields.md 同步)。
const (
	SessionOnline         = "ONLINE"          // 在线
	SessionPendingOffline = "PENDING_OFFLINE" // 已下发下线未确认,重试中
	SessionOffline        = "OFFLINE"         // 已下线(终态)
	SessionOfflineFailed  = "OFFLINE_FAILED"  // 下线失败:重试耗尽(终态)
)

// 会话/话单关闭原因词汇表(aaa_online_sessions.close_reason 与 cdrs.close_reason 同词)。
const (
	CloseReasonAcctStop      = "ACCT_STOP"      // 计账 Stop 正常关闭
	CloseReasonCoA           = "COA_DISCONNECT" // CoA/Disconnect 下发成功关闭
	CloseReasonZombie        = "ZOMBIE_REAP"    // 僵尸清理(超时未更新)
	CloseReasonOfflineFailed = "OFFLINE_FAILED" // 下线重试耗尽
)

// 认证失败原因(auth_logs.reason)。
const AuthFailReasonConcurrent = "CONCURRENT_LIMIT"

// RADIUS Acct-Status-Type(rfc2866):会话维护按此分流。
const (
	AcctStatusStart   = 1
	AcctStatusStop    = 2
	AcctStatusInterim = 3
)

// SessionRecord 在线会话(radacct 模式):计账 Start 建/Interim 累加/Stop 关。
type SessionRecord struct {
	ID                 int64      `json:"id"`
	Loid               string     `json:"loid"`
	SessionID          string     `json:"sessionId"`
	NasIP              string     `json:"nasIp"`
	StartedAt          time.Time  `json:"startedAt"`
	LastUpdate         time.Time  `json:"lastUpdate"`
	InputOctets        int64      `json:"inputOctets"`
	OutputOctets       int64      `json:"outputOctets"`
	Status             string     `json:"status"`
	DisconnectAttempts int        `json:"disconnectAttempts"`
	CloseReason        string     `json:"closeReason"`
	ClosedAt           *time.Time `json:"closedAt"`
}

// SessionMaintainer 计账链路会话维护口(RADIUS/gRPC 计账入口消费;PGStore 实现)。
type SessionMaintainer interface {
	// StartSession 建 ONLINE 会话;(loid, session_id) 重复 Start 幂等去重,返回是否新建。
	StartSession(ctx context.Context, rec SessionRecord) (bool, error)
	// TouchSessionTraffic Interim:累加上下行流量并刷新最近更新时间。
	TouchSessionTraffic(ctx context.Context, loid, sessionID string, inputOctets, outputOctets int64) error
	// StopSession 关闭会话;返回会话是否存在(孤儿 Stop 返回 false 不报错)。
	StopSession(ctx context.Context, loid, sessionID, closeReason string) (bool, error)
}

// SessionGate 并发会话闸口(认证链路消费;PGStore 实现)。
type SessionGate interface {
	// AllowNewSession 判定 LOID 是否可再建会话;返回(放行, 当前在线占用数, err)。
	AllowNewSession(ctx context.Context, loid string, limit int) (bool, int, error)
}

// DisconnectSender CoA/DM 下发口(RFC 5176 Disconnect;radius.CoAClient 实现)。
type DisconnectSender interface {
	// SendDisconnect 向 NAS 发 Disconnect-Request;收到 ACK 返回 nil,超时/NAK/网络错误返回 err。
	SendDisconnect(ctx context.Context, nasIP, loid, sessionID string) error
}

// SessionPage 在线会话管理分页参数(loid/nasIp/status 过滤)。
type SessionPage struct {
	Page     int
	PageSize int
	Loid     string
	NasIP    string
	Status   string
}

// SessionAdminQuery 管理端在线会话分页查询口(PGStore 实现,窄口断言)。
type SessionAdminQuery interface {
	ListSessionsPage(ctx context.Context, q SessionPage, scope AdminScope) (AdminPageResult[SessionRecord], error)
}

// SessionForceOffline 强制下线操作口(SessionControlService 实现;admin 路由消费)。
type SessionForceOffline interface {
	// ForceOfflineSessionByID 对单条在线会话下发 Disconnect。
	ForceOfflineSessionByID(ctx context.Context, sessionID int64, reason string) error
	// ForceOfflineLoid 对 LOID 全部在线会话下发 Disconnect,返回处理条数。
	ForceOfflineLoid(ctx context.Context, loid, reason string) (int, error)
}

// SuspendOfflineLink 停复机联动口:停机成功自动下线该 LOID 全部在线会话。
type SuspendOfflineLink interface {
	SuspendWithOffline(ctx context.Context, loAccountID int64) error
}
