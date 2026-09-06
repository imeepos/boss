package aaa

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// 在线会话状态控制面:强制下线迁移/重试计数/僵尸扫描(PGStore 实现 SessionControlRepo)。

const sessionCols = "id, loid, session_id, nas_ip, started_at, last_update, input_octets, output_octets, status, disconnect_attempts, close_reason, closed_at"

// scanSession 单行扫描;closed_at 允许 NULL。
func scanSession(scan func(dest ...any) error) (SessionRecord, error) {
	var r SessionRecord
	if err := scan(&r.ID, &r.Loid, &r.SessionID, &r.NasIP, &r.StartedAt, &r.LastUpdate,
		&r.InputOctets, &r.OutputOctets, &r.Status, &r.DisconnectAttempts, &r.CloseReason, &r.ClosedAt); err != nil {
		return SessionRecord{}, err
	}
	return r, nil
}

// GetSessionByID 单条会话;未命中 ErrSessionNotFound。
func (s *PGStore) GetSessionByID(ctx context.Context, id int64) (*SessionRecord, error) {
	r, err := scanSession(func(dest ...any) error {
		return s.db.QueryRow(ctx, `SELECT `+sessionCols+` FROM aaa_online_sessions WHERE id = $1`, id).Scan(dest...)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("aaa: get session: %w", err)
	}
	return &r, nil
}

// ListOnlineSessions 指定 LOID 的全部 ONLINE 会话(强制下线目标集,started_at 序)。
func (s *PGStore) ListOnlineSessions(ctx context.Context, loid string) ([]SessionRecord, error) {
	return s.querySessions(ctx, 
		`SELECT `+sessionCols+` FROM aaa_online_sessions WHERE loid = $1 AND status = $2 ORDER BY id`,
		loid, SessionOnline)
}

// ListPendingOfflineSessions 全部 PENDING_OFFLINE(重试扫描集,含已达上限的兜底转失败)。
func (s *PGStore) ListPendingOfflineSessions(ctx context.Context, limit int) ([]SessionRecord, error) {
	return s.querySessions(ctx, 
		`SELECT `+sessionCols+` FROM aaa_online_sessions WHERE status = $1 ORDER BY id LIMIT $2`,
		SessionPendingOffline, limit)
}

// ListStaleOnlineSessions last_update 早于 staleBefore 的 ONLINE 会话(僵尸扫描集)。
func (s *PGStore) ListStaleOnlineSessions(ctx context.Context, staleBefore time.Time, limit int) ([]SessionRecord, error) {
	return s.querySessions(ctx, 
		`SELECT `+sessionCols+` FROM aaa_online_sessions WHERE status = $1 AND last_update < $2 ORDER BY id LIMIT $3`,
		SessionOnline, staleBefore, limit)
}

// querySessions 会话列表通用查询。
func (s *PGStore) querySessions(ctx context.Context, sql string, args ...any) ([]SessionRecord, error) {
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("aaa: query sessions: %w", err)
	}
	defer rows.Close()
	out := make([]SessionRecord, 0)
	for rows.Next() {
		r, err := scanSession(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("aaa: scan session: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MarkSessionPendingOffline ONLINE→PENDING_OFFLINE(下发失败等重试);并发已迁移时 no-op。
func (s *PGStore) MarkSessionPendingOffline(ctx context.Context, id int64) error {
	return s.transitSession(ctx, id, SessionPendingOffline, "", []string{SessionOnline})
}

// MarkSessionOffline 非终态→OFFLINE(计账 Stop 之外的 CoA 成功路径),带关闭原因。
func (s *PGStore) MarkSessionOffline(ctx context.Context, id int64, closeReason string) error {
	return s.transitSession(ctx, id, SessionOffline, closeReason, []string{SessionOnline, SessionPendingOffline})
}

// MarkSessionOfflineFailed PENDING_OFFLINE→OFFLINE_FAILED(重试耗尽),带关闭原因。
func (s *PGStore) MarkSessionOfflineFailed(ctx context.Context, id int64, closeReason string) error {
	return s.transitSession(ctx, id, SessionOfflineFailed, closeReason, []string{SessionPendingOffline})
}

// transitSession 状态迁移;0 行属并发竞态(已被 Stop/重试转移),非错误。
func (s *PGStore) transitSession(ctx context.Context, id int64, to, closeReason string, from []string) error {
	tag, err := s.db.Exec(ctx, 
		`UPDATE aaa_online_sessions SET status = $2, close_reason = $3, closed_at = now()
		WHERE id = $1 AND status = ANY($4)`,
		id, to, closeReason, from)
	if err != nil {
		return fmt.Errorf("aaa: transit session %d to %s: %w", id, to, err)
	}
	_ = tag
	return nil
}

// IncrementSessionAttempts 重试计数+1,返回新值。
func (s *PGStore) IncrementSessionAttempts(ctx context.Context, id int64) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, 
		`UPDATE aaa_online_sessions SET disconnect_attempts = disconnect_attempts + 1 WHERE id = $1 RETURNING disconnect_attempts`,
		id).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("aaa: bump session attempts: %w", err)
	}
	return n, nil
}

// ReapSessionZombie 僵尸关闭:ONLINE→OFFLINE(原因 ZOMBIE_REAP);返回是否本调用关闭(幂等)。
func (s *PGStore) ReapSessionZombie(ctx context.Context, id int64) (bool, error) {
	tag, err := s.db.Exec(ctx, 
		`UPDATE aaa_online_sessions SET status = $2, close_reason = $3, closed_at = now()
		WHERE id = $1 AND status = $4`,
		id, SessionOffline, CloseReasonZombie, SessionOnline)
	if err != nil {
		return false, fmt.Errorf("aaa: reap session: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}