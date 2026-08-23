package order

import "context"

const partnerDefaultRate = 0.10

type PartnerCommissionAccrual interface {
	AccrueCommission(ctx context.Context, orderID, legalEntityID int64, orderAmount, rate float64) (int64, error)
}
