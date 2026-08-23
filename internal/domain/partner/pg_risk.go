package partner

import (
	"context"
	"fmt"
)

func (s *PGStore) CheckOrderRisk(ctx context.Context, accountID, customerID int64) (OrderRiskDecision, error) {
	entityID, err := s.entityOfAccount(ctx, accountID)
	if err != nil {
		return OrderRiskDecision{}, err
	}
	if entityID == 0 {
		return OrderRiskDecision{}, ErrNotPartner
	}
	var daily, recent int64
	err = s.db.QueryRow(ctx, `
SELECT
 (SELECT count(*) FROM orders WHERE legal_entity_id=$1 AND created_at >= date_trunc('day', now())),
 (SELECT count(*) FROM orders WHERE legal_entity_id=$1 AND customer_id=$2 AND created_at >= now() - interval '24 hours')`, entityID, customerID).Scan(&daily, &recent)
	if err != nil {
		return OrderRiskDecision{}, fmt.Errorf("partner: check order risk: %w", err)
	}
	decision := OrderRiskDecision{Allowed: true, DailyOrderCount: daily, CustomerRecentCount: recent}
	if daily >= DefaultDailyOrderLimit {
		decision.Allowed = false
		decision.Reason = "DAILY_ORDER_LIMIT"
	} else if recent > 0 {
		decision.Allowed = false
		decision.Reason = "CUSTOMER_COOLDOWN"
	}
	return decision, nil
}
