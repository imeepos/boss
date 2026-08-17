package customer

import (
	"context"
	"time"
)

// CustomerHistory 客户归属台账(转品牌/搬家的时间段)。
type CustomerHistory struct {
	ID                int64
	CustomerID        int64
	LegalEntityID     int64
	LegalEntityName   string
	AddressID         int64
	AddressName       string
	RegionID          int64
	RegionName        string
	Reason            string
	OperatorAccountID int64 // 0=空
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time // nil=至今
}

// ProductPriceHistory 产品调价台账(基础月费每次调整)。
type ProductPriceHistory struct {
	ID                int64     `json:"id"`
	OfferID           int64     `json:"offerId"`
	OldMonthlyFee     float64   `json:"oldMonthlyFee"`
	NewMonthlyFee     float64   `json:"newMonthlyFee"`
	EffectiveAt       time.Time `json:"effectiveAt"`
	Reason            string    `json:"reason"`
	OperatorAccountID int64     `json:"operatorAccountId"` // 0=空
}

// RegionPriceHistory 区域调价台账(区域月费每次调整)。
type RegionPriceHistory struct {
	ID                int64
	RegionOfferID     int64
	OldMonthlyFee     float64
	NewMonthlyFee     float64
	EffectiveAt       time.Time
	Reason            string
	OperatorAccountID int64
}

// CustomerLedgerService 客户与资费台账域服务口(阶段2)。
type CustomerLedgerService interface {
	ListCustomerHistories(ctx context.Context, customerID int64) ([]CustomerHistory, error)
	AppendCustomerHistory(ctx context.Context, h CustomerHistory) (int64, error)
	ListProductPriceHistories(ctx context.Context, offerID int64) ([]ProductPriceHistory, error)
	AppendProductPriceHistory(ctx context.Context, h ProductPriceHistory) (int64, error)
	ListRegionPriceHistories(ctx context.Context, regionOfferID int64) ([]RegionPriceHistory, error)
	AppendRegionPriceHistory(ctx context.Context, h RegionPriceHistory) (int64, error)
}
