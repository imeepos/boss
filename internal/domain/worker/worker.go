package worker

import (
	"context"
	"time"
)

// Group 师傅班组(UI 别名"装维队",运营主体自定义组织,公司内 code 唯一)。
type Group struct {
	ID            int64  `json:"id"`
	LegalEntityID int64  `json:"legalEntityId"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	LeaderID      int64  `json:"leaderId"` // 0=无队长
	LeaderName    string `json:"leaderName"`
	MemberCount   int    `json:"memberCount"` // 在职成员数(装维队列表视图冗余)
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
	// keyword 空=不过滤;命中姓名/工号/手机号任一(不区分大小写)。
	ListWorkers(ctx context.Context, groupID int64, keyword string) ([]Worker, error)
	CreateWorker(ctx context.Context, w Worker) (int64, error)
	GetWorker(ctx context.Context, id int64) (*Worker, error)
}

// TeamService 装维队管理服务口(000141):队伍维护/成员调队/队长绩效视图。
// 队伍实体即 Group(班组),UI 术语"装维队/队长"对应契约"班组/组长"。
type TeamService interface {
	// UpdateGroup 改名 + 指定队长(leaderID=0 清空);队长必须为本队在职成员。
	UpdateGroup(ctx context.Context, groupID int64, name string, leaderID int64) (*Group, error)
	// SoftDeleteGroup 软删队伍(deleted_at 置位);仍有在职成员时拒绝(fields.md §7.3 铁律 1)。
	SoftDeleteGroup(ctx context.Context, groupID int64) error
	// TransferWorker 成员调队:改 workers.group_id + 落 worker_group_memberships 台账。
	TransferWorker(ctx context.Context, t Transfer) error
	// ListTeamPerformances 队成员月度绩效(队长/后台业绩统计);period 空=全部月。
	ListTeamPerformances(ctx context.Context, groupID int64, period string) ([]TeamMemberPerf, error)
	// GroupOfLeader 查队长所辖队伍;非队长返回 ErrNotFound。
	GroupOfLeader(ctx context.Context, workerID int64) (*Group, error)
}
