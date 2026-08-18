// 注册(SYS 域,阶段1):创建 ops 角色账号,注册后即可登录。
package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// ErrUsernameTaken 登录名已存在。
var ErrUsernameTaken = errors.New("user: username taken")

// ErrInvalidInput 注册入参不合法(长度/字符集)。
var ErrInvalidInput = errors.New("user: invalid input")

// RegisterRules 注册入参约束,与前端校验保持一致。
const (
	UsernameMin = 3
	UsernameMax = 64
	PasswordMin = 6
	RealNameMax = 64
)

// Register 自助注册:默认角色 ops(业务运营),状态启用;成功即返回可登录身份。
func (s *PGStore) Register(ctx context.Context, username, password, realName string) (*LoginResult, error) {
	if err := validateRegister(username, password, realName); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("user: bcrypt: %w", err)
	}
	var id int64
	var roleName string
	err = s.db.QueryRow(ctx, `
		INSERT INTO accounts (username, password_hash, real_name, role_id)
		SELECT $1, $2, $3, r.id FROM roles r WHERE r.code = 'ops'
		RETURNING accounts.id, (SELECT name FROM roles WHERE code = 'ops')`,
		username, string(hash), realName).Scan(&id, &roleName)
	if isUniqueViolation(err) {
		return nil, ErrUsernameTaken
	}
	if err != nil {
		return nil, fmt.Errorf("user: register insert: %w", err)
	}
	return &LoginResult{AccountID: id, Username: username, RealName: realName, RoleCode: "ops", RoleName: roleName}, nil
}

func validateRegister(username, password, realName string) error {
	u, p, n := utf8.RuneCountInString(username), utf8.RuneCountInString(password), utf8.RuneCountInString(realName)
	switch {
	case u < UsernameMin || u > UsernameMax || !isLoginName(username):
		return ErrInvalidInput
	case p < PasswordMin:
		return ErrInvalidInput
	case n == 0 || n > RealNameMax:
		return ErrInvalidInput
	}
	return nil
}

// isLoginName 限定字母/数字/下划线/中划线/点。
func isLoginName(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "username")
}
