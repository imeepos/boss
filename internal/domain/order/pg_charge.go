package order

// 环节4 合同收费:预付费当场收款(adopted note 2026-08-22-prepaid-postpaid-billing-mode);
// 预缴 N 月按 N x 月费收,赠送阶梯(gift_duration_rules)命中回填 orders.gift_months。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ChargeContract 环节4 合同收费(未收费不派单的硬约束由顺序守卫保证:dispatch 需 stage=7)。
// 预付费订单先当场收款(未收成不推进,REQ-CL-001);后付费维持人工确认推进。
func (s *PGStore) ChargeContract(ctx context.Context, orderID int64) error {
	if err := s.collectPrepaid(ctx, orderID); err != nil {
		return err
	}
	return s.advance(ctx, orderID, "chargeContract")
}

// collectPrepaid 预付费当场收款:预缴月数=orders.buy_months(0 视为按月缴 1 个月),
// 金额=月数 x 月费(区域覆盖优先,与出账同口径);赠送月数回填 orders.gift_months。
// collector 未注入或收款失败时返回错误,环节 4 不推进。
func (s *PGStore) collectPrepaid(ctx context.Context, orderID int64) error {
	var mode string
	var customerID, offerID, buyMonths int64
	err := s.db.QueryRow(ctx,
		`SELECT billing_mode, customer_id, offer_id, buy_months FROM orders WHERE id = $1`, orderID).
		Scan(&mode, &customerID, &offerID, &buyMonths)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("order: charge select: %w", err)
	}
	if mode != BillingModePrepaid {
		return nil
	}
	if s.prepaid == nil {
		return errors.New("order: prepaid collector not wired")
	}
	months := buyMonths
	if months < 1 {
		months = 1
	}
	fee, err := s.prepaidMonthlyFee(ctx, orderID)
	if err != nil {
		return err
	}
	gift, err := s.prepaid.Collect(ctx, customerID, fee*float64(months), offerID, int(months))
	if err != nil {
		return fmt.Errorf("order: prepaid collect: %w", err)
	}
	return s.writebackGiftMonths(ctx, orderID, gift)
}

// PrepaidAmount 预付费订单应收(环节4/师傅现场收款同口径):金额=月数 x 月费,
// 月数=orders.buy_months(0 视为按月缴 1 个月);后付费返回 prepaid=false。
func (s *PGStore) PrepaidAmount(ctx context.Context, orderID int64) (float64, int, bool, error) {
	var mode string
	var buyMonths int64
	err := s.db.QueryRow(ctx,
		`SELECT billing_mode, buy_months FROM orders WHERE id = $1`, orderID).
		Scan(&mode, &buyMonths)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, false, ErrOrderNotFound
	}
	if err != nil {
		return 0, 0, false, fmt.Errorf("order: charge select: %w", err)
	}
	if mode != BillingModePrepaid {
		return 0, 0, false, nil
	}
	months := buyMonths
	if months < 1 {
		months = 1
	}
	fee, err := s.prepaidMonthlyFee(ctx, orderID)
	if err != nil {
		return 0, 0, false, err
	}
	return fee * float64(months), int(months), true, nil
}

// writebackGiftMonths 赠送月数回填快照;0 不写。
func (s *PGStore) writebackGiftMonths(ctx context.Context, orderID int64, gift int) error {
	if gift <= 0 {
		return nil
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE orders SET gift_months = $2 WHERE id = $1`, orderID, gift); err != nil {
		return fmt.Errorf("order: writeback gift_months: %w", err)
	}
	return nil
}

// prepaidMonthlyFee 预付费月费:区域月费覆盖(region_offers)优先,否则产品基础月费。
func (s *PGStore) prepaidMonthlyFee(ctx context.Context, orderID int64) (float64, error) {
	var amount float64
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(ro.monthly_fee, po.monthly_fee)
		FROM orders o
		JOIN product_offers po ON po.id = o.offer_id
		LEFT JOIN region_offers ro ON ro.offer_id = o.offer_id AND o.region_path <> '' AND o.region_path = o.region_path
		WHERE o.id = $1`, orderID).Scan(&amount)
	if err != nil {
		return 0, fmt.Errorf("order: prepaid amount: %w", err)
	}
	return amount, nil
}
