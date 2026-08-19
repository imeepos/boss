package portal

// 自动缴费偏好(portal_billing_prefs):按客户落库,缺省未开通。

import (
	"context"
	"errors"
)

// AutoPay 读自动缴费开通状态;无记录视为未开通。
func (s *pgStore) AutoPay(ctx context.Context, customerID int64) (bool, error) {
	var on bool
	err := s.pool.QueryRow(ctx,
		`SELECT auto_pay_enabled FROM portal_billing_prefs WHERE customer_id=$1`, customerID).Scan(&on)
	if errors.Is(err, errNoRows) {
		return false, nil
	}
	return on, err
}

// SetAutoPay 开通/关闭自动缴费(upsert)。
func (s *pgStore) SetAutoPay(ctx context.Context, customerID int64, enabled bool) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO portal_billing_prefs(customer_id, auto_pay_enabled)
		VALUES ($1, $2)
		ON CONFLICT (customer_id) DO UPDATE SET auto_pay_enabled = EXCLUDED.auto_pay_enabled, updated_at = now()`,
		customerID, enabled)
	return err
}
