package userapi

// fakePromotion 桩 promotion.Service:券仓可配置,其余方法返回零值。

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/promotion"
)

type fakePromo struct {
	items  []map[string]any
	gift   string
	redeem string
	// 续费赠送断言:match 非空时 MatchGiftRule 命中;recorded 落痕记录。
	match    *promotion.GiftRule
	recorded []promotion.GiftRecord
}

func (f *fakePromo) ListTemplates(context.Context) ([]promotion.Template, error) { return nil, nil }
func (f *fakePromo) CreateTemplate(context.Context, promotion.Template) (int64, error) {
	return 0, nil
}
func (f *fakePromo) DisableTemplate(context.Context, int64) error { return nil }
func (f *fakePromo) Issue(context.Context, int64, []int64) (int, error) {
	return 0, nil
}
func (f *fakePromo) IssueToCustomer(context.Context, int64, int64, string) (string, error) {
	return "CPN-x", nil
}
func (f *fakePromo) PointsPrice(context.Context, int64) (int64, error) { return 0, nil }
func (f *fakePromo) CreateCodes(context.Context, int64, int) ([]promotion.CouponCode, error) {
	return nil, nil
}
func (f *fakePromo) ListCodes(context.Context, int64) ([]promotion.CouponCode, error) {
	return nil, nil
}
func (f *fakePromo) RedeemCode(_ context.Context, code string, _ int64) (string, error) {
	return f.redeem, nil
}
func (f *fakePromo) CreateGift(_ context.Context, _ string, _ int64) (string, error) {
	return f.gift, nil
}
func (f *fakePromo) ListCustomerCoupons(_ context.Context, _ int64, _ string, _ int64) ([]map[string]any, error) {
	return f.items, nil
}
func (f *fakePromo) RollbackRedemption(context.Context, int64) error { return nil }
func (f *fakePromo) ListGiftRules(context.Context) ([]promotion.GiftRule, error) {
	return nil, nil
}
func (f *fakePromo) CreateGiftRule(context.Context, promotion.GiftRule) (int64, error) {
	return 0, nil
}
func (f *fakePromo) DisableGiftRule(context.Context, int64) error { return nil }
func (f *fakePromo) MatchGiftRule(context.Context, int64, int) (*promotion.GiftRule, error) {
	return f.match, nil
}
func (f *fakePromo) RecordGift(_ context.Context, r promotion.GiftRecord) error {
	f.recorded = append(f.recorded, r)
	return nil
}
