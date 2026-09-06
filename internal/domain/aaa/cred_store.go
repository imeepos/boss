package aaa

// A1:LOID 凭据管理入口(管理端重置;随机口令,明文仅一次性出现在返回值,库内只有密文)。

import (
	"context"
	"errors"
	"fmt"

	"github.com/ymm-001/boss/internal/domain/aaa/credential"
)

// SetLoCredential 写入已编码凭据;loid 不存在返回 ErrNotFound。
func (s *PGStore) SetLoCredential(ctx context.Context, loid, encoded string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE lo_accounts SET password_credential = $2 WHERE loid = $1`, loid, encoded)
	if err != nil {
		return fmt.Errorf("aaa: set credential: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ResetLoPassword 重置指定 LOID 密码:随机生成 → 编码落库 → 清防爆破计数
// (新密码应从零计) → 一次性返回明文。存量凭据直接覆盖(admin 权威操作,不校验旧密码)。
func (s *PGStore) ResetLoPassword(ctx context.Context, loid string) (string, error) {
	if s.cred == nil {
		return "", errors.New("aaa: credential codec not configured")
	}
	pw, err := credential.Random(16)
	if err != nil {
		return "", err
	}
	encoded, err := s.cred.Encode(pw)
	if err != nil {
		return "", fmt.Errorf("aaa: encode credential: %w", err)
	}
	if err := s.SetLoCredential(ctx, loid, encoded); err != nil {
		return "", err
	}
	if _, err := s.db.Exec(ctx, `DELETE FROM lo_auth_lockouts WHERE loid = $1`, loid); err != nil {
		return "", fmt.Errorf("aaa: clear lock after reset: %w", err)
	}
	return pw, nil
}
