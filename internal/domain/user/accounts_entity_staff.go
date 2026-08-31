package user

// 企业员工后台录入(CH 渠道经销商域,000172):管理后台按企业维度维护员工登录信息。
// 企业员工=accounts 绑定 legal_entity_id + partner_admin/partner_staff 角色(fields.md 8C);
// 与企业工作台自助建号(partner.CreateStaff)平行,本文件是后台受权视角,可编工号/建管理员。

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// EntityStaffRoles 企业员工角色白名单:后台录入仅允许入驻企业两角色,不得借道授出平台角色。
var EntityStaffRoles = map[string]bool{"partner_admin": true, "partner_staff": true}

// ListEntityStaff 某企业的员工账号(partner_* 角色,id 升序)。
func (s *PGStore) ListEntityStaff(ctx context.Context, entityID int64) ([]AccountRow, error) {
	if entityID <= 0 {
		return nil, ErrInvalidInput
	}
	rows, err := s.db.Query(ctx, `
		SELECT `+accountCols+`
		FROM accounts a
		JOIN roles r ON r.id = a.role_id
		LEFT JOIN legal_entities le ON le.id = a.legal_entity_id
		LEFT JOIN departments d ON d.id = a.dept_id
		LEFT JOIN posts p ON p.id = a.post_id
		WHERE a.legal_entity_id = $1 AND r.code IN ('partner_admin','partner_staff')
		ORDER BY a.id`, entityID)
	if err != nil {
		return nil, fmt.Errorf("user: list entity staff: %w", err)
	}
	defer rows.Close()
	out := make([]AccountRow, 0)
	for rows.Next() {
		r, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateEntityStaff 后台录入企业员工:强制归属该企业,角色限白名单;工号冲突 40900。
func (s *PGStore) CreateEntityStaff(ctx context.Context, entityID int64, in AccountInput) (int64, error) {
	if entityID <= 0 {
		return 0, ErrInvalidInput
	}
	if !EntityStaffRoles[in.RoleCode] {
		return 0, fmt.Errorf("user: role %q not allowed for entity staff: %w", in.RoleCode, ErrInvalidInput)
	}
	in.LegalEntityID = &entityID // 封闭归属:忽略入参,一律绑路径上的企业
	return s.CreateAccount(ctx, in)
}

// SetEntityStaffPassword 重置企业员工登录密码(企业边界校验,越界 ErrNotFound)。
func (s *PGStore) SetEntityStaffPassword(ctx context.Context, entityID, accountID int64, password string) error {
	if len([]rune(password)) < PasswordMin {
		return ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("user: bcrypt: %w", err)
	}
	return s.execEntityStaff(ctx, entityID, accountID, `password_hash=$3`, hash, "set staff password")
}

// SetEntityStaffStatus 启用/停用企业员工(企业边界校验;后台为受权操作,无防自锁限制)。
func (s *PGStore) SetEntityStaffStatus(ctx context.Context, entityID, accountID int64, status int16) error {
	if status != 0 && status != 1 {
		return ErrInvalidInput
	}
	return s.execEntityStaff(ctx, entityID, accountID, `status=$3`, status, "set staff status")
}

// execEntityStaff 企业边界内的定点更新(仅命中该企业的 partner_* 账号;未命中 ErrNotFound)。
func (s *PGStore) execEntityStaff(ctx context.Context, entityID, accountID int64, set string, val any, op string) error {
	if entityID <= 0 || accountID <= 0 {
		return ErrInvalidInput
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE accounts a SET `+set+`, updated_at=now()
		FROM roles r
		WHERE a.id=$2 AND a.role_id=r.id AND a.legal_entity_id=$1
		  AND r.code IN ('partner_admin','partner_staff')`,
		entityID, accountID, val)
	if err != nil {
		return fmt.Errorf("user: %s: %w", op, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
