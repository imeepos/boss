package odn

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// projCols 施工项目查询列(000199)。
const projCols = "p.id, p.proj_no, COALESCE(p.name,''), COALESCE(p.prv_code,''), " +
	"COALESCE(p.city_prefix,''), p.status, COALESCE(p.asbuilt_note,''), COALESCE(p.accepted_by,0), " +
	"COALESCE(to_char(p.accepted_at,'YYYY-MM-DD HH24:MI:SS'),''), " +
	"(SELECT count(*) FROM construction_items i WHERE i.project_id=p.id), " +
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
		&p.Status, &p.AsbuiltNote, &p.AcceptedBy, &p.AcceptedAt, &p.ItemCount, &p.UpdatedAt)
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
			&p.Status, &p.AsbuiltNote, &p.AcceptedBy, &p.AcceptedAt, &p.ItemCount, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("odn: scan project: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// AddProjectItem 追加施工明细:PENDING/BUILDING 可加,ACCEPTED 锁定;设施须 PLANNED。
func (s *PGStore) AddProjectItem(ctx context.Context, projectID int64, facilityCode string) error {
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
	_, err = s.db.Exec(ctx, `INSERT INTO construction_items (project_id, facility_code) VALUES ($1,$2)`,
		projectID, facilityCode)
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

// StartProject 开工 PENDING→BUILDING,单内 PLANNED 设施批量推 IN_BUILD。返回翻转数。
func (s *PGStore) StartProject(ctx context.Context, id int64) (int64, error) {
	if err := s.casProjectStatus(ctx, id, CPending, CBuilding); err != nil {
		return 0, err
	}
	tag, err := s.db.Exec(ctx, `UPDATE odn_facility f SET lifecycle_status='IN_BUILD'
		FROM construction_items i WHERE i.facility_code=f.code AND i.project_id=$1
		AND f.lifecycle_status='PLANNED'`, id)
	if err != nil {
		log.Printf("[odn-construction] START FLIP FAILED proj=%d: %v", id, err)
		return 0, fmt.Errorf("odn: start flip: %w", err)
	}
	return tag.RowsAffected(), nil
}

// AcceptProject 竣工验收 BUILDING→ACCEPTED,单内设施批量推 IN_SERVICE(as-built 回填)。返回翻转数。
func (s *PGStore) AcceptProject(ctx context.Context, id int64, acceptedBy int64, note string) (int64, error) {
	tag, err := s.db.Exec(ctx, `UPDATE construction_projects SET status='ACCEPTED',
		asbuilt_note=$2, accepted_by=$3, accepted_at=now(), updated_at=now()
		WHERE id=$1 AND status='BUILDING'`, id, note, acceptedBy)
	if err != nil {
		return 0, fmt.Errorf("odn: accept project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.projectStatus(ctx, id); err != nil {
			return 0, err
		}
		return 0, ErrInvalidProjStatus
	}
	tag, err = s.db.Exec(ctx, `UPDATE odn_facility f SET lifecycle_status='IN_SERVICE'
		FROM construction_items i WHERE i.facility_code=f.code AND i.project_id=$1
		AND f.lifecycle_status IN ('PLANNED','IN_BUILD')`, id)
	if err != nil {
		log.Printf("[odn-construction] ACCEPT FLIP FAILED proj=%d: %v", id, err)
		return 0, fmt.Errorf("odn: accept flip: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ListProjectItems 单内明细清单。
func (s *PGStore) ListProjectItems(ctx context.Context, id int64) ([]ConstructionItem, error) {
	rows, err := s.db.Query(ctx, `SELECT id, project_id, facility_code
		FROM construction_items WHERE project_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("odn: list items: %w", err)
	}
	defer rows.Close()
	out := []ConstructionItem{}
	for rows.Next() {
		var it ConstructionItem
		if err := rows.Scan(&it.ID, &it.ProjectID, &it.FacilityCode); err != nil {
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
