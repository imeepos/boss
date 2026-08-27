// 换新单读写与状态机(adopted note 2026-08-27-replacement-ticket-flow):
// PENDING --Assign--> DOING --Complete--> DONE/FAILED;流转用守卫 UPDATE
// (WHERE status=$expected + RowsAffected)防并发越态,与 HandleStocktakeDiff 同款。
package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrIllegalTransition 换新单状态流转非法(终态不可再流转/重复派单)。
var ErrIllegalTransition = errors.New("asset: replacement status transition not allowed")

// scanReplacement 列 → Replacement(可空列 COALESCE/pgtype 归零值)。
func scanReplacement(row pgx.Row) (*Replacement, error) {
	var r Replacement
	var finishedAt pgtype.Timestamptz
	err := row.Scan(&r.ID, &r.ReplacementNo, &r.AssetID, &r.LegalEntityID, &r.LegalEntityName,
		&r.Reason, &r.Priority, &r.Status, &r.WorkerID, &r.WorkerName, &finishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("asset: scan replacement: %w", err)
	}
	if finishedAt.Valid {
		t := finishedAt.Time
		r.FinishedAt = &t
	}
	return &r, nil
}

// replacementCols 换新单查询列(与 scanReplacement 严格对序)。
const replacementCols = `id, replacement_no, asset_id, legal_entity_id, legal_entity_name,
		 reason, priority, status, COALESCE(worker_id, 0), worker_name, finished_at`

// ListReplacements 列出全部换新单。
func (s *PGStore) ListReplacements(ctx context.Context) ([]Replacement, error) {
	rows, err := s.db.Query(ctx, `SELECT `+replacementCols+` FROM replacements ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("asset: list replacements: %w", err)
	}
	defer rows.Close()
	out := make([]Replacement, 0)
	for rows.Next() {
		r, err := scanReplacement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// GetReplacement 按 id 查换新单;未命中返回 ErrNotFound。
func (s *PGStore) GetReplacement(ctx context.Context, id int64) (*Replacement, error) {
	r, err := scanReplacement(s.db.QueryRow(ctx,
		`SELECT `+replacementCols+` FROM replacements WHERE id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("asset: get replacement: %w", err)
	}
	return r, nil
}

// ListReplacementsByWorker 列出师傅名下的换新单(师傅端任务列表)。
func (s *PGStore) ListReplacementsByWorker(ctx context.Context, workerID int64) ([]Replacement, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+replacementCols+` FROM replacements WHERE worker_id = $1 ORDER BY id`, workerID)
	if err != nil {
		return nil, fmt.Errorf("asset: list replacements by worker: %w", err)
	}
	defer rows.Close()
	out := make([]Replacement, 0)
	for rows.Next() {
		r, err := scanReplacement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// CreateReplacement 新建换新单,返回自增 id。
// 企业锚点铁律(fields.md §8.1):法人快照缺失时从资产主档回填;资产不存在拒绝建单。
func (s *PGStore) CreateReplacement(ctx context.Context, r Replacement) (int64, error) {
	ast, err := s.GetAsset(ctx, r.AssetID)
	if err != nil {
		return 0, fmt.Errorf("asset: replacement asset %d: %w", r.AssetID, ErrForeignKeyViolation)
	}
	if r.LegalEntityID == 0 || r.LegalEntityName == "" {
		r.LegalEntityID, r.LegalEntityName = ast.LegalEntityID, ast.LegalEntityName
	}
	var id int64
	err = s.db.QueryRow(ctx, `
		INSERT INTO replacements(replacement_no, asset_id, legal_entity_id, legal_entity_name, reason, priority, status)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		r.ReplacementNo, r.AssetID, r.LegalEntityID, r.LegalEntityName, r.Reason, r.Priority, r.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: create replacement: %w", err)
	}
	return id, nil
}

// AssignReplacement 派单:回填师傅快照并 PENDING→DOING。
func (s *PGStore) AssignReplacement(ctx context.Context, id, workerID int64, workerName string) (*Replacement, error) {
	if ok, err := s.exists(ctx, "workers", workerID); err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("asset: replacement worker %d: %w", workerID, ErrForeignKeyViolation)
	}
	r, err := scanReplacement(s.db.QueryRow(ctx, `
		UPDATE replacements SET worker_id = $2, worker_name = $3, status = 'DOING'
		WHERE id = $1 AND status = 'PENDING'
		RETURNING `+replacementCols, id, idOrNil(workerID), workerName))
	if errors.Is(err, ErrNotFound) {
		return nil, s.transitionFailReason(ctx, id, "PENDING")
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// CompleteReplacement 完成/失败:DOING→DONE|FAILED 并回填 finished_at。
func (s *PGStore) CompleteReplacement(ctx context.Context, id int64, result string) (*Replacement, error) {
	if result != "DONE" && result != "FAILED" {
		return nil, fmt.Errorf("asset: replacement complete result %q: %w", result, ErrIllegalTransition)
	}
	r, err := scanReplacement(s.db.QueryRow(ctx, `
		UPDATE replacements SET status = $2, finished_at = now()
		WHERE id = $1 AND status = 'DOING'
		RETURNING `+replacementCols, id, result))
	if errors.Is(err, ErrNotFound) {
		return nil, s.transitionFailReason(ctx, id, "DOING")
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// transitionFailReason 守卫 UPDATE 0 行时区分"单不存在"与"状态非预期"。
func (s *PGStore) transitionFailReason(ctx context.Context, id int64, expect string) error {
	_, err := s.GetReplacement(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("asset: replacement %d status is not %s: %w", id, expect, ErrIllegalTransition)
}
