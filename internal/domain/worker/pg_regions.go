package worker

// 师傅负责区域 PG 实现(迁移 000175:worker_regions 多区域;workers.region_id 为主区域)。
// 匹配口径:工单区域 ∈ 师傅负责区域集合(任一方缺失=不限);见 Worker.MatchesRegion。

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// beginner 事务能力按需断言(billing 同款);mock 未实现时调用方仅走非事务路径。
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// loadRegionIDs 批量读负责区域(含主区域;按 worker_regions 行序,主区域由调用方前提)。
func (s *PGStore) loadRegionIDs(ctx context.Context, workerIDs []int64) (map[int64][]int64, error) {
	out := make(map[int64][]int64, len(workerIDs))
	if len(workerIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx,
		`SELECT worker_id, region_id FROM worker_regions WHERE worker_id = ANY($1) ORDER BY worker_id, region_id`, workerIDs)
	if err != nil {
		return nil, fmt.Errorf("worker: load worker regions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var wid, rid int64
		if err := rows.Scan(&wid, &rid); err != nil {
			return nil, fmt.Errorf("worker: scan worker region: %w", err)
		}
		out[wid] = append(out[wid], rid)
	}
	return out, rows.Err()
}

// attachRegionIDs 把负责区域回填进师傅视图(主区域前提;无行时回退 [主区域] 兜底老数据)。
func (s *PGStore) attachRegionIDs(ctx context.Context, workers []*Worker) error {
	ids := make([]int64, 0, len(workers))
	for _, w := range workers {
		ids = append(ids, w.ID)
	}
	byWorker, err := s.loadRegionIDs(ctx, ids)
	if err != nil {
		return err
	}
	for _, w := range workers {
		if rows := byWorker[w.ID]; len(rows) > 0 {
			w.RegionIDs = orderedPrimaryFirst(rows, w.RegionID)
		} else if w.RegionID > 0 {
			w.RegionIDs = []int64{w.RegionID}
		}
	}
	return nil
}

// orderedPrimaryFirst 主区域排首位,其余按 region_id 升序。
func orderedPrimaryFirst(ids []int64, primary int64) []int64 {
	out := make([]int64, 0, len(ids))
	rest := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id == primary {
			out = append(out, id)
		} else {
			rest = append(rest, id)
		}
	}
	return append(out, rest...)
}

// SetWorkerRegions 覆盖式配置负责区域:单事务重建 worker_regions + 主区域回写 workers.region_id。
// 区域不存在返回 ErrForeignKeyViolation;师傅不存在返回 ErrNotFound;
// 空集合=仅清空扩展区域,主区域保留(region_id NOT NULL,清空不产生"无区域"师傅)。
func (s *PGStore) SetWorkerRegions(ctx context.Context, workerID int64, regionIDs []int64) error {
	ok, err := s.exists(ctx, "workers", workerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	ids, primary := NormalizeRegionIDs(regionIDs, 0)
	for _, id := range ids {
		ok, err := s.exists(ctx, "regions", id)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("worker: region %d: %w", id, ErrForeignKeyViolation)
		}
	}
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return fmt.Errorf("worker: begin set regions tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM worker_regions WHERE worker_id = $1`, workerID); err != nil {
		return fmt.Errorf("worker: clear worker regions: %w", err)
	}
	for _, id := range ids {
		if _, err := tx.Exec(ctx,
			`INSERT INTO worker_regions(worker_id, region_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, workerID, id); err != nil {
			return fmt.Errorf("worker: insert worker region %d: %w", id, err)
		}
	}
	if primary > 0 {
		if _, err := tx.Exec(ctx, `UPDATE workers SET region_id = $2 WHERE id = $1`, workerID, primary); err != nil {
			return fmt.Errorf("worker: update primary region: %w", err)
		}
	}
	return tx.Commit(ctx)
}
