package quadlink

import "context"

// QuadLink 四码合一(资产-客户-端口-地址 四码关联,任一码反查单表索引)。
type QuadLink struct {
	ID              int64
	AssetID         int64
	CustomerID      int64
	PortID          int64
	AddressID       int64
	LegalEntityID   int64
	LegalEntityName string
	Status          string // LINKED/CONFLICT/UNLINKED
}

// QuadLinkService 四码合一域服务口(阶段6)。
type QuadLinkService interface {
	ListLinks(ctx context.Context) ([]QuadLink, error)
	CreateLink(ctx context.Context, q QuadLink) (int64, error)
	GetByAsset(ctx context.Context, assetID int64) (*QuadLink, error)
	GetByCustomer(ctx context.Context, customerID int64) (*QuadLink, error)
	GetByPort(ctx context.Context, portID int64) (*QuadLink, error)
	GetByAddress(ctx context.Context, addressID int64) (*QuadLink, error)
}
