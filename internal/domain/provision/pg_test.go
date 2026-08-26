package provision

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

func TestPGStore_ListTemplates(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, legal_entity_id, code, name, content, version, status, updated_at FROM provision_templates`).
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "code", "name", "content", "version", "status", "updated_at"}).
			AddRow(int64(1), int64(1), "TPL-FTTH", "FTTH标准开通", []byte(`{}`), int32(1), "ENABLED", ts))

	s := NewPGStore(mock)
	got, err := s.ListTemplates(context.Background())
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(got) != 1 || got[0].Code != "TPL-FTTH" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateTemplate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK validation: legal entity exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO provision_templates`).
		WithArgs(int64(1), "TPL-GPON", "GPON标准开通", `{}`, "ENABLED").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateTemplate(context.Background(), Template{LegalEntityID: 1, Code: "TPL-GPON", Name: "GPON标准开通"})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListTasks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, task_no, order_id, stage_event, lo_account_id, template_id, status FROM provision_tasks`).
		WillReturnRows(mock.NewRows([]string{"id", "task_no", "order_id", "stage_event", "lo_account_id", "template_id", "status"}).
			AddRow(int64(1), "TASK-1", int64(9), "preConfigOLT", int64(88), int64(1), "DONE"))

	s := NewPGStore(mock)
	got, err := s.ListTasks(context.Background())
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(got) != 1 || got[0].Status != "DONE" || got[0].TaskNo != "TASK-1" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateTask(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO provision_tasks`).
		WithArgs("TASK-20260817-01", int64(9), "preConfigOLT", int64(88), int64(1), "PENDING").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateTask(context.Background(), Task{
		TaskNo: "TASK-20260817-01", OrderID: 9, StageEvent: "preConfigOLT",
		LoAccountID: 88, TemplateID: 1, Status: "PENDING",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GetTaskByNo(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		cols := []string{"id", "task_no", "order_id", "stage_event", "lo_account_id", "template_id", "status"}
		mock.ExpectQuery(`SELECT id, task_no, order_id, stage_event, lo_account_id, template_id, status FROM provision_tasks WHERE task_no = \$1`).
			WithArgs("TASK-1").
			WillReturnRows(mock.NewRows(cols).AddRow(int64(1), "TASK-1", int64(9), "preConfigOLT", int64(88), int64(1), "FAILED"))

		s := NewPGStore(mock)
		got, err := s.GetTaskByNo(context.Background(), "TASK-1")
		if err != nil {
			t.Fatalf("GetTaskByNo: %v", err)
		}
		if got.Status != "FAILED" || got.OrderID != 9 {
			t.Fatalf("got=%+v", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("未命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, task_no, order_id, stage_event, lo_account_id, template_id, status FROM provision_tasks WHERE task_no = \$1`).
			WithArgs("TASK-NOPE").
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		if _, err := s.GetTaskByNo(context.Background(), "TASK-NOPE"); !errors.Is(err, ErrTaskNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestPGStore_ListLogs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "task_id", "resource_id", "resource_code", "template_id", "template_code", "result", "retries", "created_at"}
	mock.ExpectQuery(`SELECT id, task_id, resource_id, COALESCE\(resource_code, ''\), template_id, COALESCE\(template_code, ''\), result, retries, created_at`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(10), "OLT-01", int64(1), "TPL-FTTH", "SUCCESS", int16(0), ts))

	s := NewPGStore(mock)
	got, err := s.ListLogs(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListLogs: %v", err)
	}
	if len(got) != 1 || got[0].Result != "SUCCESS" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendLog(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO provision_logs`).
		WithArgs(int64(1), int64(10), "OLT-01", int64(1), "TPL-FTTH", "FAILED", int16(1)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.AppendLog(context.Background(), Log{
		TaskID: 1, ResourceID: 10, ResourceCode: "OLT-01", TemplateID: 1, TemplateCode: "TPL-FTTH", Result: "FAILED", Retries: 1,
	})
	if err != nil {
		t.Fatalf("AppendLog: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
