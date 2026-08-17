package asset

import "context"

// AssetBatch 入库批次(公司采购行为,资产经此归属公司)。
type AssetBatch struct {
	ID            int64
	LegalEntityID int64
	Code          string // 批次编码,如 RK-202607-01
	Name          string
}

// Tag 电子标签(公司库存,预绑定后才指向资产)。
type Tag struct {
	TagID         int64
	LegalEntityID int64
	TagNo         string // 标签编号
	EpcCode       string // EPC 码
	Band          string // 频段,如 UHF
	BoundAssetID  int64  // 0=未绑定
	Status        string // UNBOUND/BOUND/DISABLED
	Battery       string
}

// Asset 资产台账(光猫/ONU 等装维物资的全生命周期)。
type Asset struct {
	AssetID         int64
	AssetCode       string // 资产编码,如 A-20260001
	BatchID         int64
	LegalEntityID   int64  // 企业归属快照
	LegalEntityName string
	TagID           int64  // 0=未绑定
	AddressID       int64  // 0=未部署
	RegionID        int64  // 0=未部署
	RegionName      string
	Type            string // 光猫/ONU/路由器
	Status          string // IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED
}

// AssetService 资产域服务口(阶段3):入库批次/电子标签/资产台账。
type AssetService interface {
	ListBatches(ctx context.Context) ([]AssetBatch, error)
	CreateBatch(ctx context.Context, b AssetBatch) (int64, error)
	ListTags(ctx context.Context) ([]Tag, error)
	CreateTag(ctx context.Context, t Tag) (int64, error)
	ListAssets(ctx context.Context) ([]Asset, error)
	CreateAsset(ctx context.Context, a Asset) (int64, error)
	GetAsset(ctx context.Context, id int64) (*Asset, error)
}
