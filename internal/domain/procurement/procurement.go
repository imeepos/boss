package procurement

import (
	"context"
	"errors"
	"time"
)

// 错误集。
var (
	ErrNotFound          = errors.New("procurement: not found")
	ErrInvalidTransition = errors.New("procurement: invalid status transition")
	ErrForeignKey        = errors.New("procurement: foreign key violation")
	// ErrStateConflict 状态不允许该操作(草稿编辑非 DRAFT/驳回非 DRAFT 等):HTTP 40900。
	// 与既有 ErrInvalidTransition(映射 42200)区分,不改变既有 submit/cancel 端点行为。
	ErrStateConflict = errors.New("procurement: state conflict")
	// ErrInvalidInput 入参不合法(明细行数量<=0/驳回原因超 255 字等):HTTP 42200。
	ErrInvalidInput = errors.New("procurement: invalid input")
)

// 供应商承建类型(000203):材料类为存量默认,语义与既有档案一致。
const (
	ContractorMaterial     = "MATERIAL"     // 材料类
	ContractorConstruction = "CONSTRUCTION" // 施工类(资质信息落 qualification)
)

// Supplier 供应商(L1.5 公司自定义基础数据)。
type Supplier struct {
	ID           int64  `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	ContactName  string `json:"contactName"`
	ContactPhone string `json:"contactPhone"`
	// 承建类型维度(000203):施工类含资质信息;联系人复用上方 contact 字段。
	ContractorType string    `json:"contractorType"`
	Qualification  string    `json:"qualification"`
	LegalEntityID  int64     `json:"legalEntityId"`
	Status         string    `json:"status"` // ENABLED/DISABLED
	Remark         string    `json:"remark"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// OrderItem 采购单明细。
type OrderItem struct {
	ID           int64     `json:"id"`
	OrderID      int64     `json:"orderId"`
	MaterialCode string    `json:"materialCode"`
	Spec         string    `json:"spec"`
	Quantity     int32     `json:"quantity"`
	ReceivedQty  int32     `json:"receivedQty"`
	UnitAmount   float64   `json:"unitAmount"`
	Remark       string    `json:"remark"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Order 采购单头(状态机 DRAFT→SUBMITTED→PARTIAL→RECEIVED;任意点 CANCELLED)。
type Order struct {
	ID              int64       `json:"id"`
	ProcurementNo   string      `json:"procurementNo"`
	LegalEntityID   int64       `json:"legalEntityId"`
	LegalEntityName string      `json:"legalEntityName"`
	SupplierID      int64       `json:"supplierId"`
	SupplierName    string      `json:"supplierName"`
	Status          string      `json:"status"` // DRAFT/SUBMITTED/PARTIAL/RECEIVED/CANCELLED
	TotalAmount     float64     `json:"totalAmount"`
	ExpectedDate    *time.Time  `json:"expectedDate,omitempty"`
	Remark          string      `json:"remark"`
	CreatedBy       int64       `json:"createdBy"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
	SubmittedAt     *time.Time  `json:"submittedAt,omitempty"`
	ReceivedAt      *time.Time  `json:"receivedAt,omitempty"`
	CancelledAt     *time.Time  `json:"cancelledAt,omitempty"`
	Items           []OrderItem `json:"items,omitempty"`
}

// Receipt 到货入库单(status=DRAFT/CONFIRMED/REJECTED)。
// CONFIRMED 同事务建 asset_batches + 逐台 assets IN_STOCK,资产经此进入库存。
type Receipt struct {
	ID              int64     `json:"id"`
	ReceiptNo       string    `json:"receiptNo"`
	OrderID         int64     `json:"orderId"`
	OrderNo         string    `json:"orderNo"`
	BatchID         int64     `json:"batchId"` // 0=入库后回填
	LegalEntityID   int64     `json:"legalEntityId"`
	LegalEntityName string    `json:"legalEntityName"`
	ReceivedBy      int64     `json:"receivedBy"`
	ReceivedAt      time.Time `json:"receivedAt"`
	Status          string    `json:"status"` // DRAFT/CONFIRMED/REJECTED
	Remark          string    `json:"remark"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// ReceiptItem 入库单明细(每行:物料 code + 数量,生成 N 条 assets)。
type ReceiptItem struct {
	MaterialCode string  `json:"materialCode"`
	Spec         string  `json:"spec"`
	Quantity     int32   `json:"quantity"`
	UnitAmount   float64 `json:"unitAmount"`
}

// ReceiptConfirmInput 入库确认入参。
type ReceiptConfirmInput struct {
	BatchCode    string        `json:"batchCode"`    // asset_batches.code,后端生成兜底
	BatchName    string        `json:"batchName"`    // asset_batches.name
	WarehouseLat *float64      `json:"warehouseLat"` // 可空
	WarehouseLng *float64      `json:"warehouseLng"` // 可空
	Items        []ReceiptItem `json:"items"`
}

// InventoryRow 库存视图(按物料 code 聚合 IN_STOCK 数量,实时聚合)。
type InventoryRow struct {
	MaterialCode string `json:"materialCode"`
	BatchID      int64  `json:"batchId"`
	InStockQty   int32  `json:"inStockQty"`
}

// Service 采购-库存域服务口。
type Service interface {
	// 供应商
	CreateSupplier(ctx context.Context, s Supplier) (int64, error)
	ListSuppliers(ctx context.Context, legalEntityID int64) ([]Supplier, error)
	GetSupplier(ctx context.Context, id int64) (*Supplier, error)
	DisableSupplier(ctx context.Context, id int64) error

	// 采购单
	CreateOrder(ctx context.Context, o Order) (int64, error)
	GetOrder(ctx context.Context, id int64) (*Order, error)
	ListOrders(ctx context.Context, legalEntityID int64, status string) ([]Order, error)
	SubmitOrder(ctx context.Context, id int64) error
	CancelOrder(ctx context.Context, id int64) error

	// 入库单
	CreateReceipt(ctx context.Context, r Receipt, items []ReceiptItem) (int64, error)
	ConfirmReceipt(ctx context.Context, receiptID, accountID int64, in ReceiptConfirmInput) error
	ListReceipts(ctx context.Context, orderID int64) ([]Receipt, error)

	// 库存查询(实时聚合)
	ListInventory(ctx context.Context, legalEntityID int64, materialCode string) ([]InventoryRow, error)

	// 供应商编辑/启用(P2-W2-T2):编辑部分更新(编码不可改,禁用态可改);启用幂等
	UpdateSupplier(ctx context.Context, id int64, in SupplierUpdate) error
	EnableSupplier(ctx context.Context, id int64) error

	// 采购单草稿编辑/详情(P2-W2-T2):仅 DRAFT 可编辑;详情含明细行
	UpdateOrderDraft(ctx context.Context, id int64, in OrderDraftUpdate) error
	GetOrderDetail(ctx context.Context, id int64) (*Order, error)

	// 入库驳回(P2-W2-T2):仅 DRAFT 可驳回,置 REJECTED;审计由 admin handler 层补记
	RejectReceipt(ctx context.Context, receiptID int64, reason string) error

	// 库存事务钩子(供 cmd/gis 消费器调,广播 inventory.changed 事件):
	// 业务侧不必调用,ConfirmReceipt 内部已发。
}
