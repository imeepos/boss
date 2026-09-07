package odn

// ROW 路权与 PECE 许可单(P-INFRA-1 W4,迁移 000211;审查 F3)。
// 状态机登记 terms.md 4;ROW/PECE 字典与资源链 row_status/pece_status(000210)对齐。
// 开工前置门控:EvaluatePermitGate 纯函数判定,PGStore.CheckProjectPermits 落数据与自动过期。

import (
	"context"
	"errors"
	"fmt"
)

// 许可单类型。
const (
	PermitKindROW  = "ROW"  // 路权许可
	PermitKindPECE = "PECE" // PECE 许可
)

// ROW 许可状态。
const (
	PRowNotStarted = "NOT_STARTED" // 未开始
	PRowPending    = "PENDING"     // 待处理(申请中/补件重报/过期复验中)
	PRowApproved   = "APPROVED"    // 已批准
	PRowExpired    = "EXPIRED"     // 已过期
	PermitNA       = "NA"          // 不适用(全室内敷设等;PECE 同用)
)

// PECE 许可状态。
const (
	PPecePendingSign = "PENDING_SIGN" // 待签署
	PPeceSigned      = "SIGNED"       // 已签署
	PPeceStamped     = "STAMPED"      // 已盖章(开工门控满足态)
)

// 领域错误。
var (
	// ErrPermitState 非法许可状态转移(未知/跨类/回退):40900。
	ErrPermitState = errors.New("odn: invalid permit status transition")
	// ErrPermitRequired 施工开工许可前置不满足(F3 门控拒):40900+缺失明细。
	ErrPermitRequired = errors.New("odn: construction start blocked by permit gate")
)

// Permit 许可单(证照档案要素 + 关联 + 状态机)。
type Permit struct {
	ID            int64   `json:"id"`
	PermitNo      string  `json:"permitNo"`
	Kind          string  `json:"kind"`
	Title         string  `json:"title"`
	ApprovalNo    string  `json:"approvalNo"`
	Authority     string  `json:"authority"`
	ValidFrom     string  `json:"validFrom,omitempty"`
	ValidUntil    string  `json:"validUntil,omitempty"`
	Status        string  `json:"status"`
	ProjectID     int64   `json:"projectId"`
	ProjectNo     string  `json:"projectNo"`
	FacilityCode  string  `json:"facilityCode,omitempty"`
	ChainID       int64   `json:"chainId"`
	AttachmentIDs []int64 `json:"attachmentIds"`
	Note          string  `json:"note"`
	RejectReason  string  `json:"rejectReason"`
	CreatedBy     int64   `json:"createdBy"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

// PermitInitialStatus kind 建单初始状态:ROW=未开始,PECE=待签署。
func PermitInitialStatus(kind string) (string, error) {
	switch kind {
	case PermitKindROW:
		return PRowNotStarted, nil
	case PermitKindPECE:
		return PPecePendingSign, nil
	default:
		return "", fmt.Errorf("odn: permit kind %q: %w", kind, ErrInvalidInput)
	}
}

// ValidatePermitTransition 许可单状态转移表(terms.md 4):
// ROW: 未开始-待处理(提交);待处理-已批准(批准)/-未开始(驳回,原因必填);
// 已批准-已过期(手动或门控自动);已过期-待处理(过期复验,重新走批准);
// 未开始与 NA 互转(标记/取消不适用)。PECE: 待签署-已签署-已盖章(门控满足);
// 已签署-待签署(退回补正,原因必填);待签署与 NA 互转(作废/恢复)。
func ValidatePermitTransition(kind, from, to string) error {
	if from == to {
		return nil
	}
	switch kind {
	case PermitKindROW:
		switch {
		case from == PRowNotStarted && to == PRowPending:
			return nil
		case from == PRowPending && to == PRowApproved:
			return nil
		case from == PRowPending && to == PRowNotStarted:
			return nil
		case from == PRowApproved && to == PRowExpired:
			return nil
		case from == PRowExpired && to == PRowPending:
			return nil
		case from == PRowNotStarted && to == PermitNA:
			return nil
		case from == PermitNA && to == PRowNotStarted:
			return nil
		}
	case PermitKindPECE:
		switch {
		case from == PPecePendingSign && to == PPeceSigned:
			return nil
		case from == PPeceSigned && to == PPeceStamped:
			return nil
		case from == PPeceSigned && to == PPecePendingSign:
			return nil
		case from == PPecePendingSign && to == PermitNA:
			return nil
		case from == PermitNA && to == PPecePendingSign:
			return nil
		}
	}
	return ErrPermitState
}

// permitSatisfied 单张许可是否满足对应类型的开工条件。
func permitSatisfied(p Permit, today string) bool {
	switch p.Kind {
	case PermitKindROW:
		if p.Status == PermitNA {
			return true
		}
		return p.Status == PRowApproved && p.ValidUntil != "" && p.ValidUntil >= today
	case PermitKindPECE:
		if p.Status == PermitNA {
			return true
		}
		return p.Status == PPeceStamped
	}
	return false
}

// PermitGateReport 开工许可门控判定结果。
type PermitGateReport struct {
	OK         bool     `json:"ok"`
	Problems   []string `json:"problems"`
	ExpiredIDs []int64  `json:"expiredIds"`
}

// EvaluatePermitGate 纯函数:项目关联许可的开工门控判定(F3)。
// ROW 满足 = 存在 APPROVED 且 valid_until >= today,或 NA(不适用);
// PECE 满足 = 存在 STAMPED,或 NA;APPROVED 但已过期不计满足,归入需自动过期回写。
func EvaluatePermitGate(permits []Permit, today string) PermitGateReport {
	rep := PermitGateReport{OK: true, Problems: []string{}, ExpiredIDs: []int64{}}
	rowOK, peceOK := false, false
	for _, p := range permits {
		if p.Kind == PermitKindROW && p.Status == PRowApproved &&
			p.ValidUntil != "" && p.ValidUntil < today {
			rep.ExpiredIDs = append(rep.ExpiredIDs, p.ID)
			continue
		}
		switch p.Kind {
		case PermitKindROW:
			rowOK = rowOK || permitSatisfied(p, today)
		case PermitKindPECE:
			peceOK = peceOK || permitSatisfied(p, today)
		}
	}
	if !rowOK {
		rep.OK = false
		rep.Problems = append(rep.Problems, "ROW: 无已批准且在有效期的路权许可(或未标记不适用)")
	}
	if !peceOK {
		rep.OK = false
		rep.Problems = append(rep.Problems, "PECE: 无已盖章许可(或未标记不适用)")
	}
	return rep
}

// PermitStore 许可单存储口(PGStore 实现)。
type PermitStore interface {
	CreatePermit(ctx context.Context, p Permit, createdBy int64) (*Permit, error)
	GetPermit(ctx context.Context, id int64) (*Permit, error)
	ListPermits(ctx context.Context, kind, status string, projectID int64, unlinked bool, limit int) ([]Permit, error)
	UpdatePermitArchive(ctx context.Context, id int64, p Permit) error
	TransitionPermit(ctx context.Context, id, accountID int64, to, reason, approvalNo, validFrom, validUntil string) (*Permit, error)
	LinkPermitProject(ctx context.Context, id, projectID int64) error
	UnlinkPermitProject(ctx context.Context, id int64) error
	ListProjectPermits(ctx context.Context, projectID int64) ([]Permit, error)
	CheckProjectPermits(ctx context.Context, projectID int64) (*PermitGateReport, error)
}
