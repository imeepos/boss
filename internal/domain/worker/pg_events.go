package worker

import (
	"context"
	"fmt"
)

// ListMaterials 列出师傅物料领用;workerID=0 返回全部。
func (s *PGStore) ListMaterials(ctx context.Context, workerID int64) ([]Material, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, `+factSnap+`, name, qty FROM worker_materials WHERE ($1 = 0 OR worker_id = $1) ORDER BY id`, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker: list materials: %w", err)
	}
	defer rows.Close()
	out := make([]Material, 0)
	for rows.Next() {
		var m Material
		if err := rows.Scan(&m.ID, &m.WorkerID, &m.GroupID, &m.GroupName, &m.LegalEntityID, &m.LegalEntityName, &m.RegionID, &m.RegionName, &m.Name, &m.Qty); err != nil {
			return nil, fmt.Errorf("worker: scan material: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// AppendMaterial 追加物料领用,返回自增 id。
func (s *PGStore) AppendMaterial(ctx context.Context, m Material) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO worker_materials(`+factSnap+`, name, qty) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		m.WorkerID, m.GroupID, m.GroupName, m.LegalEntityID, m.LegalEntityName, m.RegionID, m.RegionName, m.Name, m.Qty).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: append material: %w", err)
	}
	return id, nil
}

// ListTools 列出师傅工具借用;workerID=0 返回全部。
func (s *PGStore) ListTools(ctx context.Context, workerID int64) ([]Tool, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, `+factSnap+`, name, borrowed FROM worker_tools WHERE ($1 = 0 OR worker_id = $1) ORDER BY id`, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker: list tools: %w", err)
	}
	defer rows.Close()
	out := make([]Tool, 0)
	for rows.Next() {
		var t Tool
		if err := rows.Scan(&t.ID, &t.WorkerID, &t.GroupID, &t.GroupName, &t.LegalEntityID, &t.LegalEntityName, &t.RegionID, &t.RegionName, &t.Name, &t.Borrowed); err != nil {
			return nil, fmt.Errorf("worker: scan tool: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// AppendTool 追加工具借用,返回自增 id。
func (s *PGStore) AppendTool(ctx context.Context, t Tool) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO worker_tools(`+factSnap+`, name, borrowed) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		t.WorkerID, t.GroupID, t.GroupName, t.LegalEntityID, t.LegalEntityName, t.RegionID, t.RegionName, t.Name, t.Borrowed).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: append tool: %w", err)
	}
	return id, nil
}

// ListFeedbacks 列出师傅服务评价;workerID=0 返回全部。
func (s *PGStore) ListFeedbacks(ctx context.Context, workerID int64) ([]Feedback, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, worker_id, worker_name, group_id, group_name, legal_entity_id, legal_entity_name, region_id, region_name, ticket_id, customer_id, customer_name, score, need_review
		 FROM worker_feedbacks WHERE ($1 = 0 OR worker_id = $1) ORDER BY id`, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker: list feedbacks: %w", err)
	}
	defer rows.Close()
	out := make([]Feedback, 0)
	for rows.Next() {
		var f Feedback
		if err := rows.Scan(&f.ID, &f.WorkerID, &f.WorkerName, &f.GroupID, &f.GroupName, &f.LegalEntityID, &f.LegalEntityName, &f.RegionID, &f.RegionName, &f.TicketID, &f.CustomerID, &f.CustomerName, &f.Score, &f.NeedReview); err != nil {
			return nil, fmt.Errorf("worker: scan feedback: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// AppendFeedback 追加服务评价,返回自增 id。
func (s *PGStore) AppendFeedback(ctx context.Context, f Feedback) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO worker_feedbacks(worker_id, worker_name, group_id, group_name, legal_entity_id, legal_entity_name, region_id, region_name, ticket_id, customer_id, customer_name, score, need_review)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`,
		f.WorkerID, f.WorkerName, f.GroupID, f.GroupName, f.LegalEntityID, f.LegalEntityName, f.RegionID, f.RegionName, f.TicketID, f.CustomerID, f.CustomerName, f.Score, f.NeedReview).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: append feedback: %w", err)
	}
	return id, nil
}

// ListAssetReturns 列出师傅资产归还;workerID=0 返回全部。
func (s *PGStore) ListAssetReturns(ctx context.Context, workerID int64) ([]AssetReturn, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, `+factSnap+`, asset_id, reason, status FROM asset_returns WHERE ($1 = 0 OR worker_id = $1) ORDER BY id`, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker: list asset returns: %w", err)
	}
	defer rows.Close()
	out := make([]AssetReturn, 0)
	for rows.Next() {
		var r AssetReturn
		if err := rows.Scan(&r.ID, &r.WorkerID, &r.GroupID, &r.GroupName, &r.LegalEntityID, &r.LegalEntityName, &r.RegionID, &r.RegionName, &r.AssetID, &r.Reason, &r.Status); err != nil {
			return nil, fmt.Errorf("worker: scan asset return: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AppendAssetReturn 追加资产归还,返回自增 id。
func (s *PGStore) AppendAssetReturn(ctx context.Context, r AssetReturn) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO asset_returns(`+factSnap+`, asset_id, reason, status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		r.WorkerID, r.GroupID, r.GroupName, r.LegalEntityID, r.LegalEntityName, r.RegionID, r.RegionName, r.AssetID, r.Reason, r.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: append asset return: %w", err)
	}
	return id, nil
}
