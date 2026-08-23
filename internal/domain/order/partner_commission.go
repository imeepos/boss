package order

import (
	"context"
	"strconv"
)

const partnerDefaultRate = 0.10

type PartnerCommissionRate interface {
	GetParam(ctx context.Context, key string) (string, error)
}

type PartnerCommissionAccrual interface {
	AccrueCommission(ctx context.Context, orderID, legalEntityID int64, orderAmount, rate float64) (int64, error)
}

func partnerRate(ctx context.Context, params PartnerCommissionRate) float64 {
	if params == nil {
		return partnerDefaultRate
	}
	value, err := params.GetParam(ctx, "commission.partnerDefaultRate")
	if err != nil {
		return partnerDefaultRate
	}
	rate, err := strconv.ParseFloat(value, 64)
	if err != nil || rate < 0 || rate > 1 {
		return partnerDefaultRate
	}
	return rate
}
