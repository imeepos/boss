package worker

import "errors"

// 装维队管理(000141)类型与错误。

// TeamMemberPerf 装维队成员月度绩效视图(后台/队长查看业绩统计)。
// 绩效行为"师傅×月×班组×区域"粒度,月中调队/换区拆多行,此处按成员聚合。
type TeamMemberPerf struct {
	WorkerID   int64   `json:"workerId"`
	StaffNo    string  `json:"staffNo"`
	Name       string  `json:"name"`
	Phone      string  `json:"phone"`
	IsLeader   bool    `json:"isLeader"`
	Finished   int32   `json:"finished"`   // 完成单量(该月多行求和)
	OnTimeRate int16   `json:"onTimeRate"` // 准时率%(多行均值)
	Score      float64 `json:"score"`      // 评分(多行均值)
}

// Transfer 成员调队请求(worker_group_memberships 台账,fields.md §7.2)。
type Transfer struct {
	WorkerID          int64  `json:"workerId"`
	TargetGroupID     int64  `json:"groupId"`
	Reason            string `json:"reason"` // 调队原因(可追溯)
	OperatorAccountID int64  `json:"operatorAccountId"`
}

var (
	// ErrGroupNotEmpty 队伍仍有在职成员,不允许软删。
	ErrGroupNotEmpty = errors.New("worker: group not empty")
	// ErrLeaderNotMember 指定队长不是本队在职成员。
	ErrLeaderNotMember = errors.New("worker: leader not active member of group")
	// ErrSameGroup 目标队伍与当前归属相同。
	ErrSameGroup = errors.New("worker: worker already in target group")
)
