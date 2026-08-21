package quadlink

import (
	"context"
)

// QuadLink 四码合一(资产-客户-端口-地址 四码关联,任一码反查单表索引)。
// AssetID 在预绑定阶段可为 0/空(扫码环节再回填),migration 000086 改为允许 NULL。
type QuadLink struct {
	ID              int64  `json:"id"`
	AssetID         int64  `json:"assetId"` // 预绑定阶段可为 0,扫码绑定(环节9)回填真实资产 ID
	CustomerID      int64  `json:"customerId"`
	PortID          int64  `json:"portId"`
	AddressID       int64  `json:"addressId"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	Status          string `json:"status"` // LINKED/CONFLICT/UNLINKED
}

// ErrForeignKeyViolation 四码关联中某个成员 ID 对应的实体不存在(assets/customers/ports/addresses/legal_entities);定义见 pg.go。

// QuadLinkService 四码合一域服务口(阶段6)。
type QuadLinkService interface {
	ListLinks(ctx context.Context) ([]QuadLink, error)
	CreateLink(ctx context.Context, q QuadLink) (int64, error)
	GetByAsset(ctx context.Context, assetID int64) (*QuadLink, error)
	GetByCustomer(ctx context.Context, customerID int64) (*QuadLink, error)
	GetByPort(ctx context.Context, portID int64) (*QuadLink, error)
	GetByAddress(ctx context.Context, addressID int64) (*QuadLink, error)

	// VerifyScan 扫码绑定(环节9 强制):实物 EPC ↔ 预绑定资产核对;不一致返回 ErrScanMismatch。
	VerifyScan(ctx context.Context, req ScanReq) (string, error)
	// UnbindRequireScan 拆机必扫码:不扫码(ErrScanRequired)/不一致(ErrScanMismatch)均拒;一致则解绑。
	UnbindRequireScan(ctx context.Context, orderID int64, scannedEPC string) error
	// Reconcile 四码对账任务:成员缺失置 CONFLICT,返回状态统计。
	Reconcile(ctx context.Context) (*ReconcileReport, error)
	// ResolveConflict 冲突人工处理:CONFLICT → UNLINKED(非冲突态拒)。
	ResolveConflict(ctx context.Context, linkID int64) error
	// PurgeOrphans 删除所有孤儿 quad_link 行(成员不存在则删),返回删除条数。
	PurgeOrphans(ctx context.Context) (int64, error)
}
