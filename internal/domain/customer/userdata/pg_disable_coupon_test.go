package userdata

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// DisableCoupon 值集回归:必须写大写 'DISABLED'。
// 2026-09 审计:000102 已把值集迁为 ISSUED/USED/DISABLED,本方法仍写小写
// 'disabled',与 promotion 域大写读写并存,补偿统计把停用行误计入已用;
// 000177 归一存量并加 CHECK 后,小写写入会直接违反约束。
func TestDisableCoupon_UppercaseStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec("UPDATE coupons SET status = .DISABLED.").
		WithArgs("CPN-1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	if err := s.DisableCoupon(context.Background(), "CPN-1"); err != nil {
		t.Fatalf("DisableCoupon: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
