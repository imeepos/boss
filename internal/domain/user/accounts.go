package user

// 账号列表查询(基础配置 · 账号与角色页数据源,列名对照 docs/admin/account.html)。

import (
	"context"
	"fmt"
)

// AccountRow 账号列表行(不含密码哈希;status 1启用 0停用;regionScope 空=全集团)。
type AccountRow struct {
	ID              int64  `json:"id"`
	Username        string `json:"username"`
	RealName        string `json:"realName"`
	Phone           string `json:"phone"`
	RoleCode        string `json:"roleCode"`
	RoleName        string `json:"roleName"`
	LegalEntityName string `json:"legalEntityName"`
	DeptName        string `json:"deptName"`
	PostName        string `json:"postName"`
	RegionScope     string `json:"regionScope"`
	Status          int16  `json:"status"`
}

// ListAccounts 全量账号(按 id 升序;账号量级小,不分页)。
func (s *PGStore) ListAccounts(ctx context.Context) ([]AccountRow, error) {
	rows, err := s.db.Query(ctx, `
		SELECT a.id, a.username, a.real_name, COALESCE(a.phone,''),
		       r.code, r.name,
		       COALESCE(le.name,''), COALESCE(d.name,''), COALESCE(p.name,''),
		       COALESCE(a.region_scope::text,''), a.status
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
		var r AccountRow
		if err := rows.Scan(&r.ID, &r.Username, &r.RealName, &r.Phone, &r.RoleCode, &r.RoleName,
			&r.LegalEntityName, &r.DeptName, &r.PostName, &r.RegionScope, &r.Status); err != nil {
			return nil, fmt.Errorf("user: scan account: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
