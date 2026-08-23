package billing

import (
	"context"
	"fmt"
	"time"
)

func (s *PGStore) GenerateAgingSnapshots(ctx context.Context, day time.Time) (int, error) {
	date := day.UTC().Format("2006-01-02")
	tag, err := s.db.Exec(ctx, `
		INSERT INTO ar_aging_snapshots
		(customer_id, snapshot_date, current_amount, days_1_30, days_31_60, days_61_90, days_90_plus, total_amount)
		SELECT customer_id, $1::date,
			COALESCE(SUM(amount) FILTER (WHERE days BETWEEN 0 AND 0), 0),
			COALESCE(SUM(amount) FILTER (WHERE days BETWEEN 1 AND 30), 0),
			COALESCE(SUM(amount) FILTER (WHERE days BETWEEN 31 AND 60), 0),
			COALESCE(SUM(amount) FILTER (WHERE days BETWEEN 61 AND 90), 0),
			COALESCE(SUM(amount) FILTER (WHERE days > 90), 0), COALESCE(SUM(amount), 0)
		FROM arrears GROUP BY customer_id
		ON CONFLICT (customer_id, snapshot_date) DO UPDATE SET
		current_amount=EXCLUDED.current_amount, days_1_30=EXCLUDED.days_1_30,
		days_31_60=EXCLUDED.days_31_60, days_61_90=EXCLUDED.days_61_90,
		days_90_plus=EXCLUDED.days_90_plus, total_amount=EXCLUDED.total_amount`, date)
	if err != nil {
		return 0, fmt.Errorf("billing: generate aging snapshots: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (s *PGStore) ListAgingSnapshots(ctx context.Context, day time.Time, customerID int64) ([]AgingSnapshot, error) {
	rows, err := s.db.Query(ctx, `SELECT id, customer_id, snapshot_date, current_amount, days_1_30, days_31_60, days_61_90, days_90_plus, total_amount FROM ar_aging_snapshots WHERE snapshot_date=$1::date AND ($2::bigint=0 OR customer_id=$2) ORDER BY customer_id`, day.UTC().Format("2006-01-02"), customerID)
	if err != nil {
		return nil, fmt.Errorf("billing: list aging snapshots: %w", err)
	}
	defer rows.Close()
	out := make([]AgingSnapshot, 0)
	for rows.Next() {
		var v AgingSnapshot
		if err := rows.Scan(&v.ID, &v.CustomerID, &v.SnapshotDate, &v.Current, &v.Days1To30, &v.Days31To60, &v.Days61To90, &v.Days90Plus, &v.Total); err != nil {
			return nil, fmt.Errorf("billing: scan aging snapshot: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *PGStore) ListPaymentPromises(ctx context.Context, customerID int64) ([]PaymentPromise, error) {
	rows, err := s.db.Query(ctx, `SELECT id, customer_id, amount, promised_at, due_at, status, fulfilled_at FROM ar_payment_promises WHERE ($1::bigint=0 OR customer_id=$1) ORDER BY due_at, id`, customerID)
	if err != nil {
		return nil, fmt.Errorf("billing: list payment promises: %w", err)
	}
	defer rows.Close()
	out := make([]PaymentPromise, 0)
	for rows.Next() {
		var v PaymentPromise
		if err := rows.Scan(&v.ID, &v.CustomerID, &v.Amount, &v.PromisedAt, &v.DueAt, &v.Status, &v.FulfilledAt); err != nil {
			return nil, fmt.Errorf("billing: scan payment promise: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *PGStore) CreatePaymentPromise(ctx context.Context, v PaymentPromise) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `INSERT INTO ar_payment_promises(customer_id, amount, promised_at, due_at, status) VALUES($1,$2,$3,$4,COALESCE(NULLIF($5,''),'OPEN')) RETURNING id`, v.CustomerID, v.Amount, v.PromisedAt, v.DueAt, v.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("billing: create payment promise: %w", err)
	}
	return id, nil
}

func (s *PGStore) UpdatePaymentPromiseStatus(ctx context.Context, id int64, status string) error {
	tag, err := s.db.Exec(ctx, `UPDATE ar_payment_promises SET status=$2, fulfilled_at=CASE WHEN $2='FULFILLED' THEN COALESCE(fulfilled_at,now()) ELSE fulfilled_at END WHERE id=$1`, id, status)
	if err != nil {
		return fmt.Errorf("billing: update payment promise: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PGStore) ListWriteoffs(ctx context.Context, customerID int64) ([]Writeoff, error) {
	rows, err := s.db.Query(ctx, `SELECT id, customer_id, amount, reason, approved_by, approved_at FROM ar_writeoffs WHERE ($1::bigint=0 OR customer_id=$1) ORDER BY id DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("billing: list writeoffs: %w", err)
	}
	defer rows.Close()
	out := make([]Writeoff, 0)
	for rows.Next() {
		var v Writeoff
		if err := rows.Scan(&v.ID, &v.CustomerID, &v.Amount, &v.Reason, &v.ApprovedBy, &v.ApprovedAt); err != nil {
			return nil, fmt.Errorf("billing: scan writeoff: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *PGStore) CreateWriteoff(ctx context.Context, v Writeoff) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `INSERT INTO ar_writeoffs(customer_id, amount, reason, approved_by, approved_at) VALUES($1,$2,$3,$4,CASE WHEN $4 IS NULL THEN NULL ELSE now() END) RETURNING id`, v.CustomerID, v.Amount, v.Reason, v.ApprovedBy).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("billing: create writeoff: %w", err)
	}
	return id, nil
}
