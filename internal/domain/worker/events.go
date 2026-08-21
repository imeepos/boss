package worker

import (
	"context"
	"time"
)

// Material 师傅物料领用记录。
type Material struct {
	ID              int64  `json:"id"`
	WorkerID        int64  `json:"workerId"`
	GroupID         int64  `json:"groupId"`
	GroupName       string `json:"groupName"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	RegionID        int64  `json:"regionId"`
	RegionName      string `json:"regionName"`
	ItemID          int64  `json:"itemId"` // → material_items(E13);0=历史行未回填
	Name            string `json:"name"`
	Qty             int32  `json:"qty"`
}

// Tool 师傅工具借用记录。
type Tool struct {
	ID              int64  `json:"id"`
	WorkerID        int64  `json:"workerId"`
	GroupID         int64  `json:"groupId"`
	GroupName       string `json:"groupName"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	RegionID        int64  `json:"regionId"`
	RegionName      string `json:"regionName"`
	ToolID          int64  `json:"toolId"` // → material_tools(E13);0=历史行未回填
	Name            string `json:"name"`
	Borrowed        bool   `json:"borrowed"`
}

// Feedback 客户对工单师傅的服务评价。
type Feedback struct {
	ID              int64  `json:"id"`
	WorkerID        int64  `json:"workerId"`
	WorkerName      string `json:"workerName"`
	GroupID         int64  `json:"groupId"`
	GroupName       string `json:"groupName"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	RegionID        int64  `json:"regionId"`
	RegionName      string `json:"regionName"`
	TicketID        int64  `json:"ticketId"`
	CustomerID      int64  `json:"customerId"`
	CustomerName    string `json:"customerName"`
	Score           int16  `json:"score"` // 1~5
	NeedReview      bool   `json:"needReview"`
}

// AssetReturn 师傅资产归还记录。
type AssetReturn struct {
	ID              int64  `json:"id"`
	WorkerID        int64  `json:"workerId"`
	GroupID         int64  `json:"groupId"`
	GroupName       string `json:"groupName"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	RegionID        int64  `json:"regionId"`
	RegionName      string `json:"regionName"`
	AssetID         int64  `json:"assetId"`
	Reason          string `json:"reason"`
	Status          string `json:"status"` // PENDING/RETURNED
}

// WorkerEventService 师傅事件事实域服务口(阶段2)。
type WorkerEventService interface {
	ListMaterials(ctx context.Context, workerID int64) ([]Material, error)
	AppendMaterial(ctx context.Context, m Material) (int64, error)
	ListTools(ctx context.Context, workerID int64) ([]Tool, error)
	AppendTool(ctx context.Context, t Tool) (int64, error)
	ListFeedbacks(ctx context.Context, workerID int64) ([]Feedback, error)
	AppendFeedback(ctx context.Context, f Feedback) (int64, error)
	// ReviewFeedback 差评复核:need_review → false;未命中返回 ErrNotFound。
	ReviewFeedback(ctx context.Context, feedbackID int64) error
	ListAssetReturns(ctx context.Context, workerID int64) ([]AssetReturn, error)
	AppendAssetReturn(ctx context.Context, r AssetReturn) (int64, error)
	// ConfirmAssetReturn 确认返库:status PENDING → RETURNED;未命中或已返库返回 ErrNotFound。
	ConfirmAssetReturn(ctx context.Context, returnID int64) error
	// GetMaterialItem/GetTool 主档查询;未命中返回 ErrNotFound。
	GetMaterialItem(ctx context.Context, id int64) (*MaterialItem, error)
	GetTool(ctx context.Context, id int64) (*ToolItem, error)
	// ListToolItems 工具主档全量(师傅端借还列表)。
	ListToolItems(ctx context.Context) ([]ToolItem, error)
	// ListMaterialItems 物料主档全量(师傅端领料目录)。
	ListMaterialItems(ctx context.Context) ([]MaterialItem, error)
	// AppendReplaceLog/ListReplaceLogs 换件登记流水(师傅端换机页)。
	AppendReplaceLog(ctx context.Context, r ReplaceLog) (int64, error)
	ListReplaceLogs(ctx context.Context, ticketNo string) ([]ReplaceLog, error)
}

// ReplaceLog 换件登记流水。
type ReplaceLog struct {
	ID               int64     `json:"id"`
	WorkerID         int64     `json:"workerId"`
	DispatchTicketID int64     `json:"dispatchTicketId"` // → dispatch_tickets(E15);0=历史行未回填
	TicketNo         string    `json:"ticketNo"`         // 展示快照
	OldTagID         int64     `json:"oldTagId"`         // → tags(E15,按 EPC 解析);0=未命中
	NewTagID         int64     `json:"newTagId"`
	OldEpc           string    `json:"oldEpc"` // 展示快照
	NewEpc           string    `json:"newEpc"`
	CreatedAt        time.Time `json:"createdAt"`
}

// MaterialItem 物料主档。
type MaterialItem struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Spec string `json:"spec"`
	Unit string `json:"unit"`
}

// ToolItem 工具主档。
type ToolItem struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}
