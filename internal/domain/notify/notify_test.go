// notify 域 Service 契约测试(内存实现;PG 实现同契约,联调用 e2e)。
package notify

import (
	"context"
	"errors"
	"testing"
)

func emitT(t *testing.T, s Service, in Input) {
	t.Helper()
	if err := s.Emit(context.Background(), in); err != nil {
		t.Fatalf("Emit: %v", err)
	}
}

func TestEmitIdempotent(t *testing.T) {
	s := NewMemStore()
	in := Input{Category: CategoryTask, Title: "导入完成", RefType: "importer", RefID: "7"}
	emitT(t, s, in)
	emitT(t, s, in)
	items, total, err := s.List(context.Background(), "sysadmin", 1, Filter{})
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("want 1 item, got total=%d len=%d err=%v", total, len(items), err)
	}
}

func TestEmitInvalid(t *testing.T) {
	s := NewMemStore()
	err := s.Emit(context.Background(), Input{Category: "spam", Title: "x", RefType: "a", RefID: "1"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestReadReceiptsPerAccount(t *testing.T) {
	s := NewMemStore()
	emitT(t, s, Input{Category: CategoryTask, Title: "n1", RefType: "r", RefID: "1"})
	// 账号 1 全部已读;账号 2 仍未读
	if err := s.MarkRead(context.Background(), "sysadmin", 1, nil); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	n1, _ := s.UnreadCount(context.Background(), "sysadmin", 1)
	n2, _ := s.UnreadCount(context.Background(), "sysadmin", 2)
	if n1 != 0 || n2 != 1 {
		t.Fatalf("want acct1=0 acct2=1, got %d/%d", n1, n2)
	}
}

func TestRoleVisibility(t *testing.T) {
	s := NewMemStore()
	emitT(t, s, Input{Category: CategoryTask, Title: "all", RefType: "r", RefID: "1"})
	emitT(t, s, Input{Category: CategoryTask, Title: "ops-only", RefType: "r", RefID: "2", TargetRole: "ops"})
	_, totalOps, _ := s.List(context.Background(), "ops", 1, Filter{})
	_, totalOther, _ := s.List(context.Background(), "analyst", 1, Filter{})
	if totalOps != 2 || totalOther != 1 {
		t.Fatalf("want ops=2 analyst=1, got %d/%d", totalOps, totalOther)
	}
}

func TestResolveHidesTodo(t *testing.T) {
	s := NewMemStore()
	emitT(t, s, Input{Category: CategoryTodo, Title: "待审核", RefType: "worker_reg", RefID: "9"})
	if err := s.Resolve(context.Background(), "worker_reg", "9"); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	n, _ := s.UnreadCount(context.Background(), "sysadmin", 1)
	if n != 0 {
		t.Fatalf("resolved todo should not count unread, got %d", n)
	}
	items, _, _ := s.List(context.Background(), "sysadmin", 1, Filter{Category: CategoryTodo})
	if len(items) != 1 || !items[0].Resolved {
		t.Fatalf("resolve should keep row but mark resolved: %+v", items)
	}
}

// 回归(实名驳回→重提场景):已办结 todo 再次 Emit 同键必须复活为未办、
// 刷新标题并清读回执;行 id 不变(不产生第二条)。task 类同契约。
func TestEmitRevivesResolvedTodo(t *testing.T) {
	s := NewMemStore()
	ctx := context.Background()
	in := Input{Category: CategoryTodo, Title: "实名待审核:张*生", RefType: "realname", RefID: "customer/88"}
	emitT(t, s, in)
	if err := s.Resolve(ctx, "realname", "customer/88"); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	_ = s.MarkRead(ctx, "sysadmin", 1, nil)

	emitT(t, s, Input{Category: CategoryTodo, Level: LevelWarn,
		Title: "实名待审核:张*生(重提)", RefType: "realname", RefID: "customer/88"})

	items, total, err := s.List(ctx, "sysadmin", 1, Filter{HideResolved: true})
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("revive should yield exactly one unresolved item, total=%d len=%d err=%v", total, len(items), err)
	}
	it := items[0]
	if it.Resolved || it.ID == 0 {
		t.Fatalf("revived item wrong state: %+v", it)
	}
	if it.Title != "实名待审核:张*生(重提)" || it.Level != LevelWarn {
		t.Fatalf("revive should refresh content: %+v", it)
	}
	un, _ := s.UnreadCount(ctx, "sysadmin", 1)
	if un != 1 {
		t.Fatalf("revive must clear read receipts, unread=%d", un)
	}
}

func TestFilterAndPaging(t *testing.T) {
	s := NewMemStore()
	emitT(t, s, Input{Category: CategoryTask, Level: LevelWarn, Title: "w", RefType: "r", RefID: "1"})
	emitT(t, s, Input{Category: CategoryTask, Level: LevelInfo, Title: "i1", RefType: "r", RefID: "2"})
	emitT(t, s, Input{Category: CategoryTask, Level: LevelInfo, Title: "i2", RefType: "r", RefID: "3"})
	items, total, _ := s.List(context.Background(), "sysadmin", 1, Filter{Level: LevelInfo, Limit: 1})
	if total != 2 || len(items) != 1 || items[0].Title != "i2" {
		t.Fatalf("filter/paging wrong: total=%d items=%+v", total, items)
	}
	// 未读过滤:读掉 i2 后仅剩 i1
	_ = s.MarkRead(context.Background(), "sysadmin", 1, []int64{items[0].ID})
	un, _, _ := s.List(context.Background(), "sysadmin", 1, Filter{Level: LevelInfo, Unread: true})
	if len(un) != 1 || un[0].Title != "i1" {
		t.Fatalf("unread filter wrong: %+v", un)
	}
}

func TestMarkReadRespectsRole(t *testing.T) {
	s := NewMemStore()
	emitT(t, s, Input{Category: CategoryTask, Title: "ops-only", RefType: "r", RefID: "1", TargetRole: "ops"})
	// analyst 无权读 ops-only:全部已读不生效
	_ = s.MarkRead(context.Background(), "analyst", 5, nil)
	n, _ := s.UnreadCount(context.Background(), "ops", 6)
	if n != 1 {
		t.Fatalf("ops should still see 1 unread, got %d", n)
	}
}
