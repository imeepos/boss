package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_GetArrears 契约:按客户查欠费;未命中返回 ErrNotFound。
func TestPGStore_GetArrears(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, customer_id, amount, days, status FROM arrears`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"id", "customer_id", "amount", "days", "status"}).
				AddRow(int64(1), int64(1), 299.00, int32(30), "催收中"))

		s := NewPGStore(mock)
		a, err := s.GetArrears(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetArrears: %v", err)
		}
		if a.Amount != 299.00 || a.Days != 30 {
			t.Fatalf("a=%+v", a)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("未命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, customer_id, amount, days, status FROM arrears`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.GetArrears(context.Background(), 99)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// TestPGStore_UpsertArrears 契约:客户唯一快照,冲突更新。
func TestPGStore_UpsertArrears(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO arrears`).
		WithArgs(int64(1), 299.00, int32(30), "催收中").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.UpsertArrears(context.Background(), Arrears{CustomerID: 1, Amount: 299.00, Days: 30, Status: "催收中"})
	if err != nil {
		t.Fatalf("UpsertArrears: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ListStopResumeTasks 契约:按客户过滤停复机流水;customerID=0 返回全部。
func TestPGStore_ListStopResumeTasks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, customer_id, lo_account_id, action, status FROM stop_resume_tasks`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "customer_id", "lo_account_id", "action", "status"}).
			AddRow(int64(1), int64(1), int64(88), "STOP", "DONE"))

	s := NewPGStore(mock)
	got, err := s.ListStopResumeTasks(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListStopResumeTasks: %v", err)
	}
	if len(got) != 1 || got[0].Action != "STOP" || got[0].LoAccountID != 88 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_AppendStopResumeTask 契约:追加停复机流水并返回自增 id。
func TestPGStore_AppendStopResumeTask(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO stop_resume_tasks`).
		WithArgs(int64(1), int64(88), "RESUME", "PENDING").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.AppendStopResumeTask(context.Background(), StopResumeTask{CustomerID: 1, LoAccountID: 88, Action: "RESUME", Status: "PENDING"})
	if err != nil {
		t.Fatalf("AppendStopResumeTask: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
