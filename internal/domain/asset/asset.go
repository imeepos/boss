package asset

import (
	"context"
	"time"
)

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

// AssetLifecycle 资产状态轨迹(每次状态/位置变更一行,历史不随当前状态漂移)。
type AssetLifecycle struct {
	ID          int64
	AssetID     int64
	Status      string
	AddressID   int64 // 0=空
	AddressName string
	WorkerID    int64 // 0=空
	WorkerName  string
	ChangedAt   time.Time
}

// Replacement 换新单(故障资产换新流程)。
type Replacement struct {
	ID              int64
	ReplacementNo   string
	AssetID         int64
	LegalEntityID   int64
	LegalEntityName string
	Reason          string
	Priority        string // HIGH/MEDIUM/LOW
	Status          string // PENDING/DOING/DONE/FAILED
}

// AssetAssignment 资产持有台账(每次领用/部署/归还的时间段,历史不随当前值漂移)。
type AssetAssignment struct {
	ID                int64
	AssetID           int64
	WorkerID          int64 // 0=空
	WorkerName        string
	AddressID         int64 // 0=空
	AddressName       string
	Reason            string
	OperatorAccountID int64      // 0=空
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time // nil=至今
}

// Stocktake 盘点任务(按区域盘点资产,输出差异)。
type Stocktake struct {
	ID            int64
	LegalEntityID int64
	Scope         string
	Progress      int16 // 0~100
	DiffCount     int32
	Status        string // DOING/DONE
}

// AssetService 资产域服务口(阶段3):入库批次/电子标签/资产台账/状态轨迹/换新/盘点。
type AssetService interface {
	ListBatches(ctx context.Context) ([]AssetBatch, error)
	CreateBatch(ctx context.Context, b AssetBatch) (int64, error)
	ListTags(ctx context.Context) ([]Tag, error)
	CreateTag(ctx context.Context, t Tag) (int64, error)
	ListAssets(ctx context.Context) ([]Asset, error)
	CreateAsset(ctx context.Context, a Asset) (int64, error)
	GetAsset(ctx context.Context, id int64) (*Asset, error)

	ListLifecycles(ctx context.Context, assetID int64) ([]AssetLifecycle, error)
	AppendLifecycle(ctx context.Context, l AssetLifecycle) (int64, error)
	ListReplacements(ctx context.Context) ([]Replacement, error)
	CreateReplacement(ctx context.Context, r Replacement) (int64, error)
	ListStocktakes(ctx context.Context) ([]Stocktake, error)
	CreateStocktake(ctx context.Context, s Stocktake) (int64, error)

	ListAssignments(ctx context.Context, assetID int64) ([]AssetAssignment, error)
	AssignAsset(ctx context.Context, a AssetAssignment) (int64, error)
}
