package app

// prepaidCollector 包装 billing/portal/promotion 域为 order.PrepaidCollector 接口
// (环节4 当场收款 + 赠送时长阶梯命中)。

import (
	"context"
	"log"

	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/loy"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/promotion"
)

type prepaidCollector struct {
	bill   *billing.PGStore
	portal portal.Service
	promo  promotion.Service
	points loy.Service // 可空:缴费自动积分,失败不阻塞收款
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
	c.earnPoints(ctx, customerID, payID, amount)
	return c.collectGift(ctx, customerID, offerID, months, payID)
}

// earnPoints 缴费自动积分(2028 Q2):按 loy_earn_rules 换算,以 payment_id 幂等;
// 失败仅记日志不阻塞收款(过期清算循环/下次重放可兜底)。
func (c prepaidCollector) earnPoints(ctx context.Context, customerID, payID int64, amount float64) {
	if c.points == nil {
		return
	}
	cents := int64(amount * 100)
	if _, err := c.points.EarnForPayment(ctx, payID, customerID, cents); err != nil {
		log.Printf("[loy-earn] payment %d earn failed: %v", payID, err)
	}
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
