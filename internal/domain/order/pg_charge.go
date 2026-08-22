package order

// 环节4 合同收费:预付费当场收款(adopted note 2026-08-22-prepaid-postpaid-billing-mode)。

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

// collectPrepaid 预付费当场收款:金额=区域月费覆盖优先,否则产品基础月费(与出账同口径)。
// collector 未注入或收款失败时返回错误,环节 4 不推进。
func (s *PGStore) collectPrepaid(ctx context.Context, orderID int64) error {
	var mode string
	var customerID int64
	err := s.db.QueryRow(ctx,
		`SELECT billing_mode, customer_id FROM orders WHERE id = $1`, orderID).
		Scan(&mode, &customerID)
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
	amount, err := s.prepaidAmount(ctx, orderID)
	if err != nil {
		return err
	}
	if err := s.prepaid.Collect(ctx, customerID, amount); err != nil {
		return fmt.Errorf("order: prepaid collect: %w", err)
	}
	return nil
}

// prepaidAmount 预付费收款金额:区域月费覆盖(region_offers)优先,否则产品基础月费。
func (s *PGStore) prepaidAmount(ctx context.Context, orderID int64) (float64, error) {
	var amount float64
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(ro.monthly_fee, po.monthly_fee)
		FROM orders o
		JOIN product_offers po ON po.id = o.offer_id
		LEFT JOIN region_offers ro ON ro.offer_id = o.offer_id AND o.region_path <> '' AND ro.region_path = o.region_path
		WHERE o.id = $1`, orderID).Scan(&amount)
	if err != nil {
		return 0, fmt.Errorf("order: prepaid amount: %w", err)
	}
	return amount, nil
}
