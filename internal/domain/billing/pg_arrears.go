package billing

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetArrears 按客户查欠费快照;未命中返回 ErrNotFound。
func (s *PGStore) GetArrears(ctx context.Context, customerID int64) (*Arrears, error) {
	var a Arrears
	err := s.db.QueryRow(ctx,
		`SELECT id, customer_id, amount, days, status FROM arrears WHERE customer_id = $1`, customerID).
		Scan(&a.ID, &a.CustomerID, &a.Amount, &a.Days, &a.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("billing: get arrears: %w", err)
	}
	return &a, nil
}

// UpsertArrears 客户唯一欠费快照,存在则更新金额/天数/状态,返回 id。
func (s *PGStore) UpsertArrears(ctx context.Context, a Arrears) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO arrears(customer_id, amount, days, status)
		VALUES($1,$2,$3,$4)
		ON CONFLICT (customer_id) DO UPDATE SET amount = EXCLUDED.amount, days = EXCLUDED.days, status = EXCLUDED.status
		RETURNING id`,
		a.CustomerID, a.Amount, a.Days, a.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("billing: upsert arrears: %w", err)
	}
	return id, nil
}

// ListArrears 欠费列表读模型(联表客户名,按欠费金额倒序)。
func (s *PGStore) ListArrears(ctx context.Context) ([]ArrearsItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT ar.customer_id, COALESCE(c.name,''), ar.amount, ar.days, ar.status
		FROM arrears ar LEFT JOIN customers c ON ar.customer_id = c.id
		ORDER BY ar.amount DESC`)
	if err != nil {
		return nil, fmt.Errorf("billing: list arrears: %w", err)
	}
	defer rows.Close()
	out := make([]ArrearsItem, 0)
	for rows.Next() {
		var it ArrearsItem
		if err := rows.Scan(&it.CustomerID, &it.CustomerName, &it.Amount, &it.Days, &it.Status); err != nil {
			return nil, fmt.Errorf("billing: scan arrears: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ListStopResumeTasks 列出停复机流水;customerID=0 返回全部。
func (s *PGStore) ListStopResumeTasks(ctx context.Context, customerID int64) ([]StopResumeTask, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, customer_id, lo_account_id, action, status FROM stop_resume_tasks WHERE ($1 = 0 OR customer_id = $1) ORDER BY id`, customerID)
	if err != nil {
		return nil, fmt.Errorf("billing: list stop resume: %w", err)
	}
	defer rows.Close()
	out := make([]StopResumeTask, 0)
	for rows.Next() {
		var t StopResumeTask
		if err := rows.Scan(&t.ID, &t.CustomerID, &t.LoAccountID, &t.Action, &t.Status); err != nil {
			return nil, fmt.Errorf("billing: scan stop resume: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// AppendStopResumeTask 追加停复机流水,返回自增 id。
func (s *PGStore) AppendStopResumeTask(ctx context.Context, t StopResumeTask) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO stop_resume_tasks(customer_id, lo_account_id, action, status)
		VALUES($1,$2,$3,$4) RETURNING id`,
		t.CustomerID, t.LoAccountID, t.Action, t.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("billing: append stop resume: %w", err)
	}
	return id, nil
}
