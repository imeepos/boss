package userdata

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrPlanNotFound 套餐未命中或不属于该客户。
var ErrPlanNotFound = errors.New("userdata: plan not found")

// GetPlanForRenewal 续费前置:套餐归属校验 + 产品基础月费。
func (s *PGStore) GetPlanForRenewal(ctx context.Context, planID, customerID int64) (int64, float64, error) {
	var productID int64
	var fee float64
	err := s.db.QueryRow(ctx, `
		SELECT p.product_id, po.monthly_fee
		FROM user_plans p JOIN product_offers po ON po.id = p.product_id
		WHERE p.id = $1 AND p.customer_id = $2`,
		planID, customerID).Scan(&productID, &fee)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, ErrPlanNotFound
	}
	if err != nil {
		return 0, 0, fmt.Errorf("userdata: plan for renewal: %w", err)
	}
	return productID, fee, nil
}

// RenewPlan 延长合约到期月:contract_end 为 'YYYY-MM',空/过期自当前月起算。
func (s *PGStore) RenewPlan(ctx context.Context, planID, customerID int64, months int) (string, error) {
	var end string
	err := s.db.QueryRow(ctx, `
		UPDATE user_plans SET contract_end = to_char(
			GREATEST(date_trunc('month', now())::date,
			         COALESCE((contract_end || '-01')::date, date_trunc('month', now())::date))
			+ make_interval(months => $3), 'YYYY-MM')
		WHERE id = $1 AND customer_id = $2
		RETURNING contract_end`,
		planID, customerID, months).Scan(&end)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrPlanNotFound
	}
	if err != nil {
		return "", fmt.Errorf("userdata: renew plan: %w", err)
	}
	return end, nil
}
