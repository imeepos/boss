package partner

import "context"

const (
	CommissionAccrued = "ACCRUED"
	CommissionSettled = "SETTLED"
	CommissionVoid    = "VOID"
)

type CommissionLedgerRow struct {
	ID               int64   `json:"id"`
	OrderID          int64   `json:"orderId"`
	LegalEntityID    int64   `json:"legalEntityId"`
	OrderAmount      float64 `json:"orderAmount"`
	CommissionRate   float64 `json:"commissionRate"`
	CommissionAmount float64 `json:"commissionAmount"`
	Status           string  `json:"status"`
	SettledAt        string  `json:"settledAt,omitempty"`
	SettledBy        int64   `json:"settledBy,omitempty"`
	CreatedAt        string  `json:"createdAt"`
}

type CommissionLedgerService interface {
	ListCommissionLedger(ctx context.Context, accountID int64, status string) ([]CommissionLedgerRow, error)
	AccrueCommission(ctx context.Context, orderID, legalEntityID int64, orderAmount, rate float64) (int64, error)
	SettleCommissionLedger(ctx context.Context, ledgerID, accountID int64) error
}
