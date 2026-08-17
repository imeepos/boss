package user

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

// TestPGStore_ListAccountOrgHistories 契约:按账号过滤组织归属台账;accountID=0 返回全部。
func TestPGStore_ListAccountOrgHistories(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	to := ts.Add(24 * time.Hour)
	mock.ExpectQuery(`SELECT id, account_id, COALESCE\(legal_entity_id, 0\), COALESCE\(legal_entity_name, ''\), COALESCE\(dept_id, 0\), COALESCE\(dept_name, ''\), COALESCE\(post_id, 0\), COALESCE\(post_name, ''\), COALESCE\(reason, ''\), COALESCE\(operator_account_id, 0\), effective_from, effective_to`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{
			"id", "account_id", "legal_entity_id", "legal_entity_name", "dept_id", "dept_name",
			"post_id", "post_name", "reason", "operator_account_id", "effective_from", "effective_to",
		}).AddRow(int64(1), int64(1), int64(1), "Acme Ltd", int64(10), "运维部", int64(100), "工程师", "转岗", int64(3), ts, to))

	s := NewPGStore(mock)
	got, err := s.ListAccountOrgHistories(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListAccountOrgHistories: %v", err)
	}
	if len(got) != 1 || got[0].DeptName != "运维部" || got[0].PostName != "工程师" || got[0].EffectiveTo == nil || !got[0].EffectiveTo.Equal(to) {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_AppendAccountOrgHistory 契约:追加组织归属台账并返回自增 id;0 值写 NULL。
func TestPGStore_AppendAccountOrgHistory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	to := ts.Add(24 * time.Hour)
	mock.ExpectQuery(`INSERT INTO account_org_histories`).
		WithArgs(int64(1), int64(1), "Acme Ltd", int64(10), "运维部", nil, "工程师", "转岗", nil, ts, &to).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))

	s := NewPGStore(mock)
	id, err := s.AppendAccountOrgHistory(context.Background(), AccountOrgHistory{
		AccountID: 1, LegalEntityID: 1, LegalEntityName: "Acme Ltd", DeptID: 10, DeptName: "运维部",
		PostName: "工程师", Reason: "转岗", EffectiveFrom: ts, EffectiveTo: &to,
	})
	if err != nil {
		t.Fatalf("AppendAccountOrgHistory: %v", err)
	}
	if id != 7 {
		t.Fatalf("id=%d, want 7", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
