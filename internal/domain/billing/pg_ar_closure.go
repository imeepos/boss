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
	rows, err := s.db.Query(ctx, `SELECT id, customer_id, amount, reason, COALESCE(approved_by,0), approved_at, status, COALESCE(requested_by,0), COALESCE(approved_note,'') FROM ar_writeoffs WHERE ($1::bigint=0 OR customer_id=$1) ORDER BY id DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("billing: list writeoffs: %w", err)
	}
	defer rows.Close()
	out := make([]Writeoff, 0)
	for rows.Next() {
		var v Writeoff
		if err := rows.Scan(&v.ID, &v.CustomerID, &v.Amount, &v.Reason, &v.ApprovedBy, &v.ApprovedAt, &v.Status, &v.RequestedBy, &v.ApprovedNote); err != nil {
			return nil, fmt.Errorf("billing: scan writeoff: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *PGStore) CreateWriteoff(ctx context.Context, v Writeoff) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `INSERT INTO ar_writeoffs(customer_id, amount, reason, requested_by, status) VALUES($1,$2,$3,$4,'PENDING') RETURNING id`, v.CustomerID, v.Amount, v.Reason, v.RequestedBy).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("billing: create writeoff: %w", err)
	}
	return id, nil
}

func (s *PGStore) ApproveWriteoff(ctx context.Context, id, approver int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE ar_writeoffs SET status='APPROVED', approved_by=$2, approved_at=now() WHERE id=$1 AND status='PENDING'`, id, approver)
	if err != nil {
		return fmt.Errorf("billing: approve writeoff: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PGStore) RejectWriteoff(ctx context.Context, id int64, note string) error {
	tag, err := s.db.Exec(ctx, `UPDATE ar_writeoffs SET status='REJECTED', approved_note=$2 WHERE id=$1 AND status='PENDING'`, id, note)
	if err != nil {
		return fmt.Errorf("billing: reject writeoff: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// EvaluateCreditProfile evaluates credit level using rules: STANDARD → WATCH → RESTRICTED → SUSPENDED.
// stop_threshold = total_amount * 1.5 (rule-based buffer).
func (s *PGStore) EvaluateCreditProfile(ctx context.Context, customerID int64) (*CreditProfile, error) {
	var cp CreditProfile
	err := s.db.QueryRow(ctx, `
		WITH arrears_sum AS (
			SELECT COALESCE(SUM(amount),0) AS total, COALESCE(MAX(days),0) AS max_days
			FROM arrears WHERE customer_id=$1
		)
		SELECT $1,
			CASE
				WHEN max_days > 90 OR total > 5000 THEN 'SUSPENDED'
				WHEN max_days > 60 THEN 'RESTRICTED'
				WHEN max_days > 30 OR total > 1000 THEN 'WATCH'
				ELSE 'STANDARD'
			END,
			LEAST(total/max(1.0,1000), 5.0) AS risk_score,
			GREATEST(total*1.5, 500) AS stop_threshold,
			now()
		FROM arrears_sum`, customerID).Scan(&cp.CustomerID, &cp.CreditLevel, &cp.RiskScore, &cp.StopThreshold, &cp.EvaluatedAt)
	if err != nil {
		return nil, fmt.Errorf("billing: evaluate credit profile: %w", err)
	}
	// Upsert into ar_credit_profiles
	_, err = s.db.Exec(ctx, `INSERT INTO ar_credit_profiles(customer_id,credit_level,risk_score,stop_threshold,evaluated_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT(customer_id) DO UPDATE SET credit_level=EXCLUDED.credit_level,risk_score=EXCLUDED.risk_score,stop_threshold=EXCLUDED.stop_threshold,evaluated_at=EXCLUDED.evaluated_at`, cp.CustomerID, cp.CreditLevel, cp.RiskScore, cp.StopThreshold, cp.EvaluatedAt)
	if err != nil {
		return nil, fmt.Errorf("billing: upsert credit profile: %w", err)
	}
	return &cp, nil
}

// GenerateCollectionTasks creates tasks for overdue customers without open tasks.
func (s *PGStore) GenerateCollectionTasks(ctx context.Context, day time.Time) (int, error) {
	date := day.UTC().Format("2006-01-02")
	tag, err := s.db.Exec(ctx, `
		INSERT INTO ar_collection_tasks(customer_id, aging_snapshot_id, task_type, priority, status, due_at)
		SELECT a.customer_id, s.id, 'PHONE_CALL',
			CASE WHEN s.days_90_plus > 0 THEN 'HIGH' WHEN s.days_61_90 > 0 THEN 'HIGH' WHEN s.days_31_60 > 0 THEN 'NORMAL' ELSE 'NORMAL' END,
			'PENDING', $1::date + INTERVAL '3 days'
		FROM ar_aging_snapshots s
		JOIN arrears a ON a.customer_id = s.customer_id
		WHERE s.snapshot_date = $1::date AND s.total_amount > 0
		AND NOT EXISTS (SELECT 1 FROM ar_collection_tasks t WHERE t.customer_id = a.customer_id AND t.status = 'PENDING')
		ON CONFLICT DO NOTHING`, date)
	if err != nil {
		return 0, fmt.Errorf("billing: generate collection tasks: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (s *PGStore) RecordReplayEvent(ctx context.Context, customerID int64, sourceType string, sourceID int64, eventType, payload string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO ar_replay_events(customer_id,source_type,source_id,event_type,payload) VALUES($1,$2,$3,$4,$5::jsonb) ON CONFLICT(source_type,source_id,event_type) DO NOTHING`, customerID, sourceType, sourceID, eventType, payload)
	if err != nil {
		return fmt.Errorf("billing: record replay event: %w", err)
	}
	return nil
}

func (s *PGStore) ReplayEvents(ctx context.Context, sourceType string) (int, error) {
	rows, err := s.db.Query(ctx, `SELECT id, customer_id, source_type, source_id, event_type, payload::text, created_at FROM ar_replay_events WHERE source_type=$1 ORDER BY id`, sourceType)
	if err != nil {
		return 0, fmt.Errorf("billing: list replay events: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var e ReplayEvent
		if err := rows.Scan(&e.ID, &e.CustomerID, &e.SourceType, &e.SourceID, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return count, fmt.Errorf("billing: scan replay event: %w", err)
		}
		// Replay: recalculate arrears from bills, payments, writeoffs for this customer
		_, err := s.db.Exec(ctx, `
			UPDATE arrears a SET amount = COALESCE((
				SELECT COALESCE(SUM(b.amount),0) - COALESCE(SUM(p.amount) FILTER (WHERE p.status='SUCCESS'),0) - COALESCE(SUM(w.amount) FILTER (WHERE w.status='APPROVED'),0)
				FROM bills b LEFT JOIN payments p ON p.customer_id=b.customer_id AND p.status='SUCCESS' LEFT JOIN ar_writeoffs w ON w.customer_id=b.customer_id AND w.status='APPROVED'
				WHERE b.customer_id=$1 AND b.status='OVERDUE'
			), 0) WHERE a.customer_id=$1`, e.CustomerID)
		if err != nil {
			return count, fmt.Errorf("billing: replay event %d: %w", e.ID, err)
		}
		count++
	}
	return count, rows.Err()
}

func (s *PGStore) ListReplayEvents(ctx context.Context, sourceType string) ([]ReplayEvent, error) {
	rows, err := s.db.Query(ctx, `SELECT id, customer_id, source_type, source_id, event_type, payload::text, created_at FROM ar_replay_events WHERE ($1='' OR source_type=$1) ORDER BY id`, sourceType)
	if err != nil {
		return nil, fmt.Errorf("billing: list replay events: %w", err)
	}
	defer rows.Close()
	out := make([]ReplayEvent, 0)
	for rows.Next() {
		var e ReplayEvent
		if err := rows.Scan(&e.ID, &e.CustomerID, &e.SourceType, &e.SourceID, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("billing: scan replay event: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
