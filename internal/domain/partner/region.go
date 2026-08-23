package partner

import "context"

type RegionScope struct {
	LegalEntityID int64  `json:"legalEntityId"`
	RegionPath    string `json:"regionPath"`
}

type RegionScopeService interface {
	GetRegionScope(ctx context.Context, accountID int64) (RegionScope, error)
	SetRegionScope(ctx context.Context, adminAccountID int64, regionPath string) error
}
