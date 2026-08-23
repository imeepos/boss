// WebhookStore 的 PostgreSQL 实现(迁移 000125 open_webhook_deliveries)。
package openplat

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// InsertDeliveries 为匹配订阅批量建投递行;幂等(UNIQUE + DO NOTHING)。
func (s *PGStore) InsertDeliveries(ctx context.Context, eventType, eventID string, payload []byte) (int64, error) {
	var n int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO open_webhook_deliveries (subscription_id, event_id, event_type, payload)
		SELECT sub.id, $1, sub.event_type, $3
		FROM open_webhook_subscriptions sub
		JOIN open_apps app ON app.id = sub.app_id
		WHERE sub.event_type = $2 AND sub.status = 1 AND app.status = 1
		ON CONFLICT (subscription_id, event_id) DO NOTHING`,
		eventID, eventType, payload).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("openplat: insert deliveries: %w", err)
	}
	return n, nil
}

const dueCols = `d.id, d.subscription_id, d.event_id, d.event_type, d.payload::text,
	d.status, d.attempts, d.next_attempt_at, d.http_status, d.last_error, d.delivered_at, d.created_at`

// ListDue 取到期投递(含订阅端点与应用 Secret)。
func (s *PGStore) ListDue(ctx context.Context, now time.Time, limit int) ([]DueDelivery, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+dueCols+`, sub.endpoint_url, app.secret
		FROM open_webhook_deliveries d
		JOIN open_webhook_subscriptions sub ON sub.id = d.subscription_id
		JOIN open_apps app ON app.id = sub.app_id
		WHERE d.status = 0 AND d.next_attempt_at <= $1
		ORDER BY d.next_attempt_at LIMIT $2`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("openplat: list due: %w", err)
	}
	defer rows.Close()
	out := make([]DueDelivery, 0)
	for rows.Next() {
		var dl DueDelivery
		var next, delivered, created pgtype.Timestamptz
		if err := rows.Scan(&dl.ID, &dl.SubscriptionID, &dl.EventID, &dl.EventType, &dl.Payload,
			&dl.Status, &dl.Attempts, &next, &dl.HTTPStatus, &dl.LastError, &delivered, &created,
			&dl.EndpointURL, &dl.Secret); err != nil {
			return nil, fmt.Errorf("openplat: scan due: %w", err)
		}
		dl.NextAttemptAt = fmtTime(next)
		dl.DeliveredAt = fmtTime(delivered)
		dl.CreatedAt = fmtTime(created)
		out = append(out, dl)
	}
	return out, rows.Err()
}

// MarkResult 落投递结果;失败按 Backoff(attempts+1) 排下次,超上限死信。
func (s *PGStore) MarkResult(ctx context.Context, id int64, success bool, httpStatus int, errMsg string) error {
	var err error
	if success {
		_, err = s.db.Exec(ctx, `
			UPDATE open_webhook_deliveries
			SET status=1, attempts=attempts+1, http_status=$2, last_error='', delivered_at=now()
			WHERE id=$1`, id, httpStatus)
	} else {
		_, err = s.db.Exec(ctx, `
			UPDATE open_webhook_deliveries
			SET attempts=attempts+1, http_status=$2, last_error=$3,
			    status = CASE WHEN attempts+1 >= $4 THEN 2 ELSE 0 END,
			    next_attempt_at = now() + least(interval '1 hour',
			        interval '30 seconds' * power(2::float8, attempts))
			WHERE id=$1`, id, httpStatus, errMsg, MaxAttempts)
	}
	if err != nil {
		return fmt.Errorf("openplat: mark result: %w", err)
	}
	return nil
}

// ListDeliveries 管理面查询(按订阅,0=全部,倒序)。
func (s *PGStore) ListDeliveries(ctx context.Context, subscriptionID int64) ([]Delivery, error) {
	q := `SELECT ` + dueCols + ` FROM open_webhook_deliveries d`
	args := []any{}
	if subscriptionID > 0 {
		q += ` WHERE d.subscription_id = $1`
		args = append(args, subscriptionID)
	}
	q += ` ORDER BY d.id DESC LIMIT 200`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("openplat: list deliveries: %w", err)
	}
	defer rows.Close()
	out := make([]Delivery, 0)
	for rows.Next() {
		var dl Delivery
		var next, delivered, created pgtype.Timestamptz
		if err := rows.Scan(&dl.ID, &dl.SubscriptionID, &dl.EventID, &dl.EventType, &dl.Payload,
			&dl.Status, &dl.Attempts, &next, &dl.HTTPStatus, &dl.LastError, &delivered, &created); err != nil {
			return nil, fmt.Errorf("openplat: scan delivery: %w", err)
		}
		dl.NextAttemptAt = fmtTime(next)
		dl.DeliveredAt = fmtTime(delivered)
		dl.CreatedAt = fmtTime(created)
		out = append(out, dl)
	}
	return out, rows.Err()
}

// Requeue 重置死信/失败行为待投递(attempts 清零,立即到期)。
func (s *PGStore) Requeue(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE open_webhook_deliveries
		SET status=0, attempts=0, next_attempt_at=now()
		WHERE id=$1 AND status != 1`, id)
	if err != nil {
		return fmt.Errorf("openplat: requeue: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
