package user

// 账号列表查询(基础配置 · 账号与角色页数据源,列名对照 docs/admin/account.html)。

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// AccountRow 账号列表行(不含密码哈希;status 1启用 0停用;regionScope 空=全集团;组织 ID 0=不限)。
type AccountRow struct {
	ID              int64  `json:"id"`
	Username        string `json:"username"`
	RealName        string `json:"realName"`
	Phone           string `json:"phone"`
	StaffNo         string `json:"staffNo"` // 工号(000173,空=未编)
	RoleCode        string `json:"roleCode"`
	RoleName        string `json:"roleName"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	DeptID          int64  `json:"deptId"`
	DeptName        string `json:"deptName"`
	PostID          int64  `json:"postId"`
	PostName        string `json:"postName"`
	RegionScope     string `json:"regionScope"`
	Status          int16  `json:"status"`
}

// accountCols 账号列表统一列(ID+名称成对;name 为 COALESCE 空串、id 为 COALESCE 0=不限)。
const accountCols = `a.id, a.username, a.real_name, COALESCE(a.phone,''), COALESCE(a.staff_no,''),
		       r.code, r.name,
		       COALESCE(a.legal_entity_id,0), COALESCE(le.name,''),
		       COALESCE(a.dept_id,0), COALESCE(d.name,''),
		       COALESCE(a.post_id,0), COALESCE(p.name,''),
		       COALESCE(a.region_scope::text,''), a.status`

// scanAccount 按 accountCols 顺序扫描一行。
func scanAccount(rows pgx.Rows) (AccountRow, error) {
	var r AccountRow
	if err := rows.Scan(&r.ID, &r.Username, &r.RealName, &r.Phone, &r.StaffNo, &r.RoleCode, &r.RoleName,
		&r.LegalEntityID, &r.LegalEntityName, &r.DeptID, &r.DeptName, &r.PostID, &r.PostName,
		&r.RegionScope, &r.Status); err != nil {
		return r, fmt.Errorf("user: scan account: %w", err)
	}
	return r, nil
}

// ListAccounts 全量账号(按 id 升序;账号量级小,不分页)。
func (s *PGStore) ListAccounts(ctx context.Context) ([]AccountRow, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+accountCols+`
		FROM accounts a
		JOIN roles r ON r.id = a.role_id
		LEFT JOIN legal_entities le ON le.id = a.legal_entity_id
		LEFT JOIN departments d ON d.id = a.dept_id
		LEFT JOIN posts p ON p.id = a.post_id
		ORDER BY a.id`)
	if err != nil {
		return nil, fmt.Errorf("user: list accounts: %w", err)
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

// ListDataScopes 账号数据范围清单(kw 模糊匹配账号/姓名/角色名;空=全量)。
func (s *PGStore) ListDataScopes(ctx context.Context, kw string) ([]AccountRow, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+accountCols+`
		FROM accounts a
		JOIN roles r ON r.id = a.role_id
		LEFT JOIN legal_entities le ON le.id = a.legal_entity_id
		LEFT JOIN departments d ON d.id = a.dept_id
		LEFT JOIN posts p ON p.id = a.post_id
		WHERE ($1 = '' OR a.username ILIKE '%' || $1 || '%'
		       OR a.real_name ILIKE '%' || $1 || '%' OR r.name ILIKE '%' || $1 || '%')
		ORDER BY a.id`, kw)
	if err != nil {
		return nil, fmt.Errorf("user: list data scopes: %w", err)
	}
	defer rows.Close()
	out := make([]AccountRow, 0)
	for rows.Next() {
		r, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("user: scan data scope: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
