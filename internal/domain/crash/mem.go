package crash

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// MemStore 进程内崩溃日志存储(测试 + 轻量部署回退)。
type MemStore struct {
	mu    sync.Mutex
	seq   atomic.Int64
	logs  []Log
	clock func() time.Time
}

// NewMemStore 构造;无 clock 表示用 time.Now。
func NewMemStore() *MemStore { return &MemStore{clock: time.Now} }

// SetClock 注入时间(测试用)。
func (m *MemStore) SetClock(f func() time.Time) { m.clock = f }

func (m *MemStore) Insert(_ context.Context, l Log) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq.Add(1)
	l.ID = m.seq.Load()
	if l.SubjectType == "" {
		l.SubjectType = "worker"
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = m.clock()
	}
	l.Log = Clip(l.Log)
	m.logs = append(m.logs, l)
	return nil
}

func (m *MemStore) ListRecent(_ context.Context, limit int) ([]Log, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Log, len(m.logs))
	copy(out, m.logs)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
