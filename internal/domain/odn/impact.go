package odn

import (
	"context"
)

// 影响面分析输出(P7,T10;运维侧"一缆断/一设施坏影响谁")。
type ImpactCoverage struct {
	AddressID   int64  `json:"addressId"`
	AddressName string `json:"addressName,omitempty"`
	Status      string `json:"status"`
}

type ImpactCustomer struct {
	CustomerID int64  `json:"customerId"`
	Name       string `json:"name"`
	Phone      string `json:"phone,omitempty"`
}

// ImpactReport 设施维度影响面报告。
type ImpactReport struct {
	FacilityCode string           `json:"facilityCode"`
	FacilityName string           `json:"facilityName,omitempty"`
	Coverages    []ImpactCoverage `json:"coverages"`
	Customers    []ImpactCustomer `json:"customers"`
}

// ImpactStore 影响面查询口(PGStore 实现)。
type ImpactStore interface {
	ImpactByFacility(ctx context.Context, facilityCode string) (*ImpactReport, error)
}
