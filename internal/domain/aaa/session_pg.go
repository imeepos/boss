package aaa

import (
	"context"
	"fmt"
)

// 在线会话计账维护面:Start 幂等建 / Interim 累加 / Stop 关闭(PGStore 实现 SessionMaintainer)。

// StartSession 建 ONLINE 会话;(loid, session_id) 唯一索引兜底,重复 Start 幂等去重。
func (s *PGStore) StartSession(ctx context.Context, rec SessionRecord) (bool, error) {
	tag, err := s.db.Exec(ctx,
		`INSERT INTO aaa_online_sessions(loid, session_id, nas_ip, input_octets, output_octets)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT (loid, session_id) DO NOTHING`,
		rec.Loid, rec.SessionID, rec.NasIP, rec.InputOctets, rec.OutputOctets)
	if err != nil {
		return false, fmt.Errorf("aaa: start session: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// TouchSessionTraffic Interim:累加流量并刷新 last_update;仅 ONLINE 生效,会话不存在为 no-op
// (孤儿 Interim 不影响计账响应,未知会话由 RowsAffected=0 静默跳过属协议语义而非吞错)。
func (s *PGStore) TouchSessionTraffic(ctx context.Context, loid, sessionID string, inputOctets, outputOctets int64) error {
	_, err := s.db.Exec(ctx,
		`UPDATE aaa_online_sessions
		SET input_octets = input_octets + $3, output_octets = output_octets + $4, last_update = now()
		WHERE loid = $1 AND session_id = $2 AND status = $5`,
		loid, sessionID, inputOctets, outputOctets, SessionOnline)
	if err != nil {
		return fmt.Errorf("aaa: touch session: %w", err)
	}
	return nil
}

// StopSession 计账 Stop:ONLINE/PENDING_OFFLINE → OFFLINE;返回会话是否存在(孤儿 Stop 不报错)。
func (s *PGStore) StopSession(ctx context.Context, loid, sessionID, closeReason string) (bool, error) {
	tag, err := s.db.Exec(ctx,
		`UPDATE aaa_online_sessions
		SET status = $3, close_reason = $4, closed_at = now()
		WHERE loid = $1 AND session_id = $2 AND status = ANY($5)`,
		loid, sessionID, SessionOffline, closeReason, []string{SessionOnline, SessionPendingOffline})
	if err != nil {
		return false, fmt.Errorf("aaa: stop session: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// CountActiveSessions 在线占用数(ONLINE+PENDING_OFFLINE:PENDING 仍占 NAS 会话资源,计入上限)。
func (s *PGStore) CountActiveSessions(ctx context.Context, loid string) (int, error) {
	var n int
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM aaa_online_sessions WHERE loid = $1 AND status = ANY($2)`,
		loid, []string{SessionOnline, SessionPendingOffline}).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("aaa: count active sessions: %w", err)
	}
	return n, nil
}

// AllowNewSession 并发闸口:在线占用数 < limit 放行(PGStore 实现 SessionGate)。
func (s *PGStore) AllowNewSession(ctx context.Context, loid string, limit int) (bool, int, error) {
	n, err := s.CountActiveSessions(ctx, loid)
	if err != nil {
		return false, 0, err
	}
	return n < limit, n, nil
}
