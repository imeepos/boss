package device

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_ListAlarms 契约:按设备过滤告警;resourceID=0 返回全部。
func TestPGStore_ListAlarms(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, alarm_no, level, source, content, COALESCE\(resource_id, 0\), status, created_at`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{
			"id", "alarm_no", "level", "source", "content", "resource_id", "status", "created_at",
		}).AddRow(int64(1), "ALM-001", "CRITICAL", "device", "光功率过低", int64(1), "OPEN", ts))

	s := NewPGStore(mock)
	got, err := s.ListAlarms(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListAlarms: %v", err)
	}
	if len(got) != 1 || got[0].AlarmNo != "ALM-001" || got[0].Level != "CRITICAL" || got[0].Status != "OPEN" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_CreateAlarm 契约:新增告警并返回自增 id;resource=0 写 NULL(平台级告警)。
func TestPGStore_CreateAlarm(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO alarms`).
		WithArgs("ALM-001", "WARNING", "quadlink", "四码不一致", nil, "OPEN", ts).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.CreateAlarm(context.Background(), Alarm{
		AlarmNo: "ALM-001", Level: "WARNING", Source: "quadlink", Content: "四码不一致", Status: "OPEN", CreatedAt: ts,
	})
	if err != nil {
		t.Fatalf("CreateAlarm: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_CreateAlarm_ResourceGate 契约:resource_id 非零时资源必须存在
// (曾 12 条悬空资源告警,audit 2026-08-25),缺失拒建。
func TestPGStore_CreateAlarm_ResourceGate(t *testing.T) {
	t.Run("资源存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`INSERT INTO alarms`).
			WithArgs("ALM-002", "CRITICAL", "device", "光功率过低", int64(7), "OPEN", ts).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(4)))
		if _, err := NewPGStore(mock).CreateAlarm(context.Background(), Alarm{
			AlarmNo: "ALM-002", Level: "CRITICAL", Source: "device", Content: "光功率过低",
			ResourceID: 7, Status: "OPEN", CreatedAt: ts,
		}); err != nil {
			t.Fatalf("CreateAlarm: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("资源不存在拒建", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(999)).
			WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))
		_, err := NewPGStore(mock).CreateAlarm(context.Background(), Alarm{
			AlarmNo: "ALM-X", Level: "CRITICAL", Source: "device", Content: "x",
			ResourceID: 999, Status: "OPEN", CreatedAt: ts,
		})
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})
}

// TestPGStore_UpdateAlarmStatus 契约:确认/关闭告警。
func TestPGStore_UpdateAlarmStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE alarms SET status`).
		WithArgs("CLOSED", int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	if err := s.UpdateAlarmStatus(context.Background(), 1, "CLOSED"); err != nil {
		t.Fatalf("UpdateAlarmStatus: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
