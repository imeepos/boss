// 盘点差异明细读写:快照清单 / 扫码回填 / 逐条处置。
// 关单守卫(HandleStocktakeDiff)见 pg_write.go。
package asset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// stocktakeResolution 处置动作 → 落库 resolution。
var stocktakeResolution = map[string]string{
	"CONFIRM":  "CONFIRMED",
	"FIX":      "FIXED",
	"ESCALATE": "ESCALATED",
}

// ListStocktakeItems 盘点差异明细清单,按行 id 升序。
func (s *PGStore) ListStocktakeItems(ctx context.Context, taskID int64) ([]StocktakeItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, task_id, asset_id, COALESCE(expected_status,''), COALESCE(scanned_status,''),
		       scanned_at, kind, resolution, COALESCE(handled_by,0), handled_at, COALESCE(note,'')
		FROM stocktake_items WHERE task_id = $1 ORDER BY id`, taskID)
	if err != nil {
		return nil, fmt.Errorf("asset: list stocktake items: %w", err)
	}
	defer rows.Close()
	out := make([]StocktakeItem, 0)
	for rows.Next() {
		var it StocktakeItem
		var scannedAt, handledAt pgtype.Timestamptz
		if err := rows.Scan(&it.ID, &it.TaskID, &it.AssetID, &it.ExpectedStatus, &it.ScannedStatus,
			&scannedAt, &it.Kind, &it.Resolution, &it.HandledBy, &handledAt, &it.Note); err != nil {
			return nil, fmt.Errorf("asset: scan stocktake item: %w", err)
		}
		it.ScannedAt, it.HandledAt = tsPtr(scannedAt), tsPtr(handledAt)
		out = append(out, it)
	}
	return out, rows.Err()
}

// ScanStocktake 回填一次扫码:预期行判 OK/MISMATCH(重扫重开差异),计划外行记 EXTRA;
// 重算任务进度(实扫/计划)与差异数,返回(明细 id, kind)。
func (s *PGStore) ScanStocktake(ctx context.Context, taskID, assetID int64, scannedStatus string) (int64, string, error) {
	if err := s.scanGuards(ctx, taskID, assetID); err != nil {
		return 0, "", err
	}
	if err := s.upsertScan(ctx, taskID, assetID, scannedStatus); err != nil {
		return 0, "", err
	}
	if err := s.recalcScanProgress(ctx, taskID); err != nil {
		return 0, "", err
	}
	var id int64
	var kind string
	err := s.db.QueryRow(ctx,
		`SELECT id, kind FROM stocktake_items WHERE task_id = $1 AND asset_id = $2`,
		taskID, assetID).Scan(&id, &kind)
	if err != nil {
		return 0, "", fmt.Errorf("asset: reload scan line %d/%d: %w", taskID, assetID, err)
	}
	return id, kind, nil
}

// scanGuards 任务须在盘(DOING),资产须存在。
func (s *PGStore) scanGuards(ctx context.Context, taskID, assetID int64) error {
	var status string
	err := s.db.QueryRow(ctx, `SELECT status FROM stocktakes WHERE id = $1`, taskID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("asset: load stocktake %d: %w", taskID, err)
	}
	if status != "DOING" {
		return fmt.Errorf("asset: stocktake %d is %s: %w", taskID, status, ErrStocktakeState)
	}
	ok, err := s.exists(ctx, "assets", assetID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("asset: asset %d: %w", assetID, ErrForeignKeyViolation)
	}
	return nil
}

// upsertScan 预期行 UPDATE 判定;计划外行 INSERT EXTRA(重复扫 EXTRA 行则刷新实盘证据)。
func (s *PGStore) upsertScan(ctx context.Context, taskID, assetID int64, scannedStatus string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE stocktake_items SET scanned_status = $3, scanned_at = now(), resolution = 'OPEN',
		       kind = CASE WHEN expected_status = $3 THEN 'OK' ELSE 'MISMATCH' END
		 WHERE task_id = $1 AND asset_id = $2 AND expected_status IS NOT NULL`,
		taskID, assetID, scannedStatus)
	if err != nil {
		return fmt.Errorf("asset: scan expected line: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	if _, err := s.db.Exec(ctx, `
		INSERT INTO stocktake_items(task_id, asset_id, expected_status, scanned_status, scanned_at, kind, resolution)
		VALUES($1,$2,NULL,$3,now(),'EXTRA','OPEN')
		ON CONFLICT (task_id, asset_id) DO UPDATE
		SET scanned_status = EXCLUDED.scanned_status, scanned_at = now(), kind = 'EXTRA'`,
		taskID, assetID, scannedStatus); err != nil {
		return fmt.Errorf("asset: scan extra line: %w", err)
	}
	return nil
}

// recalcScanProgress 进度 = 实扫/计划快照行;差异数 = MISMATCH/MISSING/EXTRA 行数。
func (s *PGStore) recalcScanProgress(ctx context.Context, taskID int64) error {
	if _, err := s.db.Exec(ctx, `
		UPDATE stocktakes SET progress = CASE WHEN agg.total = 0 THEN 0
		        ELSE ROUND(100.0 * agg.scanned / agg.total)::smallint END,
		    diff_count = agg.diffs
		FROM (SELECT count(*) AS total, count(scanned_status) AS scanned,
		             count(*) FILTER (WHERE kind IN ('MISMATCH','MISSING','EXTRA')) AS diffs
		      FROM stocktake_items WHERE task_id = $1) agg
		WHERE id = $1`, taskID); err != nil {
		return fmt.Errorf("asset: recalc stocktake %d: %w", taskID, err)
	}
	return nil
}

// HandleStocktakeItem 逐条处置差异:
// CONFIRM 承认差异(MISMATCH 时按实盘修正台账+留轨迹;MISSING/EXTRA 仅留痕);
// FIX 现场核实台账为准;ESCALATE 上报转人工。均要求任务在盘、差异行 OPEN。
func (s *PGStore) HandleStocktakeItem(ctx context.Context, taskID, itemID int64, action string, accountID int64) error {
	res, ok := stocktakeResolution[action]
	if !ok {
		return fmt.Errorf("asset: action %q: %w", action, ErrStocktakeState)
	}
	it, err := s.loadHandleableItem(ctx, taskID, itemID)
	if err != nil {
		return err
	}
	if action == "CONFIRM" {
		if err := s.applyConfirmedScan(ctx, it); err != nil {
			return err
		}
	}
	if _, err := s.db.Exec(ctx, `
		UPDATE stocktake_items SET resolution = $3, handled_by = $4, handled_at = $5
		WHERE id = $1 AND task_id = $2`,
		itemID, taskID, res, idOrNil(accountID), time.Now()); err != nil {
		return fmt.Errorf("asset: resolve item %d: %w", itemID, err)
	}
	return nil
}

// loadHandleableItem 载入可处置行:任务 DOING、差异行(kind≠OK)且 OPEN。
func (s *PGStore) loadHandleableItem(ctx context.Context, taskID, itemID int64) (*StocktakeItem, error) {
	var status string
	err := s.db.QueryRow(ctx, `SELECT status FROM stocktakes WHERE id = $1`, taskID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("asset: load stocktake %d: %w", taskID, err)
	}
	if status != "DOING" {
		return nil, fmt.Errorf("asset: stocktake %d is %s: %w", taskID, status, ErrStocktakeState)
	}
	it := &StocktakeItem{}
	err = s.db.QueryRow(ctx, `
		SELECT id, task_id, asset_id, COALESCE(expected_status,''), COALESCE(scanned_status,''), kind, resolution
		FROM stocktake_items WHERE id = $1 AND task_id = $2`, itemID, taskID).
		Scan(&it.ID, &it.TaskID, &it.AssetID, &it.ExpectedStatus, &it.ScannedStatus, &it.Kind, &it.Resolution)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("asset: load item %d: %w", itemID, err)
	}
	if it.Kind == "OK" || it.Resolution != "OPEN" {
		return nil, fmt.Errorf("asset: item %d kind=%s resolution=%s: %w",
			itemID, it.Kind, it.Resolution, ErrStocktakeState)
	}
	return it, nil
}

// applyConfirmedScan CONFIRM 且实盘与快照不一致时,把台账修正到实盘并留轨迹。
func (s *PGStore) applyConfirmedScan(ctx context.Context, it *StocktakeItem) error {
	if it.ScannedStatus == "" || it.ScannedStatus == it.ExpectedStatus {
		return nil // MISSING/EXTRA 或已一致:台账不动,仅处置留痕
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE assets SET status = $2 WHERE id = $1`, it.AssetID, it.ScannedStatus); err != nil {
		return fmt.Errorf("asset: confirm item %d: %w", it.ID, err)
	}
	if _, err := s.db.Exec(ctx,
		`INSERT INTO asset_lifecycles(asset_id, status, changed_at) VALUES($1,$2,now())`,
		it.AssetID, it.ScannedStatus); err != nil {
		return fmt.Errorf("asset: confirm item %d lifecycle: %w", it.ID, err)
	}
	return nil
}

// tsPtr pgtype.Timestamptz → *time.Time(NULL→nil)。
func tsPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
