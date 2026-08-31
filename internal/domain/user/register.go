// 账号入参约束(SYS 域,阶段1):超管引导(EnsureSuperAdmin)与账号管理共用。
// admin 端为封闭账号模型:自助注册已移除(契约裁定见 domain-map.md),账号仅经 org/account 受权流程创建。
package user

import (
	"errors"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrUsernameTaken 登录名已存在。
var ErrUsernameTaken = errors.New("user: username taken")

// ErrStaffNoTaken 工号已存在(accounts.staff_no,000172)。
var ErrStaffNoTaken = errors.New("user: staff no taken")

// ErrInvalidInput 入参不合法(长度/字符集)。
var ErrInvalidInput = errors.New("user: invalid input")

// AccountRules 账号入参约束,与前端校验保持一致。
const (
	UsernameMin = 3
	UsernameMax = 64
	PasswordMin = 6
	RealNameMax = 64
	StaffNoMax  = 32
)

// validateRegister 账号三项(登录名/密码/姓名)基础校验,引导建号复用。
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

// validateStaffNo 工号可选;填了则 ≤StaffNoMax 且字符集同登录名(对标 workers.staff_no)。
func validateStaffNo(staffNo string) error {
	n := utf8.RuneCountInString(staffNo)
	if n == 0 {
		return nil
	}
	if n > StaffNoMax || !isLoginName(staffNo) {
		return ErrInvalidInput
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	// 任意唯一约束冲突均视为冲突(账号表唯一键=username/staff_no,部门/岗位为复合唯一键)。
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isStaffNoUniqueViolation 工号唯一索引(uq_accounts_staff_no)冲突,与登录名冲突区分提示。
func isStaffNoUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uq_accounts_staff_no"
}
