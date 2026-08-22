package app

// prepaidCollector 包装 billing/portal/promotion 域为 order.PrepaidCollector 接口
// (环节4 当场收款 + 赠送时长阶梯命中)。

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/promotion"
)

type prepaidCollector struct {
	bill   *billing.PGStore
	portal portal.Service
	promo  promotion.Service
}

// Collect 预付费收款:PAY 单号 + 缴费流水(bill_id NULL + customer_id 归属),
// 再按赠送阶梯(gift_duration_rules)命中落痕,返回赠送月数(0=未命中)。
// method=cash:环节4 为人工确认收款口径,移动支付渠道接入后经 Amended 扩展。
func (c prepaidCollector) Collect(ctx context.Context, customerID int64, amount float64,
	offerID int64, months int) (int, error) {
	payNo, err := c.portal.NextNo(ctx, "PAY")
	if err != nil {
		return 0, err
	}
	payID, err := c.bill.CreatePayment(ctx, billing.Payment{
		PayNo: payNo, CustomerID: customerID, Amount: amount,
		Method: "cash", Status: "SUCCESS",
	})
	if err != nil {
		return 0, err
	}
	return c.collectGift(ctx, customerID, offerID, months, payID)
}

// collectGift 赠送命中与落痕;未配置 promotion 或落痕失败不阻塞收款主流程(返回 0)。
func (c prepaidCollector) collectGift(ctx context.Context, customerID, offerID int64, months int, payID int64) (int, error) {
	if c.promo == nil {
		return 0, nil
	}
	rule, err := c.promo.MatchGiftRule(ctx, offerID, months)
	if err != nil || rule == nil {
		return 0, nil
	}
	rec := promotion.GiftRecord{
		RuleID: rule.RuleID, CustomerID: customerID, ProductID: offerID,
		BuyMonths: months, GiftMonths: rule.GiftMonths, PaymentID: payID,
	}
	if err := c.promo.RecordGift(ctx, rec); err != nil {
		return 0, nil
	}
	return rule.GiftMonths, nil
}
