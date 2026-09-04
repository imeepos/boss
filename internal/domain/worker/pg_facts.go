package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const factSnap = `worker_id, group_id, group_name, legal_entity_id, legal_entity_name, region_id, region_name`

// ResolveFactSnapshot 解析师傅当前班组/法人/区域快照。
// 2026-09-04 任务A:工具借还/物料领用此前由 handler 传零值 group_id,稳定撞
// worker_tools_group_id_fkey(23505)→ 50000;现改服务端解析,主档不可用回 40400。
func (s *PGStore) ResolveFactSnapshot(ctx context.Context, workerID int64) (*FactSnapshot, error) {
	var f FactSnapshot
	err := s.db.QueryRow(ctx, `
		SELECT w.id, w.group_id, g.name, g.legal_entity_id, le.name, w.region_id, COALESCE(r.name, '')
		FROM workers w
		JOIN worker_groups g ON g.id = w.group_id
		JOIN legal_entities le ON le.id = g.legal_entity_id
		LEFT JOIN regions r ON r.id = w.region_id
		WHERE w.id = $1`, workerID).
		Scan(&f.WorkerID, &f.GroupID, &f.GroupName, &f.LegalEntityID, &f.LegalEntityName, &f.RegionID, &f.RegionName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGroupInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("worker: resolve fact snapshot: %w", err)
	}
	return &f, nil
}

// ListPerformances 列出师傅绩效;workerID=0 返回全部。
func (s *PGStore) ListPerformances(ctx context.Context, workerID int64) ([]Performance, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, `+factSnap+`, period, finished, on_time_rate, score
		FROM worker_performances WHERE ($1 = 0 OR worker_id = $1) ORDER BY period, id`, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker: list performances: %w", err)
	}
	defer rows.Close()
	out := make([]Performance, 0)
	for rows.Next() {
		var p Performance
		if err := rows.Scan(&p.ID, &p.WorkerID, &p.GroupID, &p.GroupName, &p.LegalEntityID, &p.LegalEntityName, &p.RegionID, &p.RegionName, &p.Period, &p.Finished, &p.OnTimeRate, &p.Score); err != nil {
			return nil, fmt.Errorf("worker: scan performance: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpsertPerformance 写入/更新绩效(师傅×月×班组×区域),返回 id。
func (s *PGStore) UpsertPerformance(ctx context.Context, p Performance) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO worker_performances(`+factSnap+`, period, finished, on_time_rate, score)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (worker_id, period, group_id, region_id) DO UPDATE SET
			group_name = $3, legal_entity_id = $4, legal_entity_name = $5, region_id = $6, region_name = $7,
			finished = $9, on_time_rate = $10, score = $11
		RETURNING id`,
		p.WorkerID, p.GroupID, p.GroupName, p.LegalEntityID, p.LegalEntityName, p.RegionID, p.RegionName,
		p.Period, p.Finished, p.OnTimeRate, p.Score).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: upsert performance: %w", err)
	}
	return id, nil
}

// ListCommissions 列出师傅佣金;workerID=0 返回全部。
func (s *PGStore) ListCommissions(ctx context.Context, workerID int64) ([]Commission, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, `+factSnap+`, period, formula, amount
		FROM worker_commissions WHERE ($1 = 0 OR worker_id = $1) ORDER BY period, id`, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker: list commissions: %w", err)
	}
	defer rows.Close()
	out := make([]Commission, 0)
	for rows.Next() {
		var c Commission
		if err := rows.Scan(&c.ID, &c.WorkerID, &c.GroupID, &c.GroupName, &c.LegalEntityID, &c.LegalEntityName, &c.RegionID, &c.RegionName, &c.Period, &c.Formula, &c.Amount); err != nil {
			return nil, fmt.Errorf("worker: scan commission: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpsertCommission 写入/更新佣金(师傅×月×班组×区域),返回 id。
func (s *PGStore) UpsertCommission(ctx context.Context, c Commission) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO worker_commissions(`+factSnap+`, period, formula, amount)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (worker_id, period, group_id, region_id) DO UPDATE SET
			group_name = $3, legal_entity_id = $4, legal_entity_name = $5, region_id = $6, region_name = $7,
			formula = $9, amount = $10
		RETURNING id`,
		c.WorkerID, c.GroupID, c.GroupName, c.LegalEntityID, c.LegalEntityName, c.RegionID, c.RegionName,
		c.Period, c.Formula, c.Amount).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: upsert commission: %w", err)
	}
	return id, nil
}

// ListSchedules 列出师傅考勤;workerID=0 返回全部。
func (s *PGStore) ListSchedules(ctx context.Context, workerID int64) ([]Schedule, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, `+factSnap+`, month, busy_days
		FROM worker_schedules WHERE ($1 = 0 OR worker_id = $1) ORDER BY month, id`, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker: list schedules: %w", err)
	}
	defer rows.Close()
	out := make([]Schedule, 0)
	for rows.Next() {
		var sc Schedule
		if err := rows.Scan(&sc.ID, &sc.WorkerID, &sc.GroupID, &sc.GroupName, &sc.LegalEntityID, &sc.LegalEntityName, &sc.RegionID, &sc.RegionName, &sc.Month, &sc.BusyDays); err != nil {
			return nil, fmt.Errorf("worker: scan schedule: %w", err)
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

// UpsertSchedule 写入/更新考勤(师傅×月×班组×区域),返回 id。
func (s *PGStore) UpsertSchedule(ctx context.Context, sc Schedule) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO worker_schedules(`+factSnap+`, month, busy_days)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (worker_id, month, group_id, region_id) DO UPDATE SET
			group_name = $3, legal_entity_id = $4, legal_entity_name = $5, region_id = $6, region_name = $7,
			busy_days = $9
		RETURNING id`,
		sc.WorkerID, sc.GroupID, sc.GroupName, sc.LegalEntityID, sc.LegalEntityName, sc.RegionID, sc.RegionName,
		sc.Month, sc.BusyDays).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: upsert schedule: %w", err)
	}
	return id, nil
}
