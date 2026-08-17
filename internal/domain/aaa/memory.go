package aaa

import (
	"context"
	"sync"
)

// MemoryAuthorizer 阶段7骨架参考实现:进程内 map,仅供协议链路验证。
// 阶段7落地替换为 DB(ProfileRepo)+Redis(授权缓存 TTL 60s)实现。
type MemoryAuthorizer struct {
	mu sync.RWMutex
	m  map[string]Profile
}

// NewMemoryAuthorizer 创建内存授权器。
func NewMemoryAuthorizer(profiles []Profile) *MemoryAuthorizer {
	a := &MemoryAuthorizer{m: make(map[string]Profile)}
	for _, p := range profiles {
		a.m[p.LOID] = p
	}
	return a
}

// Decide 判定 LOID 是否放行并产出授权属性。
func (a *MemoryAuthorizer) Decide(ctx context.Context, loid string) (Decision, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	p, ok := a.m[loid]
	if !ok {
		return Decision{}, ErrNotFound
	}
	if p.Status != StatusActive {
		return Decision{LOID: loid, Authorize: false}, ErrSuspended
	}
	return Decision{
		LOID:       loid,
		Authorize:  true,
		Bandwidth:  p.Bandwidth,
		SessionTTL: p.SessionTTL,
	}, nil
}
