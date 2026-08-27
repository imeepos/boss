package openplat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// dbtx 是 PGStore 依赖的最小数据库接口。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 Service 的 PostgreSQL 实现。
type PGStore struct{ db dbtx }

// NewPGStore 构造 PGStore。
func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

const appCols = `id, app_id, name, status, rate_limit_rpm, daily_quota, sandbox, last_used_at, created_at`

// CreateApp 创建应用;AppID 形如 op_<16hex>,Secret 形如 ops_<32hex>,均仅生成一次。
func (s *PGStore) CreateApp(ctx context.Context, name string, rpm, quota int, sandbox bool, createdBy int64) (*AppCreateResult, error) {
	if rpm <= 0 {
		rpm = 60
	}
	if quota <= 0 {
		quota = 10000
	}
	appID, secret, err := newCredentials()
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.db.QueryRow(ctx, `
		INSERT INTO open_apps (app_id, secret, name, rate_limit_rpm, daily_quota, sandbox, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		appID, secret, name, rpm, quota, sandbox, createdBy).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("openplat: create app: %w", err)
	}
	return &AppCreateResult{
		App:    App{ID: id, AppID: appID, Name: name, Status: 1, RateLimitRPM: rpm, DailyQuota: quota, Sandbox: sandbox},
		Secret: secret,
	}, nil
}

// ListApps 列出应用元数据(不含 Secret)。
func (s *PGStore) ListApps(ctx context.Context) ([]App, error) {
	rows, err := s.db.Query(ctx, `SELECT `+appCols+` FROM open_apps ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("openplat: list apps: %w", err)
	}
	defer rows.Close()
	out := make([]App, 0)
	for rows.Next() {
		var a App
		var lastUsed, createdAt pgtype.Timestamptz
		if err := rows.Scan(&a.ID, &a.AppID, &a.Name, &a.Status, &a.RateLimitRPM, &a.DailyQuota,
			&a.Sandbox, &lastUsed, &createdAt); err != nil {
			return nil, fmt.Errorf("openplat: scan app: %w", err)
		}
		a.LastUsedAt = fmtTime(lastUsed)
		a.CreatedAt = fmtTime(createdAt)
		out = append(out, a)
	}
	return out, rows.Err()
}

