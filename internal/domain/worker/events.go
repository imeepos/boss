package worker

import "context"

// Material 师傅物料领用记录。
type Material struct {
	ID              int64
	WorkerID        int64
	GroupID         int64
	GroupName       string
	LegalEntityID   int64
	LegalEntityName string
	RegionID        int64
	RegionName      string
	Name            string
	Qty             int32
}

// Tool 师傅工具借用记录。
type Tool struct {
	ID              int64
	WorkerID        int64
	GroupID         int64
	GroupName       string
	LegalEntityID   int64
	LegalEntityName string
	RegionID        int64
	RegionName      string
	Name            string
	Borrowed        bool
}

// Feedback 客户对工单师傅的服务评价。
type Feedback struct {
	ID              int64
	WorkerID        int64
	WorkerName      string
	GroupID         int64
	GroupName       string
	LegalEntityID   int64
	LegalEntityName string
	RegionID        int64
	RegionName      string
	TicketID        int64
	CustomerID      int64
	CustomerName    string
	Score           int16 // 1~5
	NeedReview      bool
}

// AssetReturn 师傅资产归还记录。
type AssetReturn struct {
	ID              int64
	WorkerID        int64
	GroupID         int64
	GroupName       string
	LegalEntityID   int64
	LegalEntityName string
	RegionID        int64
	RegionName      string
	AssetID         int64
	Reason          string
	Status          string // PENDING/DONE
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
