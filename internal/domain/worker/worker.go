package worker

import (
	"context"
	"errors"
	"time"
)

// PasswordMin 师傅端登录密码最短长度(与 accounts 侧 PasswordMin 对齐)。
const PasswordMin = 6

// ErrDuplicate 工号/手机号已存在(workers.staff_no 唯一约束冲突)。
var ErrDuplicate = errors.New("worker: duplicate staff no or phone")

// ErrInvalidPassword 登录密码长度不合法。
var ErrInvalidPassword = errors.New("worker: invalid password")

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

// WorkerService 师傅域服务口(阶段2):班组/师傅。
type WorkerService interface {
	ListGroups(ctx context.Context) ([]Group, error)
	CreateGroup(ctx context.Context, g Group) (int64, error)
	// keyword 空=不过滤;命中姓名/工号/手机号任一(不区分大小写)。
	ListWorkers(ctx context.Context, groupID int64, keyword string) ([]Worker, error)
	CreateWorker(ctx context.Context, w Worker) (int64, error)
	GetWorker(ctx context.Context, id int64) (*Worker, error)
	// MatchedRegionIDs 区域子树匹配(祖先或自身):工单区域须落在师傅负责区域
	// 子树内(ltree path <@),对齐 H3 式"粗区域先行"分层派单口径;
	// 师傅区域 0=不限区域全放行,工单区域 0=无区域放行(历史口径)。
	// 返回映射只含判定为 true 的工单区域 id。
	MatchedRegionIDs(ctx context.Context, workerRegionID int64, ticketRegionIDs []int64) (map[int64]bool, error)
	// CreateWorkerWithPassword 新建师傅并写入登录密码(admin 录入);password 为空=不可密码登录。
	CreateWorkerWithPassword(ctx context.Context, w Worker, password string) (int64, error)
	// SetPassword 重置师傅登录密码(bcrypt 落库);长度 < PasswordMin 返回 ErrInvalidPassword。
	SetPassword(ctx context.Context, workerID int64, password string) error
	// VerifyPassword 校验师傅登录密码(仅读哈希,不暴露 password_hash);未设置/不匹配返回 false。
	VerifyPassword(ctx context.Context, workerID int64, password string) (bool, error)
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
