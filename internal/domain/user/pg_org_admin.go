package user

import (
	"context"
	"fmt"
)

// normalizeTaxChannel 属地配置归一化:channel 空=manual;jurisdiction 空表示未定(合法)。
func normalizeTaxChannel(e *LegalEntity) {
	if e.TaxChannel == "" {
		e.TaxChannel = "manual"
	}
}

// CreateLegalEntity 新建法人(code 公司内唯一),返回自增 id。
func (s *PGStore) CreateLegalEntity(ctx context.Context, e LegalEntity) (int64, error) {
	normalizeTaxChannel(&e)
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO legal_entities(code, name, tax_jurisdiction, tax_channel)
		VALUES($1,$2,$3,$4) RETURNING id`, e.Code, e.Name, e.TaxJurisdiction, e.TaxChannel).
		Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("user: create legal_entity: %w", err)
	}
	return id, nil
}

// UpdateLegalEntity 编辑法人(code/name);未命中返回 ErrNotFound。
func (s *PGStore) UpdateLegalEntity(ctx context.Context, id int64, e LegalEntity) error {
	normalizeTaxChannel(&e)
	tag, err := s.db.Exec(ctx,
		`UPDATE legal_entities SET code=$2, name=$3, tax_jurisdiction=$4, tax_channel=$5
		WHERE id=$1`, id, e.Code, e.Name, e.TaxJurisdiction, e.TaxChannel)
	if err != nil {
		return fmt.Errorf("user: update legal_entity: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListMenuPermMatrix 菜单权限矩阵:角色列 × menu:* 权限行(行内含持有角色码)。
func (s *PGStore) ListMenuPermMatrix(ctx context.Context) (MenuPermMatrix, error) {
	roles, err := s.listRoles(ctx)
	if err != nil {
		return MenuPermMatrix{}, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT p.code, p.name, COALESCE(array_agg(r.code ORDER BY r.code)
			FILTER (WHERE r.code IS NOT NULL), '{}')
		FROM permissions p
		LEFT JOIN role_permissions rp ON rp.permission_id = p.id
		LEFT JOIN roles r ON r.id = rp.role_id
		WHERE p.code LIKE 'menu:%'
		GROUP BY p.code, p.name ORDER BY p.code`)
	if err != nil {
		return MenuPermMatrix{}, fmt.Errorf("user: list menu perm matrix: %w", err)
	}
	defer rows.Close()
	out := MenuPermMatrix{RoleColumns: roles, Rows: make([]MenuPermRow, 0)}
	for rows.Next() {
		var r MenuPermRow
		if err := rows.Scan(&r.Code, &r.Name, &r.Roles); err != nil {
			return MenuPermMatrix{}, fmt.Errorf("user: scan menu perm: %w", err)
		}
		out.Rows = append(out.Rows, r)
	}
	return out, rows.Err()
}

// listRoles 角色列(code/name/is_builtin;内置在前)。
func (s *PGStore) listRoles(ctx context.Context) ([]MenuRoleCol, error) {
	rows, err := s.db.Query(ctx,
		`SELECT code, name, is_builtin FROM roles ORDER BY is_builtin DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("user: list roles: %w", err)
	}
	defer rows.Close()
	out := make([]MenuRoleCol, 0)
	for rows.Next() {
		var r MenuRoleCol
		if err := rows.Scan(&r.RoleCode, &r.RoleName, &r.IsBuiltin); err != nil {
			return nil, fmt.Errorf("user: scan role: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
