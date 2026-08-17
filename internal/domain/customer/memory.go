package customer

import (
	"context"
	"sync"
)

// MemoryService 阶段2骨架参考实现:进程内 map,仅供单测与协议链路验证。
// 阶段2落地替换为 DB(repo)。Create 按递增 id 分配,并发安全。
type MemoryService struct {
	mu  sync.RWMutex
	m   map[int64]Customer
	seq int64
}

// NewMemoryService 创建内存客户服务。
func NewMemoryService() *MemoryService {
	return &MemoryService{m: make(map[int64]Customer)}
}

// Create 建档并分配自增 id。
func (s *MemoryService) Create(ctx context.Context, c Customer) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	c.ID = s.seq
	s.m[c.ID] = c
	return c.ID, nil
}

// Get 按 id 查客户。
func (s *MemoryService) Get(ctx context.Context, id int64) (*Customer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.m[id]
	if !ok {
		return nil, ErrCustomerNotFound
	}
	cp := c
	return &cp, nil
}

// List 按条件过滤,未匹配返回空片(非 nil)。
func (s *MemoryService) List(ctx context.Context, q CustomerQuery) ([]Customer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Customer, 0, len(s.m))
	for _, c := range s.m {
		if !match(c, q) {
			continue
		}
		out = append(out, c)
	}
	// 简单分页:offset 截断,limit 截长(limit<=0 返回全部匹配)。
	if q.Offset > 0 && q.Offset < len(out) {
		out = out[q.Offset:]
	} else if q.Offset > 0 {
		out = out[:0]
	}
	if q.Limit > 0 && q.Limit < len(out) {
		out = out[:q.Limit]
	}
	return out, nil
}

func match(c Customer, q CustomerQuery) bool {
	if q.NameKeyword != "" && !contains(c.Name, q.NameKeyword) {
		return false
	}
	if q.Phone != "" && c.Phone != q.Phone {
		return false
	}
	if q.Status != "" && c.ServiceStatus != q.Status {
		return false
	}
	return true
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
