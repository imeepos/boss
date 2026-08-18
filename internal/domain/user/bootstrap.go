// 超管引导(SYS 域,阶段1):服务启动时幂等初始化超级管理员,管理最顶层信息。
// 原则:密码不入迁移/不入种子文件;已存在则跳过,绝不覆盖(防重启重置密码)。
package user

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// ErrBootstrapPassword 弱口令:长度低于 PasswordMin。
var ErrBootstrapPassword = errors.New("user: bootstrap password too short")

// EnsureSuperAdmin 幂等创建超级管理员(sysadmin 角色,状态启用)。
// 仅当 username 不存在时插入;返回 created 表示本次是否新建。
func (s *PGStore) EnsureSuperAdmin(ctx context.Context, username, password, realName string) (created bool, err error) {
	if err := validateRegister(username, password, realName); err != nil {
		return false, ErrBootstrapPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false, fmt.Errorf("user: bcrypt: %w", err)
	}
	tag, err := s.db.Exec(ctx, `
		INSERT INTO accounts (username, password_hash, real_name, role_id, status)
		SELECT $1, $2, $3, r.id, 1 FROM roles r WHERE r.code = 'sysadmin'
		ON CONFLICT (username) DO NOTHING`,
		username, string(hash), realName)
	if err != nil {
		return false, fmt.Errorf("user: bootstrap super admin: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
