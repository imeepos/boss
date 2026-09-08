package odn

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// 施工进度存储(P0-B,迁移 000213):幂等上报/列表/清单级聚合。

const progressCols = "p.id, p.project_id, p.facility_code, p.done_qty, COALESCE(p.lat,0), COALESCE(p.lng,0), COALESCE(p.note,''), p.photo_ids, p.client_msg_id, p.reporter_type, COALESCE(p.reported_by,0), " +
	"COALESCE(CASE WHEN p.reporter_type='WORKER' THEN w.name ELSE a.real_name END,''), to_char(p.reported_at,'YYYY-MM-DD HH24:MI:SS')"

const progressFrom = " FROM construction_progress p " +
	"LEFT JOIN workers w ON p.reporter_type='WORKER' AND w.id=p.reported_by " +
	"LEFT JOIN accounts a ON p.reporter_type<>'WORKER' AND a.id=p.reported_by"

func scanProgress(row pgx.Row, p *ProgressEntry) error {
	return row.Scan(&p.ID, &p.ProjectID, &p.FacilityCode, &p.DoneQty, &p.Lat, &p.Lng, &p.Note, &p.PhotoIDs, &p.ClientMsgID, &p.ReporterType, &p.ReportedBy, &p.ReporterName, &p.ReportedAt)
}

// RecordProgress 幂等记录进度:唯一键冲突时返回既有 id 且 created=false(不重复计量)。
// 设施必须已在项目明细范围内,否则 ErrFacilityNotInScope。
func (s *PGStore) RecordProgress(ctx context.Context, p ProgressEntry) (int64, bool, error) {
	if err := ValidateProgressEntry(p); err != nil {
		return 0, false, err
	}
	rtype := p.ReporterType
	if rtype == "" {
		rtype = ProgressReporterAccount
	}
	var inScope bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM construction_items WHERE project_id=$1 AND facility_code=$2)", p.ProjectID, p.FacilityCode).Scan(&inScope)
	if err != nil {
		log.Printf("[odn-construction] PROGRESS SCOPE READ FAILED proj=%d fac=%s: %v", p.ProjectID, p.FacilityCode, err)
		return 0, false, fmt.Errorf("odn: progress scope read: %w", err)
	}
	if !inScope {
		return 0, false, fmt.Errorf("%w: fac=%s", ErrFacilityNotInScope, p.FacilityCode)
	}
	if rtype == ProgressReporterWorker {
		var status string
		err := s.db.QueryRow(ctx, "SELECT status FROM construction_projects WHERE id=$1", p.ProjectID).Scan(&status)
		if err != nil {
			log.Printf("[odn-construction] PROGRESS STATUS READ FAILED proj=%d: %v", p.ProjectID, err)
			return 0, false, fmt.Errorf("odn: progress status read: %w", err)
		}
		if status != CBuilding {
			return 0, false, fmt.Errorf("odn: project status=%s: %w", status, ErrInvalidProjStatus)
		}
	}
	var id int64
	err = s.db.QueryRow(ctx, "INSERT INTO construction_progress "+
		"(project_id, facility_code, done_qty, lat, lng, note, photo_ids, client_msg_id, reporter_type, reported_by) "+
		"VALUES ($1,$2,$3,NULLIF($4::float8,0),NULLIF($5::float8,0),$6,$7,$8,$9,NULLIF($10,0)) "+
		"ON CONFLICT (project_id, facility_code, client_msg_id) DO NOTHING RETURNING id",
		p.ProjectID, p.FacilityCode, p.DoneQty, p.Lat, p.Lng, p.Note, p.PhotoIDs, p.ClientMsgID, rtype, p.ReportedBy).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("[odn-construction] PROGRESS INSERT FAILED proj=%d fac=%s msg=%s: %v", p.ProjectID, p.FacilityCode, p.ClientMsgID, err)
		return 0, false, fmt.Errorf("odn: progress insert: %w", err)
	}
	err = s.db.QueryRow(ctx, "SELECT id FROM construction_progress WHERE project_id=$1 AND facility_code=$2 AND client_msg_id=$3",
		p.ProjectID, p.FacilityCode, p.ClientMsgID).Scan(&id)
	if err != nil {
		log.Printf("[odn-construction] PROGRESS IDEMPOTENT READ FAILED proj=%d fac=%s msg=%s: %v", p.ProjectID, p.FacilityCode, p.ClientMsgID, err)
		return 0, false, fmt.Errorf("odn: progress idempotent read: %w", err)
	}
	return id, false, nil
}

// ListProgress 项目进度上报列表(近单优先)。
func (s *PGStore) ListProgress(ctx context.Context, projectID int64, limit int) ([]ProgressEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(ctx, "SELECT "+progressCols+progressFrom+" WHERE p.project_id=$1 ORDER BY p.id DESC LIMIT $2", projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("odn: list progress: %w", err)
	}
	defer rows.Close()
	out := []ProgressEntry{}
	for rows.Next() {
		var p ProgressEntry
		if err := scanProgress(rows, &p); err != nil {
			return nil, fmt.Errorf("odn: scan progress: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ItemProgressSummary 清单项进度聚合:计划量=清单 quantity,完成量=进度 done_qty 汇总。
func (s *PGStore) ItemProgressSummary(ctx context.Context, projectID int64) ([]ItemProgress, error) {
	rows, err := s.db.Query(ctx, "SELECT i.facility_code, i.quantity, COALESCE(SUM(p.done_qty),0), COUNT(p.id) "+
		"FROM construction_items i LEFT JOIN construction_progress p ON p.project_id=i.project_id AND p.facility_code=i.facility_code "+
		"WHERE i.project_id=$1 GROUP BY i.facility_code, i.quantity ORDER BY i.facility_code", projectID)
	if err != nil {
		return nil, fmt.Errorf("odn: progress summary: %w", err)
	}
	defer rows.Close()
	out := []ItemProgress{}
	for rows.Next() {
		var it ItemProgress
		if err := rows.Scan(&it.FacilityCode, &it.PlannedQty, &it.DoneQty, &it.Entries); err != nil {
			return nil, fmt.Errorf("odn: scan progress summary: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
