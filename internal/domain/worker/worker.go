package worker

import (
	"context"
	"time"
)

// Group 师傅班组(运营主体自定义组织,公司内 code 唯一)。
type Group struct {
	ID            int64
	LegalEntityID int64
	Code          string
	Name          string
	LeaderID      int64 // 0=无组长
	LeaderName    string
}

// Worker 装维师傅(归属班组,服务区域须落班组公司经营区域)。
type Worker struct {
	ID       int64
	StaffNo  string
	Name     string
	GroupID  int64
	RegionID int64
	Phone    string
	Status   int16      // 1在职 0离职
	JoinedAt time.Time
	LeftAt   *time.Time // nil=在职
}

// WorkerService 师傅域服务口(阶段2):班组/师傅。
type WorkerService interface {
	ListGroups(ctx context.Context) ([]Group, error)
	CreateGroup(ctx context.Context, g Group) (int64, error)
	ListWorkers(ctx context.Context, groupID int64) ([]Worker, error)
	CreateWorker(ctx context.Context, w Worker) (int64, error)
	GetWorker(ctx context.Context, id int64) (*Worker, error)
}
