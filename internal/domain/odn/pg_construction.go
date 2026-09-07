package odn

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// projCols 施工项目查询列(000199;000218 加预算金额与已结算金额只读派生列)。
const projCols = "p.id, p.proj_no, COALESCE(p.name,''), COALESCE(p.prv_code,''), " +
	"COALESCE(p.city_prefix,''), p.status, COALESCE(p.asbuilt_note,''), COALESCE(p.accepted_by,0), " +
	"COALESCE(to_char(p.accepted_at,'YYYY-MM-DD HH24:MI:SS'),''), " +
	"(SELECT count(*) FROM construction_items i WHERE i.project_id=p.id), " +
	"COALESCE(p.contractor_id,0), COALESCE(p.contractor_name,''), " +
	"(SELECT COALESCE(SUM(amount),0) FROM construction_items i WHERE i.project_id=p.id), " +
	"p.budget_amount, " +
	"(SELECT COALESCE(SUM(cs.total_amount),0) FROM construction_settlements cs " +
	"WHERE cs.project_id=p.id AND cs.status='SETTLED'), " +
	"to_char(p.updated_at,'YYYY-MM-DD HH24:MI:SS')"

// CreateProject 新建施工单(proj_no 唯一;冲突→ErrDuplicate)。
func (s *PGStore) CreateProject(ctx context.Context, p Construction) error {
	var prv, city any
	if p.PrvCode != "" {
		prv, city = p.PrvCode, p.CityPrefix
	}
	_, err := s.db.Exec(ctx, `INSERT INTO construction_projects (proj_no, name, prv_code, city_prefix)
		VALUES ($1,$2,$3,$4)`, p.ProjNo, p.Name, prv, city)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicate
		}
		log.Printf("[odn-construction] CREATE FAILED proj=%s: %v", p.ProjNo, err)
		return fmt.Errorf("odn: create project: %w", err)
	}
	return nil
}

// GetProject 施工单详情(不含明细;ErrNoRows→nil)。
func (s *PGStore) GetProject(ctx context.Context, id int64) (*Construction, error) {
	var p Construction
	q := `SELECT ` + projCols + ` FROM construction_projects p WHERE p.id=$1`
	err := s.db.QueryRow(ctx, q, id).Scan(&p.ID, &p.ProjNo, &p.Name, &p.PrvCode, &p.CityPrefix,
		&p.Status, &p.AsbuiltNote, &p.AcceptedBy, &p.AcceptedAt, &p.ItemCount,
		&p.ContractorID, &p.ContractorName, &p.ItemsAmount, &p.BudgetAmount, &p.SettledAmount, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("odn: get project: %w", err)
	}
	return &p, nil
}

// ListProjects 施工单列表(近单优先)。
func (s *PGStore) ListProjects(ctx context.Context, limit int) ([]Construction, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `SELECT ` + projCols + ` FROM construction_projects p ORDER BY p.id DESC LIMIT $1`
	rows, err := s.db.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("odn: list projects: %w", err)
	}
	defer rows.Close()
	out := []Construction{}
	for rows.Next() {
		var p Construction
		if err := rows.Scan(&p.ID, &p.ProjNo, &p.Name, &p.PrvCode, &p.CityPrefix,
			&p.Status, &p.AsbuiltNote, &p.AcceptedBy, &p.AcceptedAt, &p.ItemCount,
			&p.ContractorID, &p.ContractorName, &p.ItemsAmount, &p.BudgetAmount, &p.SettledAmount, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("odn: scan project: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// AddProjectItem 追加施工明细(工程量清单行):PENDING/BUILDING 可加,ACCEPTED 锁定;
// 设施须 PLANNED;数量/单价可带(>=0,缺省 0=未定额),金额由生成列按 数量x单价 计算。
func (s *PGStore) AddProjectItem(ctx context.Context, projectID int64, facilityCode string, qty, unitPrice float64) error {
	if qty < 0 || unitPrice < 0 {
		return fmt.Errorf("odn: item qty=%v unitPrice=%v: %w", qty, unitPrice, ErrInvalidInput)
	}
	st, err := s.projectStatus(ctx, projectID)
	if err != nil {
		return err
	}
	if st == CAccepted {
		return ErrItemLocked
	}
	var lc string
	err = s.db.QueryRow(ctx, `SELECT lifecycle_status FROM odn_facility WHERE code=$1`, facilityCode).Scan(&lc)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("odn: item facility read: %w", err)
	}
	if lc != LCPlanned {
		return fmt.Errorf("%w: 设施 %s 生命周期=%s 非规划态", ErrItemLocked, facilityCode, lc)
	}
	_, err = s.db.Exec(ctx, `INSERT INTO construction_items (project_id, facility_code, quantity, unit_price)
		VALUES ($1,$2,$3,$4)`,
		projectID, facilityCode, qty, unitPrice)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicate
		}
		log.Printf("[odn-construction] ADD ITEM FAILED proj=%d fac=%s: %v", projectID, facilityCode, err)
		return fmt.Errorf("odn: add item: %w", err)
	}
	return nil
}

