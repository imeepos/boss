package odn

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// milestoneCols 里程碑查询列(000218)。
const milestoneCols = "id, project_id, name, COALESCE(to_char(planned_date,'YYYY-MM-DD'),''), " +
	"status, COALESCE(to_char(done_at,'YYYY-MM-DD HH24:MI:SS'),''), " +
	"to_char(created_at,'YYYY-MM-DD HH24:MI:SS')"

func scanMilestone(row pgx.Row, m *Milestone) error {
	return row.Scan(&m.ID, &m.ProjectID, &m.Name, &m.PlannedDate, &m.Status, &m.DoneAt, &m.CreatedAt)
}

// SetProjectBudget 设置/清除项目预算(NULL=清除);仅项目 PENDING(BUILDING 前)可改。
func (s *PGStore) SetProjectBudget(ctx context.Context, projectID int64, amount *float64) error {
	if amount != nil && *amount < 0 {
		return fmt.Errorf("odn: budget %v: %w", *amount, ErrInvalidInput)
	}
	tag, err := s.db.Exec(ctx, `UPDATE construction_projects SET budget_amount=$2, updated_at=now()
		WHERE id=$1 AND status='PENDING'`, projectID, amount)
	if err != nil {
		log.Printf("[odn-budget] SET FAILED proj=%d: %v", projectID, err)
		return fmt.Errorf("odn: set budget: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.projectStatus(ctx, projectID); err != nil {
			return err
		}
		return fmt.Errorf("odn: project %d status 非 PENDING: %w", projectID, ErrBudgetLocked)
	}
	return nil
}

// ListMilestones 项目里程碑清单(序同创建序)。
func (s *PGStore) ListMilestones(ctx context.Context, projectID int64) ([]Milestone, error) {
	rows, err := s.db.Query(ctx, `SELECT `+milestoneCols+
		` FROM construction_milestones WHERE project_id=$1 ORDER BY id`, projectID)
	if err != nil {
		log.Printf("[odn-budget] MILESTONE LIST FAILED proj=%d: %v", projectID, err)
		return nil, fmt.Errorf("odn: list milestones: %w", err)
	}
	defer rows.Close()
	out := []Milestone{}
	for rows.Next() {
		var m Milestone
		if err := scanMilestone(rows, &m); err != nil {
			return nil, fmt.Errorf("odn: scan milestone: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// AddMilestone 追加里程碑(名称必填,计划日可空 YYYY-MM-DD);仅项目 PENDING 可改。
func (s *PGStore) AddMilestone(ctx context.Context, projectID int64, name, plannedDate string) (*Milestone, error) {
	if name == "" {
		return nil, fmt.Errorf("odn: milestone name required: %w", ErrInvalidInput)
	}
	st, err := s.projectStatus(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := ValidateMilestoneEdit(st); err != nil {
		return nil, err
	}
	var planned any
	if plannedDate != "" {
		planned = plannedDate
	}
	var m Milestone
	err = s.db.QueryRow(ctx, `INSERT INTO construction_milestones (project_id, name, planned_date)
		VALUES ($1,$2,$3) RETURNING `+milestoneCols, projectID, name, planned).
		Scan(&m.ID, &m.ProjectID, &m.Name, &m.PlannedDate, &m.Status, &m.DoneAt, &m.CreatedAt)
	if err != nil {
		log.Printf("[odn-budget] MILESTONE ADD FAILED proj=%d name=%s: %v", projectID, name, err)
		return nil, fmt.Errorf("odn: add milestone: %w", err)
	}
	return &m, nil
}

// UpdateMilestone 编辑里程碑(名称/计划日);项目须 PENDING(经 join 校验,CAS 兜底并发)。
func (s *PGStore) UpdateMilestone(ctx context.Context, milestoneID int64, name, plannedDate string) error {
	if name == "" {
		return fmt.Errorf("odn: milestone name required: %w", ErrInvalidInput)
	}
	var planned any
	if plannedDate != "" {
		planned = plannedDate
	}
	tag, err := s.db.Exec(ctx, `UPDATE construction_milestones m SET name=$2, planned_date=$3, updated_at=now()
		FROM construction_projects p
		WHERE m.id=$1 AND m.project_id=p.id AND p.status='PENDING'`, milestoneID, name, planned)
	if err != nil {
		log.Printf("[odn-budget] MILESTONE UPDATE FAILED id=%d: %v", milestoneID, err)
		return fmt.Errorf("odn: update milestone: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var pid int64
		err := s.db.QueryRow(ctx, `SELECT project_id FROM construction_milestones WHERE id=$1`, milestoneID).Scan(&pid)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("odn: milestone read: %w", err)
		}
		return fmt.Errorf("odn: milestone %d project 非 PENDING: %w", milestoneID, ErrBudgetLocked)
	}
	return nil
}

// MarkMilestone 里程碑状态标记(PENDING/DONE);项目 ACCEPTED 后锁定,PENDING/BUILDING 可标记。
func (s *PGStore) MarkMilestone(ctx context.Context, milestoneID int64, status string) error {
	if err := ValidateMilestoneStatus(status); err != nil {
		return err
	}
	var projStatus string
	err := s.db.QueryRow(ctx, `SELECT p.status FROM construction_milestones m
		JOIN construction_projects p ON p.id=m.project_id WHERE m.id=$1`, milestoneID).Scan(&projStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("odn: milestone project read: %w", err)
	}
	if err := ValidateMilestoneMark(projStatus); err != nil {
		return fmt.Errorf("odn: milestone %d project %s: %w", milestoneID, projStatus, err)
	}
	tag, err := s.db.Exec(ctx, `UPDATE construction_milestones SET status=$2,
		done_at=CASE WHEN $2='DONE' THEN now() ELSE NULL END, updated_at=now() WHERE id=$1`,
		milestoneID, status)
	if err != nil {
		log.Printf("[odn-budget] MILESTONE MARK FAILED id=%d status=%s: %v", milestoneID, status, err)
		return fmt.Errorf("odn: mark milestone: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
