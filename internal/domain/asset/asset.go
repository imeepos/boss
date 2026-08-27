package asset

import (
	"context"
	"time"
)

// AssetBatch 入库批次(公司采购行为,资产经此归属公司)。
type AssetBatch struct {
	ID            int64  `json:"id"`
	LegalEntityID int64  `json:"legalEntityId"`
	Code          string `json:"code"` // 批次编码,如 RK-202607-01
	Name          string `json:"name"`
}

// Tag 电子标签(公司库存,预绑定后才指向资产)。
type Tag struct {
	TagID         int64  `json:"tagId"`
	LegalEntityID int64  `json:"legalEntityId"`
	TagNo         string `json:"tagNo"`        // 标签编号
	EpcCode       string `json:"epcCode"`      // EPC 码
	Band          string `json:"band"`         // 频段,如 UHF
	BoundAssetID  int64  `json:"boundAssetId"` // 0=未绑定
	Status        string `json:"status"`       // UNBOUND/BOUND/DISABLED
	Battery       string `json:"battery"`
}

// Asset 资产台账(光猫/ONU 等装维物资的全生命周期)。
type Asset struct {
	AssetID         int64  `json:"assetId"`
	AssetCode       string `json:"assetCode"` // 资产编码,如 A-20260001
	BatchID         int64  `json:"batchId"`
	LegalEntityID   int64  `json:"legalEntityId"` // 企业归属快照
	LegalEntityName string `json:"legalEntityName"`
	TagID           int64  `json:"tagId"`     // 0=未绑定
	AddressID       int64  `json:"addressId"` // 0=未部署
	RegionID        int64  `json:"regionId"`  // 0=未部署
	RegionName      string `json:"regionName"`
	Type            string `json:"type"`   // 光猫/ONU/路由器
	Status          string `json:"status"` // IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED
}

// AssetLifecycle 资产状态轨迹(每次状态/位置变更一行,历史不随当前状态漂移)。
type AssetLifecycle struct {
	ID          int64     `json:"id"`
	AssetID     int64     `json:"assetId"`
	Status      string    `json:"status"`
	AddressID   int64     `json:"addressId"` // 0=空
	AddressName string    `json:"addressName"`
	WorkerID    int64     `json:"workerId"` // 0=空
	WorkerName  string    `json:"workerName"`
	ChangedAt   time.Time `json:"changedAt"`
}

// Replacement 换新单(故障资产换新流程)。
type Replacement struct {
	ID              int64  `json:"id"`
	ReplacementNo   string `json:"replacementNo"`
	AssetID         int64  `json:"assetId"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	Reason          string `json:"reason"`
	Priority        string `json:"priority"` // HIGH/MEDIUM/LOW
	Status          string `json:"status"`   // PENDING/DOING/DONE/FAILED
}

// AssetAssignment 资产持有台账(每次领用/部署/归还的时间段,历史不随当前值漂移)。
type AssetAssignment struct {
	ID                int64      `json:"id"`
	AssetID           int64      `json:"assetId"`
	WorkerID          int64      `json:"workerId"` // 0=空
	WorkerName        string     `json:"workerName"`
	AddressID         int64      `json:"addressId"` // 0=空
	AddressName       string     `json:"addressName"`
	Reason            string     `json:"reason"`
	OperatorAccountID int64      `json:"operatorAccountId"` // 0=空
	EffectiveFrom     time.Time  `json:"effectiveFrom"`
	EffectiveTo       *time.Time `json:"effectiveTo,omitempty"` // nil=至今
}

// Stocktake 盘点任务(按区域盘点资产,输出差异)。
type Stocktake struct {
	ID            int64  `json:"id"`
	LegalEntityID int64  `json:"legalEntityId"`
	Scope         string `json:"scope"`
	Progress      int16  `json:"progress"` // 0~100,实扫/计划快照行
	DiffCount     int32  `json:"diffCount"`
	Status        string `json:"status"` // DOING/DONE
}

// StocktakeItem 盘点差异明细(建单冻结快照,扫码回填,逐条处置)。
type StocktakeItem struct {
	ID             int64      `json:"id"`
	TaskID         int64      `json:"taskId"`
	AssetID        int64      `json:"assetId"`
	ExpectedStatus string     `json:"expectedStatus"` // ""=计划外(EXTRA 行)
	ScannedStatus  string     `json:"scannedStatus"`  // ""=未扫
	ScannedAt      *time.Time `json:"scannedAt,omitempty"`
	Kind           string     `json:"kind"`       // PENDING/OK/MISMATCH/MISSING/EXTRA
	Resolution     string     `json:"resolution"` // OPEN/CONFIRMED/FIXED/ESCALATED
	HandledBy      int64      `json:"handledBy"`  // 0=未处置
	HandledAt      *time.Time `json:"handledAt,omitempty"`
	Note           string     `json:"note"`
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
	// ListStocktakeItems 盘点差异明细清单。
	ListStocktakeItems(ctx context.Context, taskID int64) ([]StocktakeItem, error)
	// ScanStocktake 回填一次扫码,返回(明细 id, kind)。
	ScanStocktake(ctx context.Context, taskID, assetID int64, scannedStatus string) (int64, string, error)
	// HandleStocktakeItem 逐条处置差异(action: CONFIRM/FIX/ESCALATE)。
	HandleStocktakeItem(ctx context.Context, taskID, itemID int64, action, note string, accountID int64) error
	// HandleStocktakeDiff 关单:未处置差异非 0 时拒绝,全处置完任务置 DONE。
	HandleStocktakeDiff(ctx context.Context, taskID int64) error

	ListAssignments(ctx context.Context, assetID int64) ([]AssetAssignment, error)
	AssignAsset(ctx context.Context, a AssetAssignment) (int64, error)
}
