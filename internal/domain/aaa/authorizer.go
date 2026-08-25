package aaa

import (
	"context"
	"errors"
)

// ErrNotFound 认证档案不存在(认证失败/无授权)。
var ErrNotFound = errors.New("aaa: profile not found")

// ErrSuspended 账号已停(停复机:停机后拒绝认证接入)。
var ErrSuspended = errors.New("aaa: profile suspended")

// ErrClosed 账号已注销(CLOSED:永久拒绝认证接入,与停服语义分离)。
var ErrClosed = errors.New("aaa: profile closed")

// Decision 一次授权决策结果(RADIUS Access-Accept 的属性来源)。
type Decision struct {
	LOID        string
	Authorize   bool   // 是否放行
	Bandwidth   string // 带宽模板属性
	QosTemplate string // QoS 模板编码(契约 aaa/v1 GetAuthorization 透出)
	SessionTTL  int    // Session-Timeout 属性
}

// Authorizer 授权决策口。
// 实现约定(技术栈方案 3.3/3.4):
//   - 授权结果写 Redis(缓存 key=LOID,TTL 60s,停复机删 key 秒级生效)
//   - 停机(StatusSuspended)→ Authorize=false;未停→ 放行并下发带宽模板。
//
// 权威状态判定统一走本接口,不做 RADIUS 协议内二次判断。
type Authorizer interface {
	// Decide 判定 LOID 是否可接入并产出授权属性。
	Decide(ctx context.Context, loid string) (Decision, error)
}
