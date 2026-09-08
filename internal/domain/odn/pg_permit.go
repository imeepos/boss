package odn

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// permitCols 许可单查询列(000211);可空列经 COALESCE 归一扫描。
const permitCols = "p.id, p.permit_no, p.kind, p.title, p.approval_no, p.authority, " +
	"COALESCE(to_char(p.valid_from,'YYYY-MM-DD'),''), COALESCE(to_char(p.valid_until,'YYYY-MM-DD'),''), p.status, COALESCE(p.project_id,0), p.project_no, " +
	"COALESCE(p.facility_code,''), COALESCE(p.chain_id,0), p.attachment_ids, p.note, p.reject_reason, " +
	"COALESCE(p.created_by,0), to_char(p.created_at,'YYYY-MM-DD HH24:MI:SS'), " +
	"to_char(p.updated_at,'YYYY-MM-DD HH24:MI:SS')"

// scanPermit 扫描许可单行。
func scanPermit(row pgx.Row, p *Permit) error {
	return row.Scan(&p.ID, &p.PermitNo, &p.Kind, &p.Title, &p.ApprovalNo,
		&p.Authority, &p.ValidFrom, &p.ValidUntil, &p.Status, &p.ProjectID,
		&p.ProjectNo, &p.FacilityCode, &p.ChainID, &p.AttachmentIDs, &p.Note,
		&p.RejectReason, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
}

// nextPermitNo 生成许可单号 PM-YYYYMMDD-NNNNN(口径同结算单号,纳秒后 5 位兜底)。
func nextPermitNo() string {
	return fmt.Sprintf("PM-%s-%05d", time.Now().Format("20060102"), time.Now().UnixNano()%100000)
}

// dateArg 日期入参:空串传 NULL(可空 date 列),SQL 不内嵌字面量。
func dateArg(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// CreatePermit 新建许可单:kind 定初始状态(ROW=未开始,PECE=待签署);
// 关联项目时快照单号。证照档案要素可在详情页补录(补件语义)。
func (s *PGStore) CreatePermit(ctx context.Context, p Permit, createdBy int64) (*Permit, error) {
	st, err := PermitInitialStatus(p.Kind)
	if err != nil {
		return nil, err
	}
	ids := p.AttachmentIDs
	if ids == nil {
		ids = []int64{}
	}
	var created string
	qq := "INSERT INTO odn_permits (permit_no, kind, title, approval_no, authority, valid_from, valid_until, " +
		"status, project_id, project_no, facility_code, chain_id, attachment_ids, note, created_by) " +
		"VALUES ($1,$2,$3,$4,$5,$6::date,$7::date,$8," +
		"NULLIF($9,0),COALESCE((SELECT proj_no FROM construction_projects WHERE id=NULLIF($9,0)),'')," +
		"NULLIF($10,''),NULLIF($11,0),$12,$13,NULLIF($14,0)) " +
		"RETURNING id, permit_no, to_char(created_at,'YYYY-MM-DD HH24:MI:SS')"
	err = s.db.QueryRow(ctx, qq, nextPermitNo(), p.Kind, p.Title, p.ApprovalNo, p.Authority,
		dateArg(p.ValidFrom), dateArg(p.ValidUntil), st, p.ProjectID, p.FacilityCode, p.ChainID,
		ids, p.Note, createdBy).Scan(&p.ID, &p.PermitNo, &created)
	if err != nil {
		log.Printf("[odn-permit] CREATE FAILED kind=%s proj=%d: %v", p.Kind, p.ProjectID, err)
		return nil, fmt.Errorf("odn: create permit: %w", err)
	}
	p.Status, p.CreatedAt, p.UpdatedAt = st, created, created
	return &p, nil
}

// GetPermit 许可单详情(ErrNoRows 归 nil)。
func (s *PGStore) GetPermit(ctx context.Context, id int64) (*Permit, error) {
	var p Permit
	err := scanPermit(s.db.QueryRow(ctx, "SELECT "+permitCols+" FROM odn_permits p WHERE p.id=$1", id), &p)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("odn: get permit: %w", err)
	}
	return &p, nil
}

// ListPermits 许可单列表(类型/状态/项目/未关联过滤,近单优先)。
func (s *PGStore) ListPermits(ctx context.Context, kind, status string, projectID int64, unlinked bool, limit int) ([]Permit, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	where, args := []string{"1=1"}, []any{limit}
	if kind != "" {
		args = append(args, kind)
		where = append(where, "p.kind=$2")
	}
	if status != "" {
		args = append(args, status)
		where = append(where, "p.status=$3")
	}
	if projectID > 0 {
		args = append(args, projectID)
		where = append(where, "p.project_id=$4")
	}
	if unlinked {
		where = append(where, "p.project_id IS NULL")
	}
	qq := "SELECT " + permitCols + " FROM odn_permits p WHERE " + strings.Join(where, " AND ") + " ORDER BY p.id DESC LIMIT $1"
	rows, err := s.db.Query(ctx, qq, args...)
	if err != nil {
		log.Printf("[odn-permit] LIST FAILED kind=%s status=%s: %v", kind, status, err)
		return nil, fmt.Errorf("odn: list permits: %w", err)
	}
	defer rows.Close()
	out := []Permit{}
	for rows.Next() {
		var p Permit
		if err := scanPermit(rows, &p); err != nil {
			return nil, fmt.Errorf("odn: scan permit: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListProjectPermits 项目关联许可(门控与详情页数据源)。
func (s *PGStore) ListProjectPermits(ctx context.Context, projectID int64) ([]Permit, error) {
	if projectID <= 0 {
		return []Permit{}, nil
	}
	rows, err := s.db.Query(ctx, "SELECT "+permitCols+" FROM odn_permits p WHERE p.project_id=$1 ORDER BY p.id", projectID)
	if err != nil {
		return nil, fmt.Errorf("odn: list project permits: %w", err)
	}
	defer rows.Close()
	out := []Permit{}
	for rows.Next() {
		var p Permit
		if err := scanPermit(rows, &p); err != nil {
			return nil, fmt.Errorf("odn: scan project permit: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdatePermitArchive 证照档案要素补录(批复号/机构/有效期/附件/备注/关联设施与资源链);
// 状态与项目关联不经此口(补件语义:任何状态可补录)。
func (s *PGStore) UpdatePermitArchive(ctx context.Context, id int64, p Permit) error {
	ids := p.AttachmentIDs
	if ids == nil {
		ids = []int64{}
	}
	tag, err := s.db.Exec(ctx, "UPDATE odn_permits SET title=$2, approval_no=$3, authority=$4, "+
		"valid_from=$5::date, valid_until=$6::date, facility_code=NULLIF($7,"+""+"), chain_id=NULLIF($8,0), "+
		"attachment_ids=$9, note=$10, updated_at=now() WHERE id=$1",
		id, p.Title, p.ApprovalNo, p.Authority, dateArg(p.ValidFrom), dateArg(p.ValidUntil),
		p.FacilityCode, p.ChainID, ids, p.Note)
	if err != nil {
		log.Printf("[odn-permit] UPDATE ARCHIVE FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: update permit archive: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// permitExtra 流转附带字段(批准回填批复号/有效期;驳回/退回/作废落原因)。
type permitExtra struct {
	reason     string
	approvalNo string
	validFrom  string
	validUntil string
}

// validatePermitTransitionInput 流转入参业务校验(原因/批复要素必填规则)。
func validatePermitTransitionInput(kind, from, to string, ex permitExtra) error {
	if err := ValidatePermitTransition(kind, from, to); err != nil {
		return err
	}
	needReason := (kind == PermitKindROW && from == PRowPending && to == PRowNotStarted) ||
		(kind == PermitKindPECE && from == PPeceSigned && to == PPecePendingSign) ||
		(kind == PermitKindPECE && from == PPecePendingSign && to == PermitNA)
	if needReason && ex.reason == "" {
		return fmt.Errorf("odn: permit transition %s to %s reason required: %w", from, to, ErrInvalidInput)
	}
	if kind == PermitKindROW && to == PRowApproved && (ex.approvalNo == "" || ex.validUntil == "") {
		return fmt.Errorf("odn: permit approve requires approvalNo and validUntil: %w", ErrInvalidInput)
	}
	return nil
}

// TransitionPermit 许可单状态流转(CAS+附带回填;全部经 terms.md 4 转移表校验)。
func (s *PGStore) TransitionPermit(ctx context.Context, id, accountID int64, to, reason, approvalNo, validFrom, validUntil string) (*Permit, error) {
	cur, err := s.GetPermit(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, ErrNotFound
	}
	ex := permitExtra{reason: reason, approvalNo: approvalNo, validFrom: validFrom, validUntil: validUntil}
	if err := validatePermitTransitionInput(cur.Kind, cur.Status, to, ex); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-permit] TX BEGIN FAILED id=%d: %v", id, err)
		return nil, fmt.Errorf("odn: permit transition begin: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, "UPDATE odn_permits SET status=$2, updated_at=now() WHERE id=$1 AND status=$3", id, to, cur.Status)
	if err != nil {
		log.Printf("[odn-permit] TRANSITION FAILED id=%d %s to %s: %v", id, cur.Status, to, err)
		return nil, fmt.Errorf("odn: permit transition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("odn: permit %d status=%s: %w", id, cur.Status, ErrPermitState)
	}
	if err := applyPermitSideEffects(ctx, tx, id, cur.Kind, to, ex); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-permit] COMMIT FAILED id=%d to %s: %v", id, to, err)
		return nil, fmt.Errorf("odn: permit transition commit: %w", err)
	}
	return s.GetPermit(ctx, id)
}

// applyPermitSideEffects 流转附带回填:批准写批复号与有效期;驳回/退回/作废落原因。
func applyPermitSideEffects(ctx context.Context, tx pgx.Tx, id int64, kind, to string, ex permitExtra) error {
	if kind == PermitKindROW && to == PRowApproved {
		vf := ex.validFrom
		if vf == "" {
			vf = time.Now().Format("2006-01-02")
		}
		if _, err := tx.Exec(ctx, "UPDATE odn_permits SET approval_no=$2, valid_from=$3::date, valid_until=$4::date, updated_at=now() WHERE id=$1",
			id, ex.approvalNo, vf, ex.validUntil); err != nil {
			log.Printf("[odn-permit] APPROVE BACKFILL FAILED id=%d: %v", id, err)
			return fmt.Errorf("odn: permit approve backfill: %w", err)
		}
		return nil
	}
	if ex.reason != "" {
		if _, err := tx.Exec(ctx, "UPDATE odn_permits SET reject_reason=$2, updated_at=now() WHERE id=$1", id, ex.reason); err != nil {
			log.Printf("[odn-permit] REASON BACKFILL FAILED id=%d: %v", id, err)
			return fmt.Errorf("odn: permit reason backfill: %w", err)
		}
	}
	return nil
}

// LinkPermitProject 关联施工项目(单号快照);UnlinkPermitProject 解除关联。
func (s *PGStore) LinkPermitProject(ctx context.Context, id, projectID int64) error {
	if projectID <= 0 {
		return fmt.Errorf("odn: permit link project %d: %w", projectID, ErrInvalidInput)
	}
	tag, err := s.db.Exec(ctx, "UPDATE odn_permits SET project_id=$2, "+
		"project_no=(SELECT proj_no FROM construction_projects WHERE id=$2), updated_at=now() WHERE id=$1", id, projectID)
	if err != nil {
		log.Printf("[odn-permit] LINK FAILED id=%d proj=%d: %v", id, projectID, err)
		return fmt.Errorf("odn: permit link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PGStore) UnlinkPermitProject(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, "UPDATE odn_permits SET project_id=NULL, project_no='', updated_at=now() WHERE id=$1", id)
	if err != nil {
		log.Printf("[odn-permit] UNLINK FAILED id=%d: %v", id, err)
		return fmt.Errorf("odn: permit unlink: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
