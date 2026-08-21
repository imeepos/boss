package worker

import (
	"context"
	"time"
)

// Group 师傅班组(运营主体自定义组织,公司内 code 唯一)。
type Group struct {
	ID            int64  `json:"id"`
	LegalEntityID int64  `json:"legalEntityId"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	LeaderID      int64  `json:"leaderId"` // 0=无组长
	LeaderName    string `json:"leaderName"`
}

// Worker 装维师傅(归属班组,服务区域须落班组公司经营区域)。
type Worker struct {
	ID       int64      `json:"id"`
	StaffNo  string     `json:"staffNo"`
	Name     string     `json:"name"`
	GroupID  int64      `json:"groupId"`
	RegionID int64      `json:"regionId"`
	Phone    string     `json:"phone"`
	Status   int16      `json:"status"` // 1在职 0离职
	JoinedAt time.Time  `json:"joinedAt"`
	LeftAt   *time.Time `json:"leftAt,omitempty"` // nil=在职
}

// RegionMatched 师傅与工单区域是否匹配:任一方区域缺失(0=无区域)视为匹配
// (未设区域=不限区域);双方均非空且不同才算跨区。
func RegionMatched(workerRegionID, ticketRegionID int64) bool {
	if workerRegionID == 0 || ticketRegionID == 0 {
		return true
	}
	return workerRegionID == ticketRegionID
}

// WorkerService 师傅域服务口(阶段2):班组/师傅。
type WorkerService interface {
	ListGroups(ctx context.Context) ([]Group, error)
	CreateGroup(ctx context.Context, g Group) (int64, error)
	ListWorkers(ctx context.Context, groupID int64) ([]Worker, error)
	CreateWorker(ctx context.Context, w Worker) (int64, error)
	GetWorker(ctx context.Context, id int64) (*Worker, error)
}
