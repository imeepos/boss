package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/ymm-001/boss/internal/pkg/database"
)

// TestDeductForPayment_Mock 核销 SQL 契约:行锁占用券 → 写核销记录。
func TestDeductForPayment_Mock(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(`UPDATE coupons SET status='USED'`).
		WithArgs("CPN-1", int64(7), int64(11)).
		WillReturnRows(mock.NewRows([]string{"type", "face_value", "threshold", "max_discount"}).
			AddRow("FULL_CUT", int64(2000), int64(10000), int64(0)))
	mock.ExpectExec(`INSERT INTO coupon_redemptions`).
		WithArgs("CPN-1", int64(11), int64(7), int64(2000)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := NewPGStore(nil)
	got, err := s.DeductForPayment(context.Background(), tx, "CPN-1", 7, 11, 15000)
	if err != nil || got != 2000 {
		t.Fatalf("got=%d err=%v", got, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPromotion_Integration 全链路集成(需真实 PostgreSQL,BOSS_PG_TEST_DSN 未设跳过):
// 建模板 → 发放 → 券仓 → 缴费核销 → 一券多用冲突 → 转赠/兑换。
func TestPromotion_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("database.Migrate: %v", err)
	}
	s := NewPGStore(pool)

	var le, custID, custID2 int64
	cleanup := func() {
		pool.Exec(ctx, `DELETE FROM coupon_redemptions WHERE coupon_id LIKE 'CPN-%'`)
		pool.Exec(ctx, `DELETE FROM coupons WHERE customer_id IN ($1,$2)`, custID, custID2)
		pool.Exec(ctx, `DELETE FROM coupon_codes WHERE template_id IN (SELECT template_id FROM coupon_templates WHERE legal_entity_id=$1)`, le)
		pool.Exec(ctx, `DELETE FROM coupon_templates WHERE legal_entity_id=$1`, le)
		pool.Exec(ctx, `DELETE FROM gift_records WHERE customer_id IN ($1,$2)`, custID, custID2)
		pool.Exec(ctx, `DELETE FROM gift_rules WHERE legal_entity_id=$1`, le)
		pool.Exec(ctx, `DELETE FROM customers WHERE legal_entity_id=$1`, le)
		pool.Exec(ctx, `DELETE FROM legal_entities WHERE id=$1`, le)
	}
	cleanup()
	defer cleanup()
	if err := pool.QueryRow(ctx,
		`INSERT INTO legal_entities(code,name) VALUES('PROMO-T','促销测试主体') RETURNING id`).Scan(&le); err != nil {
		t.Fatalf("seed entity: %v", err)
	}
	for i, c := range []*int64{&custID, &custID2} {
		if err := pool.QueryRow(ctx, `
			INSERT INTO customers(legal_entity_id, name, phone, id_type, address_id, region_id, region_name, customer_code)
			VALUES($1,'促销测试客户',$2,'身份证',NULL,1,'马尼拉',$3) RETURNING id`, le,
			fmt.Sprintf("139000000%d", i), fmt.Sprintf("PROMO-T%d", i)).Scan(c); err != nil {
			t.Fatalf("seed customer: %v", err)
		}
	}

	tplID, err := s.CreateTemplate(ctx, Template{
		LegalEntityID: le, Name: "满100减20", Type: TypeFullCut,
		FaceValue: 2000, Threshold: 10000, ValidDays: 30,
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if n, err := s.Issue(ctx, tplID, []int64{custID}); err != nil || n != 1 {
		t.Fatalf("Issue: n=%d err=%v", n, err)
	}
	// 限领:同一客户再发被跳过
	if n, err := s.Issue(ctx, tplID, []int64{custID}); err != nil || n != 0 {
		t.Fatalf("Issue limit: n=%d err=%v", n, err)
	}
	items, err := s.ListCustomerCoupons(ctx, custID, "available", 15000)
	if err != nil || len(items) != 1 || items[0]["estDeduct"] != int64(2000) {
		t.Fatalf("ListCustomerCoupons: %v %v", items, err)
	}
	couponID := items[0]["couponId"].(string)

	// 缴费核销(mock 事务由调用方管理;此处用真事务)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d, err := s.DeductForPayment(ctx, tx, couponID, custID, 999, 15000); err != nil || d != 2000 {
		tx.Rollback(ctx)
		t.Fatalf("Deduct: d=%d err=%v", d, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	// 一券多用 → 冲突
	tx2, _ := pool.Begin(ctx)
	if _, err := s.DeductForPayment(ctx, tx2, couponID, custID, 1000, 15000); err == nil {
		t.Fatal("double redeem should conflict")
	}
	tx2.Rollback(ctx)

	// 赠送规则 + 命中
	ruleID, err := s.CreateGiftRule(ctx, GiftRule{
		LegalEntityID: le, Name: "12送3", BuyMonths: 12, GiftMonths: 3,
	})
	if err != nil {
		t.Fatalf("CreateGiftRule: %v", err)
	}
	rule, err := s.MatchGiftRule(ctx, 0, 13)
	if err != nil || rule == nil || rule.RuleID != ruleID {
		t.Fatalf("MatchGiftRule: %v %+v", err, rule)
	}
	if err := s.RecordGift(ctx, GiftRecord{
		RuleID: ruleID, CustomerID: custID, BuyMonths: 13, GiftMonths: rule.GiftMonths,
	}); err != nil {
		t.Fatalf("RecordGift: %v", err)
	}

	// 兑换码 + 转赠
	codes, err := s.CreateCodes(ctx, tplID, 1)
	if err != nil || len(codes) != 1 {
		t.Fatalf("CreateCodes: err=%v codes=%v", err, codes)
	}
	if _, err := s.RedeemCode(ctx, codes[0].Code, custID2); err != nil {
		t.Fatalf("RedeemCode: %v", err)
	}
	// 兑换码三态业务码(2026-09-04 客户端链路修复):已用/不存在/模板停用。
	if _, err := s.RedeemCode(ctx, codes[0].Code, custID2); !errors.Is(err, ErrCodeUsedOrDisabled) {
		t.Fatalf("re-redeem want ErrCodeUsedOrDisabled, got %v", err)
	}
	if _, err := s.RedeemCode(ctx, "RDM-not-exist", custID2); !errors.Is(err, ErrCodeNotFound) {
		t.Fatalf("unknown code want ErrCodeNotFound, got %v", err)
	}
	tpl2, err := s.CreateTemplate(ctx, Template{
		LegalEntityID: le, Name: "停用态模板", Type: TypeCash, FaceValue: 500,
	})
	if err != nil {
		t.Fatalf("CreateTemplate tpl2: %v", err)
	}
	codes2, err := s.CreateCodes(ctx, tpl2, 1)
	if err != nil || len(codes2) != 1 {
		t.Fatalf("CreateCodes tpl2: err=%v codes=%v", err, codes2)
	}
	if err := s.DisableTemplate(ctx, tpl2); err != nil {
		t.Fatalf("DisableTemplate: %v", err)
	}
	if _, err := s.RedeemCode(ctx, codes2[0].Code, custID2); !errors.Is(err, ErrTemplateDisabled) {
		t.Fatalf("disabled template want ErrTemplateDisabled, got %v", err)
	}
	gifts, err := s.ListCustomerCoupons(ctx, custID2, "all", 0)
	if err != nil || len(gifts) != 1 {
		t.Fatalf("custID2 coupons: %v %v", gifts, err)
	}
	gcode, err := s.CreateGift(ctx, gifts[0]["couponId"].(string), custID2)
	if err != nil {
		t.Fatalf("CreateGift: %v", err)
	}
	if _, err := s.RedeemCode(ctx, gcode, custID); err != nil {
		t.Fatalf("redeem gift: %v", err)
	}
}
