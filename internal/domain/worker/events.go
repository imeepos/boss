package worker

import "context"

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
	Status          string `json:"status"` // PENDING/DONE
}

// WorkerEventService 师傅事件事实域服务口(阶段2)。
type WorkerEventService interface {
	ListMaterials(ctx context.Context, workerID int64) ([]Material, error)
	AppendMaterial(ctx context.Context, m Material) (int64, error)
	ListTools(ctx context.Context, workerID int64) ([]Tool, error)
	AppendTool(ctx context.Context, t Tool) (int64, error)
	ListFeedbacks(ctx context.Context, workerID int64) ([]Feedback, error)
	AppendFeedback(ctx context.Context, f Feedback) (int64, error)
	ListAssetReturns(ctx context.Context, workerID int64) ([]AssetReturn, error)
	AppendAssetReturn(ctx context.Context, r AssetReturn) (int64, error)
}
