// notify 域内存实现:单测/无 PG 环境用;非并发安全(测试串行)。
package notify

import (
	"context"
	"sort"
	"time"
)

// MemStore 是 Service 的内存实现。
type MemStore struct {
	items []Item
	reads map[int64]map[int64]bool // notificationID → accountID 集合
	next  int64
}

// NewMemStore 构造内存实现。
func NewMemStore() *MemStore { return &MemStore{reads: map[int64]map[int64]bool{}} }

// Emit 幂等写入;同 (refType, refID, category) 已存在时跳过。
func (m *MemStore) Emit(_ context.Context, in Input) error {
	if !in.Valid() {
		return ErrInvalidInput
	}
	for _, it := range m.items {
		if it.RefType == in.RefType && it.RefID == in.RefID && it.Category == in.Category {
			return nil
		}
	}
	m.next++
	it := Item{
		ID: m.next, Category: in.Category, Level: in.Level, Title: in.Title,
		Content: in.Content, Link: in.Link, RefType: in.RefType, RefID: in.RefID,
		TargetRole: in.TargetRole, CreatedAt: "1970-01-01T00:00:00Z",
	}
	if in.Category == CategoryTodo && in.DueHours > 0 {
		it.DueAt = time.Now().Add(time.Duration(in.DueHours) * time.Hour).UTC().Format(time.RFC3339)
	}
	m.items = append(m.items, it)
	return nil
}

// Resolve 把匹配 ref 的未办条目置 resolved。
func (m *MemStore) Resolve(_ context.Context, refType, refID string) error {
	for i := range m.items {
		if m.items[i].RefType == refType && m.items[i].RefID == refID {
			if !m.items[i].Resolved {
				m.items[i].Resolved = true
				m.items[i].ResolvedAt = time.Now().UTC().Format(time.RFC3339)
			}
		}
	}
	return nil
}

// List 过滤清单(新→旧),含账号读状态;返回条目+过滤后总数。
func (m *MemStore) List(_ context.Context, role string, accountID int64, f Filter) ([]Item, int, error) {
	vis := m.visible(role, accountID, f)
	total := len(vis)
	if f.Offset > 0 {
		if f.Offset >= total {
			return []Item{}, total, nil
		}
		vis = vis[f.Offset:]
	}
	if f.Limit > 0 && f.Limit < len(vis) {
		vis = vis[:f.Limit]
	}
	out := make([]Item, len(vis))
	copy(out, vis)
	for i := range out {
		out[i].Read = m.readBy(out[i].ID, accountID)
	}
	return out, total, nil
}

// UnreadCount 未读数(resolved 未读不计,前端默认隐藏已办)。
func (m *MemStore) UnreadCount(_ context.Context, role string, accountID int64) (int, error) {
	n := 0
	for _, it := range m.visible(role, accountID, Filter{HideResolved: true}) {
		if !m.readBy(it.ID, accountID) {
			n++
		}
	}
	return n, nil
}

// MarkRead 批量已读;ids 为空=可见全部已读。
func (m *MemStore) MarkRead(_ context.Context, role string, accountID int64, ids []int64) error {
	if len(ids) == 0 {
		for _, it := range m.items {
			if m.matchRole(it, role) {
				m.mark(it.ID, accountID)
			}
		}
		return nil
	}
	allow := map[int64]bool{}
	for _, it := range m.items {
		if m.matchRole(it, role) {
			allow[it.ID] = true
		}
	}
	for _, id := range ids {
		if allow[id] {
			m.mark(id, accountID)
		}
	}
	return nil
}

// visible 角色可见 + 过滤器 + 新→旧排序。
func (m *MemStore) visible(role string, accountID int64, f Filter) []Item {
	out := make([]Item, 0, len(m.items))
	for _, it := range m.items {
		if !m.matchRole(it, role) {
			continue
		}
		if f.Category != "" && it.Category != f.Category {
			continue
		}
		if f.Level != "" && it.Level != f.Level {
			continue
		}
		if f.Unread && m.readBy(it.ID, accountID) {
			continue
		}
		if f.HideResolved && it.Resolved {
			continue
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

func (m *MemStore) matchRole(it Item, role string) bool {
	return it.TargetRole == "" || it.TargetRole == role
}

func (m *MemStore) readBy(id, accountID int64) bool { return m.reads[id][accountID] }

func (m *MemStore) mark(id, accountID int64) {
	if m.reads[id] == nil {
		m.reads[id] = map[int64]bool{}
	}
	m.reads[id][accountID] = true
}
