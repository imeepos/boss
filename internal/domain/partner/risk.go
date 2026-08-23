package partner

import "context"

const (
	DefaultDailyOrderLimit       = 100
	DefaultCustomerCooldownHours = 24
)

type OrderRiskDecision struct {
	Allowed             bool   `json:"allowed"`
	Reason              string `json:"reason,omitempty"`
	DailyOrderCount     int64  `json:"dailyOrderCount"`
	CustomerRecentCount int64  `json:"customerRecentCount"`
}

type OrderRiskService interface {
	CheckOrderRisk(ctx context.Context, accountID, customerID int64) (OrderRiskDecision, error)
}
