package odn

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
)

// 质量记录存储(P0-C,迁移 000214):测试 append-only + 整改闭环 + 验收前置计数。

func scanTest(row pgx.Row, t *QualityTest) error {
	return row.Scan(&t.ID, &t.ProjectID, &t.ResourceType, &t.ResourceRef, &t.TestKind, &t.Result,
		&t.AttenuationDB, &t.PowerDBM, &t.Note, &t.ReportedBy, &t.ReportedAt)
}

// RecordTest 追加测试记录(append-only)。
func (s *PGStore) RecordTest(ctx context.Context, t QualityTest) (int64, error) {
	if err := ValidateQualityTest(t); err != nil {
		return 0, err
	}
	inScope := t.ResourceType != "FACILITY"
	if !inScope {
		err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM construction_items WHERE project_id=$1 AND facility_code=$2)", t.ProjectID, t.ResourceRef).Scan(&inScope)
		if err != nil {
			log.Printf("[odn-quality] TEST SCOPE READ FAILED proj=%d ref=%s: %v", t.ProjectID, t.ResourceRef, err)
			return 0, fmt.Errorf("odn: test scope read: %w", err)
		}
	}
	if !inScope {
		return 0, fmt.Errorf("%w: ref=%s", ErrFacilityNotInScope, t.ResourceRef)
	}
	var id int64
	err := s.db.QueryRow(ctx, "INSERT INTO construction_tests "+
		"(project_id, resource_type, resource_ref, test_kind, result, attenuation_db, power_dbm, note, reported_by) "+
		"VALUES ($1,$2,$3,$4,$5,NULLIF($6,0),NULLIF($7,0),$8,NULLIF($9,0)) RETURNING id",
		t.ProjectID, t.ResourceType, t.ResourceRef, t.TestKind, t.Result, t.AttenuationDB, t.PowerDBM, t.Note, t.ReportedBy).Scan(&id)
	if err != nil {
		log.Printf("[odn-quality] TEST INSERT FAILED proj=%d ref=%s kind=%s: %v", t.ProjectID, t.ResourceRef, t.TestKind, err)
		return 0, fmt.Errorf("odn: test insert: %w", err)
	}
	return id, nil
}

// ListTests 项目测试记录(近单优先)。
func (s *PGStore) ListTests(ctx context.Context, projectID int64, limit int) ([]QualityTest, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(ctx, "SELECT id, project_id, resource_type, resource_ref, test_kind, result, "+
		"COALESCE(attenuation_db,0), COALESCE(power_dbm,0), COALESCE(note,''), COALESCE(reported_by,0), to_char(reported_at,'YYYY-MM-DD HH24:MI:SS') "+
		"FROM construction_tests WHERE project_id=$1 ORDER BY id DESC LIMIT $2", projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("odn: list tests: %w", err)
	}
	defer rows.Close()
	out := []QualityTest{}
	for rows.Next() {
		var t QualityTest
		if err := scanTest(rows, &t); err != nil {
			return nil, fmt.Errorf("odn: scan test: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// OpenDefect 开整改单:设施须在项目范围内。
func (s *PGStore) OpenDefect(ctx context.Context, d QualityDefect, openedBy int64) (int64, error) {
	if err := ValidateDefectOpen(d); err != nil {
		return 0, err
	}
	var inScope bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM construction_items WHERE project_id=$1 AND facility_code=$2)", d.ProjectID, d.FacilityCode).Scan(&inScope)
	if err != nil {
		return 0, fmt.Errorf("odn: defect scope read: %w", err)
	}
	if !inScope {
		return 0, fmt.Errorf("%w: fac=%s", ErrFacilityNotInScope, d.FacilityCode)
	}
	var id int64
	err = s.db.QueryRow(ctx, "INSERT INTO construction_defects "+
		"(project_id, facility_code, severity, description, note, opened_by) VALUES ($1,$2,$3,$4,$5,NULLIF($6,0)) RETURNING id",
		d.ProjectID, d.FacilityCode, d.Severity, d.Description, d.Note, openedBy).Scan(&id)
	if err != nil {
		log.Printf("[odn-quality] DEFECT OPEN FAILED proj=%d fac=%s: %v", d.ProjectID, d.FacilityCode, err)
		return 0, fmt.Errorf("odn: defect open: %w", err)
	}
	return id, nil
}

// ListDefects 项目整改单列表(可按状态过滤)。
func (s *PGStore) ListDefects(ctx context.Context, projectID int64, status string, limit int) ([]QualityDefect, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	where, args := []string{"project_id=$1"}, []any{projectID}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}
	args = append(args, limit)
	rows, err := s.db.Query(ctx, "SELECT id, project_id, facility_code, severity, description, status, COALESCE(note,''), "+
		"COALESCE(opened_by,0), to_char(opened_at,'YYYY-MM-DD HH24:MI:SS'), "+
		"COALESCE(to_char(rectified_at,'YYYY-MM-DD HH24:MI:SS'),''), COALESCE(verified_by,0), "+
		"COALESCE(to_char(verified_at,'YYYY-MM-DD HH24:MI:SS'),'') "+
		"FROM construction_defects WHERE "+strings.Join(where, " AND ")+" ORDER BY id DESC LIMIT "+fmt.Sprintf("$%d", len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("odn: list defects: %w", err)
	}
	defer rows.Close()
	out := []QualityDefect{}
	for rows.Next() {
		var d QualityDefect
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.FacilityCode, &d.Severity, &d.Description, &d.Status, &d.Note, &d.OpenedBy, &d.OpenedAt, &d.RectifiedAt, &d.VerifiedBy, &d.VerifiedAt); err != nil {
			return nil, fmt.Errorf("odn: scan defect: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// TransitionDefect 整改状态转移:RECTIFYING 记时间,VERIFIED 记复验人/时间。
func (s *PGStore) TransitionDefect(ctx context.Context, defectID int64, to string, accountID int64) error {
	var from string
	err := s.db.QueryRow(ctx, "SELECT status FROM construction_defects WHERE id=$1", defectID).Scan(&from)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("odn: defect read: %w", err)
	}
	if err := ValidateDefectTransition(from, to); err != nil {
		return err
	}
	var tag interface{ RowsAffected() int64 }
	if to == DefectRectifying {
		tag, err = s.db.Exec(ctx, "UPDATE construction_defects SET status=$2, rectified_at=now() WHERE id=$1 AND status<>'VERIFIED'", defectID, to)
	} else {
		tag, err = s.db.Exec(ctx, "UPDATE construction_defects SET status=$2, verified_by=NULLIF($3,0), verified_at=now() WHERE id=$1 AND status<>'VERIFIED'", defectID, to, accountID)
	}
	if err != nil {
		log.Printf("[odn-quality] DEFECT TRANSITION FAILED id=%d to=%s: %v", defectID, to, err)
		return fmt.Errorf("odn: defect transition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDefectState
	}
	return nil
}

// CountUnverifiedDefects 验收前置:未 VERIFIED 整改项计数。
func (s *PGStore) CountUnverifiedDefects(ctx context.Context, projectID int64) (int64, error) {
	var n int64
	err := s.db.QueryRow(ctx, "SELECT count(*) FROM construction_defects WHERE project_id=$1 AND status<>'VERIFIED'", projectID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("odn: defect count: %w", err)
	}
	return n, nil
}
