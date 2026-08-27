package openplat

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_CreateApp(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO open_apps`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "crm-integrator", 120, 5000, true, int64(7)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	res, err := s.CreateApp(context.Background(), "crm-integrator", 120, 5000, true, 7)
	if err != nil {
		t.Fatalf("CreateApp: %v", err)
	}
	if res.ID != 1 || res.AppID == "" || !strings.HasPrefix(res.AppID, "op_") || !strings.HasPrefix(res.Secret, "ops_") {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.RateLimitRPM != 120 || res.DailyQuota != 5000 || !res.Sandbox {
		t.Fatalf("quota fields not round-tripped: %+v", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateAppDefaults(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`INSERT INTO open_apps`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "x", 60, 10000, false, int64(1)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	res, err := s.CreateApp(context.Background(), "x", 0, 0, false, 1)
	if err != nil {
		t.Fatalf("CreateApp defaults: %v", err)
	}
	if res.RateLimitRPM != 60 || res.DailyQuota != 10000 {
		t.Fatalf("defaults not applied: %+v", res)
	}
}

func TestPGStore_TouchUsageUpsert(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`INSERT INTO open_usage_day`).
		WithArgs(int64(3), pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"call_count"}).AddRow(int64(41)))
	mock.ExpectExec(`UPDATE open_apps SET last_used_at`).WithArgs(int64(3)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	n, err := s.TouchUsage(context.Background(), 3, time.Now())
	if err != nil || n != 41 {
		t.Fatalf("TouchUsage: n=%d err=%v", n, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_LookupActiveNotFound(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT id, secret, rate_limit_rpm, daily_quota FROM open_apps`).
		WithArgs("op_missing").
		WillReturnError(pgx.ErrNoRows)

	s := NewPGStore(mock)
	// 用 ErrNoRows 语义:pgxmock 直接返回错误包装即可覆盖 ErrNotFound 分支需 pgx.ErrNoRows,
	// 此处断言错误透传。
	if _, err := s.LookupActive(context.Background(), "op_missing"); err == nil {
		t.Fatal("expected error for missing app")
	}
}

// TestPGStore_ListDueClaimsBatch 回归(持久化整改):ListDue 必须单语句原子领取
// (UPDATE ... FOR UPDATE SKIP LOCKED + next_attempt_at 推进租约窗口),
// 并发投递循环/多实例重复调用批次互不相交,同一行不会被同时 POST 两次。
func TestPGStore_ListDueClaimsBatch(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	now := time.Now()
	rows := mock.NewRows([]string{
		"id", "subscription_id", "event_id", "event_type", "payload",
		"status", "attempts", "next_attempt_at", "http_status", "last_error", "delivered_at", "created_at",
		"endpoint_url", "secret",
	}).AddRow(int64(9), int64(2), "ORD-1:stage:12", "order.stage.done", []byte(`{"ok":true}`),
		int16(0), 0, now, nil, "", nil, now, "https://ex.test/hook", "ops_s")
	mock.ExpectQuery(`WITH claimed AS `).
		WithArgs(pgxmock.AnyArg(), DeliveryBatchMax, float64(claimLeaseSeconds)).
		WillReturnRows(rows)

	s := NewPGStore(mock)
	due, err := s.ListDue(context.Background(), now, DeliveryBatchMax)
	if err != nil || len(due) != 1 {
		t.Fatalf("ListDue: n=%d err=%v", len(due), err)
	}
	if due[0].ID != 9 || due[0].EndpointURL != "https://ex.test/hook" || due[0].Secret != "ops_s" {
		t.Fatalf("unexpected row: %+v", due[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
