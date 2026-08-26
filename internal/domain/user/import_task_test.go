package user

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// 回归:时间筛选为空时必须以 NULL 参数下发(空串参与 ::timestamptz 会 22007,真库实测)。
func TestPGStore_ListImportTasks_EmptyTimeParamsPassNull(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT t.id, t.kind`).
		WithArgs("entity:department", "admin", nil, nil).
		WillReturnRows(mock.NewRows([]string{"id", "kind", "operator", "total", "imported", "failed", "skipped", "detail", "created_at"}))

	s := NewPGStore(mock)
	if _, err := s.ListImportTasks(context.Background(), "entity:department", "admin", "", ""); err != nil {
		t.Fatalf("ListImportTasks: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPGStore_RecordImportTask_UsesIdempotencyKey(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO import_tasks.*ON CONFLICT \(client_key\)`).
		WithArgs("entity:department", int64(7), 2, 1, 1, 0, "{}", "department-run-1").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	if err := NewPGStore(mock).RecordImportTask(context.Background(), "entity:department", 7, 2, 1, 1, 0, nil, "department-run-1"); err != nil {
		t.Fatalf("RecordImportTask: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// 非空时间以字符串参数下发,由 SQL 侧显式 ::timestamptz 转换。
func TestPGStore_ListImportTasks_TimeParamsPassedThrough(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT t.id, t.kind`).
		WithArgs("", "", "2026-09-01T00:00:00Z", "2026-09-04T00:00:00Z").
		WillReturnRows(mock.NewRows([]string{"id", "kind", "operator", "total", "imported", "failed", "skipped", "detail", "created_at"}))

	s := NewPGStore(mock)
	if _, err := s.ListImportTasks(context.Background(), "", "", "2026-09-01T00:00:00Z", "2026-09-04T00:00:00Z"); err != nil {
		t.Fatalf("ListImportTasks: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
