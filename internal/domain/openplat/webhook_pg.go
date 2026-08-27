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

// claimLeaseSeconds 领取租约(秒):ListDue 把到期行的 next_attempt_at 推进到
// now()+租约窗口。同批投递最长耗时约 DeliveryBatchMax×10s(HTTP 超时),取 120s 覆盖
// 正常批次;投递器崩溃时行在租约到期后自动重新可见,不丢失、不永久滞留。
const claimLeaseSeconds = 120

// ListDue 原子领取一批到期投递并返回行数据(含订阅端点与应用 Secret)。
// 单语句 UPDATE...FOR UPDATE SKIP LOCKED 完成「领取」:并发循环/多实例重复调用
// 拿到的批次互不相交,同一行不会被同时投给订阅端点两次;崩溃恢复见 claimLeaseSeconds。
func (s *PGStore) ListDue(ctx context.Context, now time.Time, limit int) ([]DueDelivery, error) {
	rows, err := s.db.Query(ctx, `
		WITH claimed AS (
			UPDATE open_webhook_deliveries d
			SET next_attempt_at = now() + ($3::float8 * interval '1 second')
			WHERE d.id IN (
				SELECT d2.id FROM open_webhook_deliveries d2
				JOIN open_webhook_subscriptions s2 ON s2.id = d2.subscription_id
				JOIN open_apps a2 ON a2.id = s2.app_id
				WHERE d2.status = 0 AND d2.next_attempt_at <= $1
				ORDER BY d2.next_attempt_at LIMIT $2
				FOR UPDATE OF d2 SKIP LOCKED
			)
			RETURNING `+dueCols+`
		)
		SELECT c.id, c.subscription_id, c.event_id, c.event_type, c.payload,
			c.status, c.attempts, c.next_attempt_at, c.http_status, c.last_error, c.delivered_at, c.created_at,
			sub.endpoint_url, app.secret
		FROM claimed c
		JOIN open_webhook_deliveries d ON d.id = c.id
		JOIN open_webhook_subscriptions sub ON sub.id = d.subscription_id
		JOIN open_apps app ON app.id = sub.app_id
		ORDER BY c.next_attempt_at`, now, limit, float64(claimLeaseSeconds))
	if err != nil {
		return nil, fmt.Errorf("openplat: list due: %w", err)
	}
	defer rows.Close()
	out := make([]DueDelivery, 0)
	for rows.Next() {
		var dl DueDelivery
		var next, delivered, created pgtype.Timestamptz
		// http_status/last_error 允许 NULL(从未投递过的行),必须可空扫描;
		// 此前固定 *int 扫描使首轮领取即报错,待投递任务永远无法处理。
		var httpStatus pgtype.Int4
		var lastErr pgtype.Text
		if err := rows.Scan(&dl.ID, &dl.SubscriptionID, &dl.EventID, &dl.EventType, &dl.Payload,
			&dl.Status, &dl.Attempts, &next, &httpStatus, &lastErr, &delivered, &created,
			&dl.EndpointURL, &dl.Secret); err != nil {
			return nil, fmt.Errorf("openplat: scan due: %w", err)
		}
		dl.HTTPStatus = int(httpStatus.Int32)
		dl.LastError = lastErr.String
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
		// http_status/last_error 允许 NULL,可空扫描(与 ListDue 同口径)。
		var httpStatus pgtype.Int4
		var lastErr pgtype.Text
		if err := rows.Scan(&dl.ID, &dl.SubscriptionID, &dl.EventID, &dl.EventType, &dl.Payload,
			&dl.Status, &dl.Attempts, &next, &httpStatus, &lastErr, &delivered, &created); err != nil {
			return nil, fmt.Errorf("openplat: scan delivery: %w", err)
		}
		dl.HTTPStatus = int(httpStatus.Int32)
		dl.LastError = lastErr.String
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
