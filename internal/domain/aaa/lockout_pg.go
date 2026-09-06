package aaa

// A1 防爆破锁定存储(同 LOID 连续失败计数 + 窗口锁定)。
// 独立表 lo_auth_lockouts:认证热路径单行读写,不污染 lo_accounts 主档;
// 锁定期间一律拒绝(不再累计);窗口到期由下一次认证检查清行,计数从零重开。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// checkLock 锁定中 → ErrLocked;窗口已到期 → 清行自动解锁。
func (a *CredentialAuthorizer) checkLock(ctx context.Context, loid string) error {
	var failCount int
	var lockedUntil time.Time
	err := a.db.QueryRow(ctx, `SELECT fail_count, COALESCE(locked_until, to_timestamp(0)) FROM lo_auth_lockouts WHERE loid = $1`, loid).Scan(&failCount, &lockedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("aaa: check lock: %w", err)
	}
	if lockedUntil.After(a.cfg.Now()) {
		return ErrLocked
	}
	if lockedUntil.Unix() > 0 { // 到期解锁(排除 NULL 哨兵 epoch)
		return a.clearLock(ctx, loid)
	}
	return nil
}

// recordFail 失败计数 +1;第 N(阈值)次置 locked_until=now+窗口(SQL 内原子判定)。
func (a *CredentialAuthorizer) recordFail(ctx context.Context, loid string) error {
	lockAt := a.cfg.Now().Add(a.cfg.LockWindow)
	_, err := a.db.Exec(ctx, `INSERT INTO lo_auth_lockouts(loid, fail_count, locked_until, updated_at)
VALUES($1, 1, CASE WHEN 1 >= $2 THEN $3 ELSE NULL END, now())
ON CONFLICT (loid) DO UPDATE SET
fail_count = lo_auth_lockouts.fail_count + 1,
locked_until = CASE WHEN lo_auth_lockouts.fail_count + 1 >= $2 THEN $3
ELSE lo_auth_lockouts.locked_until END,
updated_at = now()`, loid, a.cfg.LockThreshold, lockAt)
	if err != nil {
		return fmt.Errorf("aaa: record auth fail: %w", err)
	}
	return nil
}

// clearLock 清计数(认证成功/窗口到期/密码重置)。
func (a *CredentialAuthorizer) clearLock(ctx context.Context, loid string) error {
	if _, err := a.db.Exec(ctx, `DELETE FROM lo_auth_lockouts WHERE loid = $1`, loid); err != nil {
		return fmt.Errorf("aaa: clear lock: %w", err)
	}
	return nil
}
