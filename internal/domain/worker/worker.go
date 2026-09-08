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

// ErrScanBindRequired 激活/签收前置缺失:未扫码绑定(环节9)不可推进。
var ErrScanBindRequired = errors.New("worker: scan bind required")

// ErrGroupInvalid 师傅班组快照不可用(主档缺失/无有效班组归属)。
var ErrGroupInvalid = errors.New("worker: group invalid")

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
// RegionIDs 全部负责区域(主区域首位;迁移 000175 起一师傅可配置多区域)。
type Worker struct {
	ID         int64      `json:"id"`
	StaffNo    string     `json:"staffNo"`
	Name       string     `json:"name"`
	GroupID    int64      `json:"groupId"`
	GroupName  string     `json:"groupName"` // 班组名(读取时 JOIN 现值,data-relations §6.2)
	RegionID   int64      `json:"regionId"`
	RegionName string     `json:"regionName"` // 主区域名(读取时 JOIN 现值,data-relations §6.2)
	RegionIDs  []int64    `json:"regionIds"`
	Phone      string     `json:"phone"`
	Status     int16      `json:"status"` // 1在职 0离职
	JoinedAt   time.Time  `json:"joinedAt"`
	LeftAt     *time.Time `json:"leftAt,omitempty"` // nil=在职
}

// MatchesRegion 工单区域是否落在师傅负责区域内(主区域 ∪ 扩展区域,000175):
// 工单无区域(0)=不限;师傅未设任何区域=不限;否则命中任一负责区域即匹配。
func (w *Worker) MatchesRegion(ticketRegionID int64) bool {
	if ticketRegionID == 0 || (w.RegionID == 0 && len(w.RegionIDs) == 0) {
		return true
	}
	if w.RegionID > 0 && w.RegionID == ticketRegionID {
		return true
	}
	for _, id := range w.RegionIDs {
		if id == ticketRegionID {
			return true
		}
	}
	return false
}

// NormalizeRegionIDs 负责区域入参归一:去重、去非正、主区域首位
// (primaryID<=0 时取首个合法区域为主区域);返回归一后集合与主区域。
func NormalizeRegionIDs(regionIDs []int64, primaryID int64) ([]int64, int64) {
	seen := make(map[int64]bool, len(regionIDs))
	out := make([]int64, 0, len(regionIDs))
	for _, id := range regionIDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	primary := primaryID
	if len(out) == 0 {
		return nil, primary
	}
	if primary <= 0 || !seen[primary] {
		primary = out[0]
		return out, primary
	}
	ordered := make([]int64, 0, len(out))
	ordered = append(ordered, primary)
	for _, id := range out {
		if id != primary {
			ordered = append(ordered, id)
		}
	}
	return ordered, primary
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
	// 集合(主区域 ∪ 扩展区域,000175)任一子树内(ltree path <@),对齐 H3 式
	// "粗区域先行"分层派单口径;师傅未设任何区域=不限全放行,工单区域 0=放行
	// (历史口径)。返回映射只含判定为 true 的工单区域 id。
	MatchedRegionIDs(ctx context.Context, w *Worker, ticketRegionIDs []int64) (map[int64]bool, error)
	// CreateWorkerWithPassword 新建师傅并写入登录密码(admin 录入);password 为空=不可密码登录。
	CreateWorkerWithPassword(ctx context.Context, w Worker, password string) (int64, error)
	// SetPassword 重置师傅登录密码(bcrypt 落库);长度 < PasswordMin 返回 ErrInvalidPassword。
	SetPassword(ctx context.Context, workerID int64, password string) error
	// VerifyPassword 校验师傅登录密码(仅读哈希,不暴露 password_hash);未设置/不匹配返回 false。
	VerifyPassword(ctx context.Context, workerID int64, password string) (bool, error)
	// SetWorkerRegions 覆盖式配置师傅负责区域(迁移 000175);
	// 首位为主区域(回写 workers.region_id),空集合=清空(不限区域)。
	SetWorkerRegions(ctx context.Context, workerID int64, regionIDs []int64) error
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
