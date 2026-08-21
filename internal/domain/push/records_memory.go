package push

// records 内存实现(测试替身)。

import (
	"context"
	"sort"
	"sync"
	"time"
)

// RecordsMemoryStore RecordsStore 内存实现。
type RecordsMemoryStore struct {
	mu     sync.Mutex
	items  []Record
	nextID int64
}

// NewRecordsMemoryStore 构造内存留痕。
func NewRecordsMemoryStore() *RecordsMemoryStore { return &RecordsMemoryStore{} }

// Record 追加一行留痕。
func (s *RecordsMemoryStore) Record(_ context.Context, r Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	r.ID = s.nextID
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	s.items = append(s.items, r)
	return nil
}

// ListBySubject 主体留痕(新→旧)。
func (s *RecordsMemoryStore) ListBySubject(_ context.Context, subjectType string, subjectID int64, limit int) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	out := []Record{}
	for i := len(s.items) - 1; i >= 0 && len(out) < limit; i-- {
		if s.items[i].SubjectType == subjectType && s.items[i].SubjectID == subjectID {
			out = append(out, s.items[i])
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}
