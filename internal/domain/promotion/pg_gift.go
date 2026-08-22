package promotion

import (
	"context"
	"fmt"
)

// 赠送时长(阶梯规则:6送1/12送3/24送6),与券共用本域。

// ListGiftRules 规则列表(含禁用)。
func (s *PGStore) ListGiftRules(ctx context.Context) ([]GiftRule, error) {
	rows, err := s.db.Query(ctx, `
		SELECT rule_id, legal_entity_id, name, scope_type, COALESCE(scope_ref,0),
			buy_months, gift_months, status
		FROM gift_rules ORDER BY rule_id`)
	if err != nil {
		return nil, fmt.Errorf("promotion: list gift rules: %w", err)
	}
	defer rows.Close()
	out := make([]GiftRule, 0)
	for rows.Next() {
		var r GiftRule
		if err := rows.Scan(&r.RuleID, &r.LegalEntityID, &r.Name, &r.ScopeType,
			&r.ScopeRef, &r.BuyMonths, &r.GiftMonths, &r.Status); err != nil {
			return nil, fmt.Errorf("promotion: scan gift rule: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateGiftRule 新建赠送规则(直接 ENABLED)。
func (s *PGStore) CreateGiftRule(ctx context.Context, r GiftRule) (int64, error) {
	if r.ScopeType == "" {
		r.ScopeType = "ALL"
	}
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO gift_rules(legal_entity_id, name, scope_type, scope_ref,
			buy_months, gift_months, status)
		VALUES($1,$2,$3,NULLIF($4,0),$5,$6,'ENABLED') RETURNING rule_id`,
		r.LegalEntityID, r.Name, r.ScopeType, r.ScopeRef, r.BuyMonths, r.GiftMonths).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("promotion: create gift rule: %w", err)
	}
	return id, nil
}

// DisableGiftRule 停用赠送规则。
func (s *PGStore) DisableGiftRule(ctx context.Context, ruleID int64) error {
	return s.execAffected(ctx, "disable gift rule",
		`UPDATE gift_rules SET status='DISABLED' WHERE rule_id=$1 AND status='ENABLED'`, ruleID)
}

// MatchGiftRule 命中实购月数的最优档(buy_months 最大且 <= 实购);无命中返回 nil。
func (s *PGStore) MatchGiftRule(ctx context.Context, productID int64, buyMonths int) (*GiftRule, error) {
	var r GiftRule
	err := s.db.QueryRow(ctx, `
		SELECT rule_id, legal_entity_id, name, scope_type, COALESCE(scope_ref,0),
			buy_months, gift_months, status
		FROM gift_rules
		WHERE status='ENABLED' AND buy_months <= $2
		  AND (scope_type='ALL' OR (scope_type='PRODUCT' AND scope_ref=$1))
		ORDER BY buy_months DESC LIMIT 1`, productID, buyMonths).
		Scan(&r.RuleID, &r.LegalEntityID, &r.Name, &r.ScopeType,
			&r.ScopeRef, &r.BuyMonths, &r.GiftMonths, &r.Status)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("promotion: match gift rule: %w", err)
	}
	return &r, nil
}

// RecordGift 赠送落痕(缴费成功后调用)。
func (s *PGStore) RecordGift(ctx context.Context, r GiftRecord) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO gift_records(rule_id, customer_id, product_id, buy_months,
			gift_months, payment_id)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,0))`,
		r.RuleID, r.CustomerID, r.ProductID, r.BuyMonths, r.GiftMonths, r.PaymentID)
	if err != nil {
		return fmt.Errorf("promotion: record gift: %w", err)
	}
	return nil
}
