package audit

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

// TestPGWriter_Write 契约:写入一条审计,detail 序列化为 JSON,空 ip 归 NULL。
func TestPGWriter_Write(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO audit_logs`).
		WithArgs(int64(1), "状态变更", "order", "ORD-1", `{"from":"PENDING","to":"RESERVED"}`, nil).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	w := NewPGWriter(mock)
	err = w.Write(context.Background(), Event{
		AccountID: 1, Action: "状态变更", TargetType: "order", TargetID: "ORD-1",
		Detail: map[string]any{"from": "PENDING", "to": "RESERVED"},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGWriter_List 契约:按人/类型/操作过滤查询。
func TestPGWriter_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT l\.id, l\.account_id, l\.action, l\.target_type, COALESCE\(l\.target_id, ''\), l\.detail::text, COALESCE\(l\.ip::text, ''\), l\.created_at, COALESCE\(a\.real_name, ''\)`).
		WithArgs(int64(1), "状态变更", "order", 1<<30, 0).
		WillReturnRows(mock.NewRows([]string{"id", "account_id", "action", "target_type", "target_id", "detail", "ip", "created_at", "operator"}).
			AddRow(int64(1), int64(1), "状态变更", "order", "ORD-1", `{"a":1}`, "", ts, "管理员"))

	w := NewPGWriter(mock)
	got, err := w.List(context.Background(), Query{AccountID: 1, Action: "状态变更", TargetType: "order"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].TargetID != "ORD-1" || got[0].Operator != "管理员" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
