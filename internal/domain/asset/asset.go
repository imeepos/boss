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

// TagEvent 标签绑定事件(append-only 审计流,P1-T2):BIND/UNBIND/RECYCLE,
// 形态依据 R2 调研(Snipe-IT action_logs / bk-cmdb cc_AuditLog)。
type TagEvent struct {
	ID             int64          `json:"id"`
	EventID        string         `json:"eventId"`
	TagID          int64          `json:"tagId"`
	AssetID        int64          `json:"assetId"`
	Action         string         `json:"action"`
	ActorAccountID int64          `json:"actorAccountId"`
	Detail         string         `json:"detail"`
	Changed        map[string]any `json:"changed,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
}

// AssetModel 资产型号字典(P1-T3,R3 调研:NetBox DeviceType/GLPI models 同款形态):
// UNIQUE(vendor,model,category,part_number) 防重;is_active 停用不物理删(引用保护)。
type AssetModel struct {
	ID         int64          `json:"id"`
	Vendor     string         `json:"vendor"`
	Model      string         `json:"model"`
	Category   string         `json:"category"`
	PartNumber string         `json:"partNumber"`
	Spec       map[string]any `json:"spec,omitempty"`
	IsActive   bool           `json:"isActive"`
	CreatedAt  time.Time      `json:"createdAt"`
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
	Type            string `json:"type"`    // 光猫/ONU/路由器(展示冗余;权威=model_id→asset_models.category)
	ModelID         int64  `json:"modelId"` // 0=未挂型号(P1-T3)
	Status          string `json:"status"`  // IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED
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
// 状态机: PENDING --Assign--> DOING --Complete--> DONE/FAILED(adopted note
// 2026-08-27-replacement-ticket-flow);终态不可再流转,重做走新单。
type Replacement struct {
	ID              int64      `json:"id"`
	ReplacementNo   string     `json:"replacementNo"`
	AssetID         int64      `json:"assetId"`
	LegalEntityID   int64      `json:"legalEntityId"`
	LegalEntityName string     `json:"legalEntityName"`
	Reason          string     `json:"reason"`
	Priority        string     `json:"priority"` // HIGH/MEDIUM/LOW
	Status          string     `json:"status"`   // PENDING/DOING/DONE/FAILED
	WorkerID        int64      `json:"workerId"` // 0=未派
	WorkerName      string     `json:"workerName"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"` // nil=未完成
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
	// UpdateAsset 受限编辑(P2-W1-T1):仅 类型/型号/标签/批次 四键;标签换绑同一
	// 事务写 UNBIND+BIND(冲突 ErrBindingConflict 整单回滚);批次仅 IN_STOCK 可改
	// (ErrBatchNotEditable)并同步企业归属快照;四键无变化幂等成功。
	UpdateAsset(ctx context.Context, assetID int64, in AssetUpdate, actorAccountID int64) error
	// DeleteAsset 守卫删除(P2-W1-T1):仅 IN_STOCK 且无标签绑定/持有台账/换新单/
	// 盘点明细/四码关联引用可物理删除;命中引用 ErrAssetReferenced(message 全量
	// 列阻断项);SCRAPPED ErrAssetScrapped 拒硬删。返回资产编码供审计载荷。
	DeleteAsset(ctx context.Context, assetID int64) (string, error)

	ListLifecycles(ctx context.Context, assetID int64) ([]AssetLifecycle, error)
	AppendLifecycle(ctx context.Context, l AssetLifecycle) (int64, error)
	// SetAssetStatus 直改资产当前状态(换新完成联动);未命中返回 ErrNotFound。
	SetAssetStatus(ctx context.Context, assetID int64, status string) error
	ListReplacements(ctx context.Context) ([]Replacement, error)
	CreateReplacement(ctx context.Context, r Replacement) (int64, error)
	// GetReplacement 按 id 查换新单;未命中返回 ErrNotFound。
	GetReplacement(ctx context.Context, id int64) (*Replacement, error)
	// ListReplacementsByWorker 列出师傅名下的换新单(师傅端任务列表)。
	ListReplacementsByWorker(ctx context.Context, workerID int64) ([]Replacement, error)
	// AssignReplacement 派单:回填师傅快照并 PENDING→DOING;
	// 单不存在返回 ErrNotFound,状态非 PENDING 返回 ErrInvalidTransition。
	AssignReplacement(ctx context.Context, id, workerID int64, workerName string) (*Replacement, error)
	// CompleteReplacement 完成/失败:DOING→DONE|FAILED 并回填 finished_at;
	// 状态非 DOING 返回 ErrInvalidTransition。result 仅接受 DONE/FAILED。
	CompleteReplacement(ctx context.Context, id int64, result string) (*Replacement, error)
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

	// UnbindTag 解绑标签(P1-T2):置 bound_asset_id=NULL+status=UNBOUND 并写 UNBIND 事件;
	// expectedAssetID>0 时校验当前绑定一致;未绑定/预期不符返回 ErrTagUnbound/ErrBindingConflict。
	UnbindTag(ctx context.Context, tagID, expectedAssetID, actorAccountID int64, detail string) error
	// DisableTag 停用标签(P2-W2-T1):仅 UNBOUND 可停用(BOUND 必须先解绑,
	// ErrBindingConflict 40900);已 DISABLED 幂等成功;未命中 ErrNotFound。
	DisableTag(ctx context.Context, tagID int64, reason string) error
	// EnableTag 启用标签(P2-W2-T1):仅对 DISABLED 生效(DISABLED → UNBOUND);
	// UNBOUND/BOUND 幂等 no-op 成功;未命中 ErrNotFound。
	EnableTag(ctx context.Context, tagID int64) error
	// ListTagEvents 标签事件流(P2-W2-T1):append-only 审计流只读回放,按时间倒序。
	ListTagEvents(ctx context.Context, tagID int64) ([]TagEvent, error)
	// ScrapAsset 报废资产(P1-T2):任意非终态 → SCRAPPED(终态幂等 no-op),强制解绑标签写
	// RECYCLE 事件(软回收禁硬删),轨迹落行;同一事务,失败整单回滚。
	ScrapAsset(ctx context.Context, assetID, actorAccountID int64, reason string) error

	// ListModels 型号字典(含停用,管理端下拉与列表)。
	ListModels(ctx context.Context) ([]AssetModel, error)
	// CreateModel 建型号:UNIQUE(vendor,model,category,part_number) 冲突返回 ErrModelExists。
	CreateModel(ctx context.Context, m AssetModel) (int64, error)
	// UpdateModel 编辑型号(P2-W2-T1):厂商/型号名/类别/料号/规格可改;唯一冲突
	// ErrModelExists(40900);停用型号拒绝编辑 ErrModelInactive(40900,先启用)。
	UpdateModel(ctx context.Context, id int64, m AssetModel) error
}
