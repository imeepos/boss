package asset

// P2-T4 事件流查询单测:倒序、limit 兜底、action 白名单过滤、before_id 游标、404 语义。
// pgxmock 正则为子串匹配,锚点用无特殊字符短片段(避 $/() 转义)。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// eventQueryRows 组一行行事件(倒序场景按传入顺序返回,断言透传保序)。
func eventQueryRows(mock pgxmock.PgxPoolIface, ids ...int64) *pgxmock.Rows {
	r := mock.NewRows([]string{"id", "event_id", "tag_id", "asset_id", "action", "actor_account_id", "detail", "changed", "created_at"})
	for _, id := range ids {
		r.AddRow(id, "ev-uuid", int64(1), int64(2), "UNBIND", int64(3), "换新回收",
			map[string]any{"bound_asset_id": []int64{2, 0}}, time.Now())
	}
	return r
}

func TestPGStore_ListTagEvents(t *testing.T) {
	t.Run("倒序+limit 透传", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("FROM tags WHERE id").
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("FROM tag_events WHERE tag_id").
			WithArgs(int64(5), int64(20)).
			WillReturnRows(eventQueryRows(mock, 12, 7))

		s := NewPGStore(mock)
		got, err := s.ListTagEvents(context.Background(), 5, 20, 0, nil)
		if err != nil {
			t.Fatalf("ListTagEvents: %v", err)
		}
		if len(got) != 2 || got[0].ID != 12 || got[1].ID != 7 {
			t.Fatalf("倒序结果失真: %v", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("标签不存在 → ErrNotFound", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("FROM tags WHERE id").
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

		s := NewPGStore(mock)
		if _, err := s.ListTagEvents(context.Background(), 5, 0, 0, nil); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("action 白名单过滤+before 游标+limit 兜底", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("FROM tags WHERE id").
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("AND action = ANY").
			WithArgs(int64(5), pgxmock.AnyArg(), int64(100), int64(50)).
			WillReturnRows(eventQueryRows(mock, 3))

		s := NewPGStore(mock)
		got, err := s.ListTagEvents(context.Background(), 5, 0, 100, []string{"BIND", "RECYCLE"})
		if err != nil {
			t.Fatalf("ListTagEvents: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("行数失真: %d", len(got))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("limit 超上限被夹到 100", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("FROM tags WHERE id").
			WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("FROM tag_events WHERE tag_id").
			WithArgs(int64(5), int64(100)).
			WillReturnRows(eventQueryRows(mock))

		s := NewPGStore(mock)
		if _, err := s.ListTagEvents(context.Background(), 5, 9999, 0, nil); err != nil {
			t.Fatalf("ListTagEvents: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_ListAssetEvents(t *testing.T) {
	t.Run("按资产过滤+倒序", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("FROM assets WHERE id").
			WithArgs(int64(2)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("FROM tag_events WHERE asset_id").
			WithArgs(int64(2), int64(50)).
			WillReturnRows(eventQueryRows(mock, 21, 4))

		s := NewPGStore(mock)
		got, err := s.ListAssetEvents(context.Background(), 2, 0, 0, nil)
		if err != nil {
			t.Fatalf("ListAssetEvents: %v", err)
		}
		if len(got) != 2 || got[0].ID != 21 {
			t.Fatalf("倒序结果失真: %v", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("资产不存在 → ErrNotFound", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery("FROM assets WHERE id").
			WithArgs(int64(2)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

		s := NewPGStore(mock)
		if _, err := s.ListAssetEvents(context.Background(), 2, 0, 0, nil); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

// ValidEventAction 白名单:写侧三动作 + CREATE(000189 存量回填)可过滤。
func TestValidEventAction(t *testing.T) {
	for _, a := range []string{"BIND", "UNBIND", "RECYCLE", "CREATE"} {
		if !ValidEventAction(a) {
			t.Errorf("%s should be valid", a)
		}
	}
	for _, a := range []string{"", "bind", "DELETE", "UPDATE"} {
		if ValidEventAction(a) {
			t.Errorf("%q should be invalid", a)
		}
	}
}
