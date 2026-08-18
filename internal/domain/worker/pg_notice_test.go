package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_Notices 契约:公告列表/发布/上下架切换。
func TestPGStore_Notices(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	pub := time.Date(2025, 8, 16, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT id, title, category, active, published_at FROM worker_notices`).
		WillReturnRows(mock.NewRows([]string{"id", "title", "category", "active", "published_at"}).
			AddRow(int64(1), "防水作业提示", "安全作业提醒", true, pub))
	mock.ExpectQuery(`INSERT INTO worker_notices`).
		WithArgs("物料配发说明", "物料公告", true).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
	mock.ExpectExec(`UPDATE worker_notices SET active = NOT active WHERE id=\$1`).
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	ctx := context.Background()

	list, err := s.ListNotices(ctx)
	if err != nil {
		t.Fatalf("ListNotices: %v", err)
	}
	if len(list) != 1 || !list[0].Active || list[0].Title != "防水作业提示" {
		t.Fatalf("list=%+v", list)
	}

	id, err := s.CreateNotice(ctx, Notice{Title: "物料配发说明", Category: "物料公告", Active: true})
	if err != nil || id != 11 {
		t.Fatalf("CreateNotice id=%d err=%v", id, err)
	}

	if err := s.ToggleNotice(ctx, 1); err != nil {
		t.Fatalf("ToggleNotice: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ToggleNoticeNotFound 契约:未命中返回 ErrNotFound。
func TestPGStore_ToggleNoticeNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectExec(`UPDATE worker_notices SET active = NOT active`).
		WithArgs(int64(99)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	err = NewPGStore(mock).ToggleNotice(context.Background(), 99)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}