// SetAppStatus 启用/停用应用。
func (s *PGStore) SetAppStatus(ctx context.Context, id int64, status int16) error {
	tag, err := s.db.Exec(ctx, `UPDATE open_apps SET status=$2 WHERE id=$1`, id, status)
	if err != nil {
		return fmt.Errorf("openplat: set status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// LookupActive 按公开 AppID 取启用中应用的验签上下文(含沙箱标记)。
func (s *PGStore) LookupActive(ctx context.Context, appID string) (*AuthContext, error) {
	var a AuthContext
	err := s.db.QueryRow(ctx, `
		SELECT id, secret, rate_limit_rpm, daily_quota, sandbox FROM open_apps
		WHERE app_id = $1 AND status = 1`, appID).Scan(&a.AppRowID, &a.Secret, &a.RateLimitRPM, &a.DailyQuota, &a.Sandbox)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("openplat: lookup: %w", err)
	}
	return &a, nil
}

// TouchUsage 记录一次调用:last_used_at 刷新 + 当日计数自增,返回当日累计。
func (s *PGStore) TouchUsage(ctx context.Context, id int64, day time.Time) (int64, error) {
	var count int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO open_usage_day (app_id, day, call_count) VALUES ($1, $2, 1)
		ON CONFLICT (app_id, day) DO UPDATE SET call_count = open_usage_day.call_count + 1
		RETURNING call_count`, id, day).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("openplat: touch usage: %w", err)
	}
	_, _ = s.db.Exec(ctx, `UPDATE open_apps SET last_used_at = now() WHERE id = $1`, id)
	return count, nil
}

// CreateSubscription 新增事件订阅(同 app+事件+端点唯一)。
func (s *PGStore) CreateSubscription(ctx context.Context, appID int64, eventType, endpointURL string) (*Subscription, error) {
	var id int64
	var createdAt pgtype.Timestamptz
	err := s.db.QueryRow(ctx, `
		INSERT INTO open_webhook_subscriptions (app_id, event_type, endpoint_url)
		VALUES ($1, $2, $3) RETURNING id, created_at`, appID, eventType, endpointURL).Scan(&id, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("openplat: create subscription: %w", err)
	}
	return &Subscription{ID: id, AppID: appID, EventType: eventType, EndpointURL: endpointURL, Status: 1, CreatedAt: fmtTime(createdAt)}, nil
}

// CreateSubscriptions 一个端点批量订阅多个事件:单条 INSERT..SELECT unnest 原子写入,
// 唯一约束 (app_id,event_type,endpoint_url) 让重复订阅幂等跳过;返回该端点命中的订阅行。
func (s *PGStore) CreateSubscriptions(ctx context.Context, appID int64, eventTypes []string, endpointURL string) ([]Subscription, error) {
	if len(eventTypes) == 0 {
		return nil, nil
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO open_webhook_subscriptions (app_id, event_type, endpoint_url)
		SELECT $1, t, $2 FROM unnest($3::text[]) t
		ON CONFLICT (app_id, event_type, endpoint_url) DO NOTHING`, appID, endpointURL, eventTypes)
	if err != nil {
		return nil, fmt.Errorf("openplat: create subscriptions: %w", err)
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, app_id, event_type, endpoint_url, status, created_at
		FROM open_webhook_subscriptions
		WHERE app_id = $1 AND endpoint_url = $2 AND event_type = ANY($3)
		ORDER BY id`, appID, endpointURL, eventTypes)
	if err != nil {
		return nil, fmt.Errorf("openplat: list created subscriptions: %w", err)
	}
	defer rows.Close()
	out := make([]Subscription, 0, len(eventTypes))
	for rows.Next() {
		var sub Subscription
		var createdAt pgtype.Timestamptz
		if err := rows.Scan(&sub.ID, &sub.AppID, &sub.EventType, &sub.EndpointURL, &sub.Status, &createdAt); err != nil {
			return nil, fmt.Errorf("openplat: scan created subscription: %w", err)
		}
		sub.CreatedAt = fmtTime(createdAt)
		out = append(out, sub)
	}
	return out, rows.Err()
}

// ListSubscriptions 列出应用的事件订阅。
func (s *PGStore) ListSubscriptions(ctx context.Context, appID int64) ([]Subscription, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, app_id, event_type, endpoint_url, status, created_at
		FROM open_webhook_subscriptions WHERE app_id = $1 ORDER BY id`, appID)
	if err != nil {
		return nil, fmt.Errorf("openplat: list subscriptions: %w", err)
	}
	defer rows.Close()
	out := make([]Subscription, 0)
	for rows.Next() {
		var sub Subscription
		var createdAt pgtype.Timestamptz
		if err := rows.Scan(&sub.ID, &sub.AppID, &sub.EventType, &sub.EndpointURL, &sub.Status, &createdAt); err != nil {
			return nil, fmt.Errorf("openplat: scan subscription: %w", err)
		}
		sub.CreatedAt = fmtTime(createdAt)
		out = append(out, sub)
	}
	return out, rows.Err()
}

// DeleteSubscription 删除订阅(硬删,唯一约束防重复建)。
func (s *PGStore) DeleteSubscription(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM open_webhook_subscriptions WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("openplat: delete subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// newCredentials 生成 AppID(op_<16hex>)与 Secret(ops_<32hex>)。
func newCredentials() (appID, secret string, err error) {
	b1, b2 := make([]byte, 8), make([]byte, 16)
	if _, err = rand.Read(b1); err != nil {
		return "", "", fmt.Errorf("openplat: rand: %w", err)
	}
	if _, err = rand.Read(b2); err != nil {
		return "", "", fmt.Errorf("openplat: rand: %w", err)
	}
	return "op_" + hex.EncodeToString(b1), "ops_" + hex.EncodeToString(b2), nil
}

func fmtTime(t pgtype.Timestamptz) string {
	if t.Valid {
		return t.Time.Format(time.RFC3339)
	}
	return ""
}
