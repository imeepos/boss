// 账号管理写操作(SYS 域,阶段1):封闭模型下的受权建号/改号。
// 建号=上级分配(角色+组织归属+数据范围),与已移除的自助注册不同源。
package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// AccountInput 建号/改号入参;指针字段 nil=修改时保持不变。
type AccountInput struct {
	Username      string  `json:"username"`
	Password      string  `json:"password"` // 新建必填;修改留空=不改
	RealName      string  `json:"realName"`
	Phone         string  `json:"phone"`
	StaffNo       string  `json:"staffNo"` // 工号(000173,可空;企业员工登录标识)
	RoleCode      string  `json:"roleCode"`
	LegalEntityID *int64  `json:"legalEntityId"` // nil/0=NULL(不限)
	DeptID        *int64  `json:"deptId"`
	PostID        *int64  `json:"postId"`
	RegionScope   *string `json:"regionScope"` // nil=不修改;""=NULL(全集团)
	Status        *int16  `json:"status"`      // nil=不修改;1启用 0停用
}

// Role 角色行(fields.md 1.2,7 角色码)。
type Role struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var ErrRoleNotFound = errors.New("user: role not found")

// CreateAccount 受权建号:角色码须存在;组织归属/数据范围可空(=全集团)。
func (s *PGStore) CreateAccount(ctx context.Context, in AccountInput) (int64, error) {
	if err := validateRegister(in.Username, in.Password, in.RealName); err != nil {
		return 0, err
	}
	if err := validatePhone(in.Phone); err != nil {
		return 0, err
	}
	if err := validateStaffNo(in.StaffNo); err != nil {
		return 0, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("user: bcrypt: %w", err)
	}
	var id int64
	err = s.db.QueryRow(ctx, `
		INSERT INTO accounts (username, password_hash, real_name, phone, staff_no, role_id,
		                      legal_entity_id, dept_id, post_id, region_scope, status)
		SELECT $1, $2, $3, $4, NULLIF($5,''), r.id, NULLIF($6,0), NULLIF($7,0), NULLIF($8,0),
		       NULLIF($9,'')::ltree, 1
		FROM roles r WHERE r.code = $10
		RETURNING accounts.id`,
		in.Username, string(hash), in.RealName, in.Phone, strings.TrimSpace(in.StaffNo),
		derefInt64(in.LegalEntityID), derefInt64(in.DeptID), derefInt64(in.PostID),
		derefString(in.RegionScope), in.RoleCode).Scan(&id)
	if isStaffNoUniqueViolation(err) {
		return 0, ErrStaffNoTaken
	}
	if isUniqueViolation(err) {
		return 0, ErrUsernameTaken
	}
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return 0, ErrRoleNotFound
		}
		return 0, fmt.Errorf("user: create account: %w", err)
	}
	return id, nil
}

// UpdateAccount 受权改号:密码留空不改;nil 字段保持不变;未命中返回 ErrNotFound。
func (s *PGStore) UpdateAccount(ctx context.Context, id int64, in AccountInput) error {
	if in.Username == "" || strings.TrimSpace(in.RealName) == "" {
		return ErrInvalidInput
	}
	if err := validatePhone(in.Phone); err != nil {
		return err
	}
	if err := validateStaffNo(in.StaffNo); err != nil {
		return err
	}
	var roleID int64
	err := s.db.QueryRow(ctx, `SELECT id FROM roles WHERE code=$1`, in.RoleCode).Scan(&roleID)
	if err != nil {
		return ErrRoleNotFound
	}
	sets := []string{"username=$2", "real_name=$3", "phone=$4", "staff_no=NULLIF($5,'')", "role_id=$6"}
	args := []any{id, in.Username, in.RealName, in.Phone, strings.TrimSpace(in.StaffNo), roleID}
	next := 7
	appendSet := func(sql string, v any) { sets = append(sets, fmt.Sprintf(sql, next)); args = append(args, v); next++ }
	appendSet("legal_entity_id=NULLIF($%d,0)", derefInt64(in.LegalEntityID))
	appendSet("dept_id=NULLIF($%d,0)", derefInt64(in.DeptID))
	appendSet("post_id=NULLIF($%d,0)", derefInt64(in.PostID))
	appendSet("region_scope=NULLIF($%d,'')::ltree", derefString(in.RegionScope))
	if in.Status != nil {
		appendSet("status=$%d", *in.Status)
	}
	if in.Password != "" {
		if len([]rune(in.Password)) < PasswordMin {
			return ErrInvalidInput
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("user: bcrypt: %w", err)
		}
		appendSet("password_hash=$%d", string(hash))
	}
	tag, err := s.db.Exec(ctx,
		"UPDATE accounts SET "+strings.Join(sets, ", ")+" WHERE id=$1", args...)
	if isStaffNoUniqueViolation(err) {
		return ErrStaffNoTaken
	}
	if isUniqueViolation(err) {
		return ErrUsernameTaken
	}
	if err != nil {
		return fmt.Errorf("user: update account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListRoles 角色清单(code/name,fields.md 1.2)。
func (s *PGStore) ListRoles(ctx context.Context) ([]Role, error) {
	cols, err := s.listRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Role, len(cols))
	for i, c := range cols {
		out[i] = Role{Code: c.RoleCode, Name: c.RoleName}
	}
	return out, nil
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// validatePhone 手机号可选;填了则 3-32 位数字/加减空格。
func validatePhone(phone string) error {
	n := len(phone)
	if n == 0 {
		return nil
	}
	if n < 3 || n > 32 {
		return ErrInvalidInput
	}
	for _, r := range phone {
		switch {
		case r >= '0' && r <= '9', r == '+', r == '-', r == ' ':
		default:
			return ErrInvalidInput
		}
	}
	return nil
}
