package promotion

// 兑换码 SQL 契约测试(pgxmock):钉死消歧后的限定列引用与三态业务错误。
// 背景:RedeemCode 曾对 JOIN 裸引用 templateCols,coupon_codes/coupon_templates
// 两表均有 template_id/status,真实 PG 必撞 42702(SQLSTATE),客户端任何兑换码
// 都落 50000(102 环境 2026-09-04 取证)。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// templateRow 模板 17 列行构造器(列序与 templateCols 严格对齐)。
func templateRow(tplID int64) *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"template_id", "legal_entity_id", "name", "type", "face_value", "threshold",
		"max_discount", "scope_type", "scope_ref", "total_qty", "issued_qty",
		"per_customer_limit", "valid_days", "valid_from", "valid_to", "status", "points_price",
	}).AddRow(tplID, int64(1), "测试券", "CASH", int64(2000), int64(0),
		int64(0), "ALL", int64(0), int64(0), int64(0),
		1, 0, "", "", "ENABLED", int64(0))
}

// expectGiftMiss 转赠试查落空(空行集 → pgx.ErrNoRows)。
func expectGiftMiss(mock pgxmock.PgxPoolIface, code string) {
	mock.ExpectQuery("UPDATE coupons SET customer_id").
		WithArgs(code, int64(7)).
		WillReturnRows(mock.NewRows([]string{"coupon_id"}))
}

// TestRedeemCode_SQL_Disambiguation_Mock 兑换成功全链路:cc. 限定列引用 + 模板单表查询。
func TestRedeemCode_SQL_Disambiguation_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	s := NewPGStore(mock)

	mock.ExpectBegin()
	expectGiftMiss(mock, "RDM-1")
	// 钉死限定列引用:消歧修复点,若回退为 JOIN 裸引用,此 pattern 不再命中。
	mock.ExpectQuery("SELECT cc.status, cc.template_id FROM coupon_codes cc WHERE cc.code").
		WithArgs("RDM-1").
		WillReturnRows(mock.NewRows([]string{"status", "template_id"}).AddRow("UNUSED", int64(5)))
	mock.ExpectQuery("FROM coupon_templates").
		WithArgs(int64(5)).
		WillReturnRows(templateRow(5))
	mock.ExpectQuery("SELECT count").WithArgs(int64(5), int64(7)).
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec("INSERT INTO coupons").
		WithArgs(pgxmock.AnyArg(), int64(7), "测试券", int64(2000), int64(5), "CASH",
			int64(2000), int64(0), int64(0), "ALL", int64(0), "REDEEM", "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("UPDATE coupon_codes SET status").
		WithArgs("RDM-1", int64(7)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec("UPDATE coupon_templates SET issued_qty").
		WithArgs(int64(5)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	couponID, err := s.RedeemCode(context.Background(), "RDM-1", 7)
	if err != nil || couponID == "" {
		t.Fatalf("RedeemCode: couponID=%q err=%v", couponID, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestRedeemCode_NotFound_Mock 码不存在 → ErrCodeNotFound(客户端 40400)。
func TestRedeemCode_NotFound_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	s := NewPGStore(mock)

	mock.ExpectBegin()
	expectGiftMiss(mock, "RDM-none")
	mock.ExpectQuery("SELECT cc.status, cc.template_id FROM coupon_codes cc WHERE cc.code").
		WithArgs("RDM-none").
		WillReturnRows(mock.NewRows([]string{"status", "template_id"}))
	mock.ExpectRollback()

	if _, err := s.RedeemCode(context.Background(), "RDM-none", 7); !errors.Is(err, ErrCodeNotFound) {
		t.Fatalf("want ErrCodeNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestRedeemCode_UsedCode_Mock 码已兑换 → ErrCodeUsedOrDisabled(客户端 40900)。
func TestRedeemCode_UsedCode_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	s := NewPGStore(mock)

	mock.ExpectBegin()
	expectGiftMiss(mock, "RDM-used")
	mock.ExpectQuery("SELECT cc.status, cc.template_id FROM coupon_codes cc WHERE cc.code").
		WithArgs("RDM-used").
		WillReturnRows(mock.NewRows([]string{"status", "template_id"}).AddRow("REDEEMED", int64(5)))
	mock.ExpectRollback()

	if _, err := s.RedeemCode(context.Background(), "RDM-used", 7); !errors.Is(err, ErrCodeUsedOrDisabled) {
		t.Fatalf("want ErrCodeUsedOrDisabled, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestRedeemCode_TemplateDisabled_Mock 码有效但模板停用 → ErrTemplateDisabled(客户端 40900)。
func TestRedeemCode_TemplateDisabled_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	s := NewPGStore(mock)

	mock.ExpectBegin()
	expectGiftMiss(mock, "RDM-td")
	mock.ExpectQuery("SELECT cc.status, cc.template_id FROM coupon_codes cc WHERE cc.code").
		WithArgs("RDM-td").
		WillReturnRows(mock.NewRows([]string{"status", "template_id"}).AddRow("UNUSED", int64(9)))
	mock.ExpectQuery("FROM coupon_templates").
		WithArgs(int64(9)).
		WillReturnRows(pgxmock.NewRows([]string{
			"template_id", "legal_entity_id", "name", "type", "face_value", "threshold",
			"max_discount", "scope_type", "scope_ref", "total_qty", "issued_qty",
			"per_customer_limit", "valid_days", "valid_from", "valid_to", "status", "points_price",
		}))
	mock.ExpectRollback()

	if _, err := s.RedeemCode(context.Background(), "RDM-td", 7); !errors.Is(err, ErrTemplateDisabled) {
		t.Fatalf("want ErrTemplateDisabled, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestRedeemCode_GiftPathUnchanged_Mock 转赠码仍优先过户(回归保护)。
func TestRedeemCode_GiftPathUnchanged_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	s := NewPGStore(mock)

	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE coupons SET customer_id").
		WithArgs("GFT-1", int64(7)).
		WillReturnRows(mock.NewRows([]string{"coupon_id"}).AddRow("CPN-9"))
	mock.ExpectCommit()

	couponID, err := s.RedeemCode(context.Background(), "GFT-1", 7)
	if err != nil || couponID != "CPN-9" {
		t.Fatalf("gift redeem: couponID=%q err=%v", couponID, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
