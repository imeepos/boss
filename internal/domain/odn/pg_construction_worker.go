package odn

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// 师傅侧施工进度视图(W7,F5a):BUILDING 项目列表与上报上下文(清单级计划/完成聚合)。

// ListBuildingProjects 师傅可见的施工中项目(近单优先)。
func (s *PGStore) ListBuildingProjects(ctx context.Context, limit int) ([]Construction, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT ` + projCols + ` FROM construction_projects p WHERE p.status=$1 ORDER BY p.id DESC LIMIT $2`
	rows, err := s.db.Query(ctx, q, CBuilding, limit)
	if err != nil {
		log.Printf("[odn-construction] WORKER LIST FAILED: %v", err)
		return nil, fmt.Errorf("odn: building projects: %w", err)
	}
	defer rows.Close()
	out := []Construction{}
	for rows.Next() {
		var p Construction
		if err := rows.Scan(&p.ID, &p.ProjNo, &p.Name, &p.PrvCode, &p.CityPrefix,
			&p.Status, &p.AsbuiltNote, &p.AcceptedBy, &p.AcceptedAt, &p.ItemCount,
			&p.ContractorID, &p.ContractorName, &p.ItemsAmount, &p.BudgetAmount, &p.SettledAmount, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("odn: scan building project: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetProjectWorkerView 师傅上报上下文:项目 + 清单级进度聚合;非 BUILDING 返回状态冲突。
func (s *PGStore) GetProjectWorkerView(ctx context.Context, id int64) (*Construction, []ItemProgress, error) {
	var status string
	err := s.db.QueryRow(ctx, "SELECT status FROM construction_projects WHERE id=$1", id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		log.Printf("[odn-construction] WORKER VIEW READ FAILED id=%d: %v", id, err)
		return nil, nil, fmt.Errorf("odn: worker view read: %w", err)
	}
	if status != CBuilding {
		return nil, nil, fmt.Errorf("odn: project status=%s: %w", status, ErrInvalidProjStatus)
	}
	items, err := s.ItemProgressSummary(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return &Construction{ID: id, Status: status}, items, nil
}
