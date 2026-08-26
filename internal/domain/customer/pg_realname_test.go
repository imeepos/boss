package customer

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_ListVerifications 契约:按客户过滤实名核验记录;customerID=0 返回全部。
func TestPGStore_ListVerifications(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, subject_id, method, COALESCE\(real_name, ''\), COALESCE\(id_card_no, ''\), verified_at, result, reject_reason, COALESCE\(operator_account_id, 0\), COALESCE\(operator_name, ''\)`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{
			"id", "customer_id", "method", "real_name", "id_card_no",
			"verified_at", "result", "reject_reason", "operator_account_id", "operator_name",
		}).AddRow(int64(1), int64(1), "人脸", "张三", "110101199001011234",
			fixedTime, "PASS", "", int64(3), "李四"))

	s := NewPGStore(mock)
	got, err := s.ListVerifications(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListVerifications: %v", err)
	}
	if len(got) != 1 || got[0].Method != "人脸" || got[0].Result != "PASS" ||
		got[0].OperatorName != "李四" || got[0].RealName != "张三" || got[0].IDCardNo != "110101199001011234" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_AppendVerification 契约:追加核验记录并返回自增 id;operator=0 写 NULL。
func TestPGStore_AppendVerification(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO verifications\(subject_type, subject_id, method, verified_at, result, operator_account_id, operator_name\)`).
		WithArgs(int64(1), "人脸", fixedTime, "PASS", nil, "张三").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))

	s := NewPGStore(mock)
	id, err := s.AppendVerification(context.Background(), RealNameVerification{
		CustomerID: 1, Method: "人脸", VerifiedAt: fixedTime, Result: "PASS", OperatorName: "张三",
	})
	if err != nil {
		t.Fatalf("AppendVerification: %v", err)
	}
	if id != 9 {
		t.Fatalf("id=%d, want 9", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