// ListProjectItems 单内明细清单。
func (s *PGStore) ListProjectItems(ctx context.Context, id int64) ([]ConstructionItem, error) {
	rows, err := s.db.Query(ctx, `SELECT id, project_id, facility_code, quantity, unit_price, amount
		FROM construction_items WHERE project_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("odn: list items: %w", err)
	}
	defer rows.Close()
	out := []ConstructionItem{}
	for rows.Next() {
		var it ConstructionItem
		if err := rows.Scan(&it.ID, &it.ProjectID, &it.FacilityCode, &it.Quantity, &it.UnitPrice, &it.Amount); err != nil {
			return nil, fmt.Errorf("odn: scan item: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// casProjectStatus 施工单状态 CAS 转移;0 行时区分不存在与状态漂移。
func (s *PGStore) casProjectStatus(ctx context.Context, id int64, from, to string) error {
	tag, err := s.db.Exec(ctx, `UPDATE construction_projects SET status=$2, updated_at=now()
		WHERE id=$1 AND status=$3`, id, to, from)
	if err != nil {
		return fmt.Errorf("odn: project cas: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.projectStatus(ctx, id); err != nil {
			return err
		}
		return ErrInvalidProjStatus
	}
	return nil
}

// UpdateProjectItem 清单定额编辑:仅 ACCEPTED 前可改(明细锁定同既有口径);金额随生成列重算。
func (s *PGStore) UpdateProjectItem(ctx context.Context, projectID, itemID int64, qty, unitPrice float64) error {
	if qty < 0 || unitPrice < 0 {
		return fmt.Errorf("odn: item qty=%v unitPrice=%v: %w", qty, unitPrice, ErrInvalidInput)
	}
	st, err := s.projectStatus(ctx, projectID)
	if err != nil {
		return err
	}
	if st == CAccepted {
		return ErrItemLocked
	}
	tag, err := s.db.Exec(ctx, `UPDATE construction_items SET quantity=$3, unit_price=$4
		WHERE id=$2 AND project_id=$1`, projectID, itemID, qty, unitPrice)
	if err != nil {
		log.Printf("[odn-construction] UPDATE ITEM FAILED proj=%d item=%d: %v", projectID, itemID, err)
		return fmt.Errorf("odn: update item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetProjectContractor 指定/更换承包商(名称快照,软引用采购域供应商档案);
// 存在有效结算单后锁定(部分唯一口径),作废单解锁可重指定。
func (s *PGStore) SetProjectContractor(ctx context.Context, projectID, contractorID int64, contractorName string) error {
	if contractorID <= 0 {
		return fmt.Errorf("odn: contractor %d: %w", contractorID, ErrInvalidInput)
	}
	if _, err := s.projectStatus(ctx, projectID); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `UPDATE construction_projects p
		SET contractor_id=$2, contractor_name=$3, updated_at=now()
		WHERE p.id=$1 AND NOT EXISTS (SELECT 1 FROM construction_settlements s
			WHERE s.project_id=p.id AND s.status<>'VOIDED')`, projectID, contractorID, contractorName)
	if err != nil {
		log.Printf("[odn-construction] ASSIGN CONTRACTOR FAILED proj=%d supplier=%d: %v", projectID, contractorID, err)
		return fmt.Errorf("odn: assign contractor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("odn: project %d 已有有效结算单,承包商锁定: %w", projectID, ErrSettlementState)
	}
	return nil
}

// projectStatus 读施工单状态(不存在→ErrNotFound)。
func (s *PGStore) projectStatus(ctx context.Context, id int64) (string, error) {
	var st string
	err := s.db.QueryRow(ctx, `SELECT status FROM construction_projects WHERE id=$1`, id).Scan(&st)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("odn: project status: %w", err)
	}
	return st, nil
}
