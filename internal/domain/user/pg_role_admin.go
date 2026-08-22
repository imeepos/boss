// 自定义角色维护(migrations/000100):内置 7 角色只读,派生角色权限集全量替换。
// 模板复用 = 前端拉取内置角色权限集作为初始勾选,后端只接收最终权限码全集。
package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrRoleProtected 内置角色不可改删;被账号/岗位引用时删除返回 ErrConflict。
var ErrRoleProtected = errors.New("user: role protected")

// RoleDetail 角色详情(含权限码全集)。
type RoleDetail struct {
	ID              int64    `json:"id"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	IsBuiltin       bool     `json:"isBuiltin"`
	PermissionCodes []string `json:"permissionCodes"`
}

// Permission 权限码清单行。
type Permission struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// randCode 生成自定义角色码(custom_ + 12hex),熵源失败退回毫秒时间戳。
func randCode() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("custom_%d", time.Now().UnixMilli())
	}
	return "custom_" + hex.EncodeToString(b)
}

// ListPermissions 全量权限码清单(角色编辑抽屉数据源)。
func (s *PGStore) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := s.db.Query(ctx, `SELECT code, name FROM permissions ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("user: list permissions: %w", err)
	}
	defer rows.Close()
	out := make([]Permission, 0)
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Code, &p.Name); err != nil {
			return nil, fmt.Errorf("user: scan permission: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListRoleDetails 角色详情全集(含内置标记与权限码聚合)。
func (s *PGStore) ListRoleDetails(ctx context.Context) ([]RoleDetail, error) {
	rows, err := s.db.Query(ctx, `
		SELECT r.id, r.code, r.name, r.is_builtin,
		       COALESCE(array_agg(p.code ORDER BY p.code) FILTER (WHERE p.code IS NOT NULL), '{}')
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN permissions p ON p.id = rp.permission_id
		GROUP BY r.id, r.code, r.name, r.is_builtin
		ORDER BY r.is_builtin DESC, r.id`)
	if err != nil {
		return nil, fmt.Errorf("user: list role details: %w", err)
	}
	defer rows.Close()
	out := make([]RoleDetail, 0)
	for rows.Next() {
		var d RoleDetail
		if err := rows.Scan(&d.ID, &d.Code, &d.Name, &d.IsBuiltin, &d.PermissionCodes); err != nil {
			return nil, fmt.Errorf("user: scan role detail: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// dedupePerms 去重去空白权限码。
func dedupePerms(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, dup := seen[c]; dup {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}

// replaceRolePerms 事务内全量替换角色权限绑定;未知权限码触发 FK 拒绝。
func replaceRolePerms(ctx context.Context, tx dbtx, roleID int64, codes []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return fmt.Errorf("user: clear role perms: %w", err)
	}
	for _, c := range codes {
		_, err := tx.Exec(ctx, `
			INSERT INTO role_permissions(role_id, permission_id)
			VALUES($1, (SELECT id FROM permissions WHERE code = $2))`, roleID, c)
		if isFKViolation(err) {
			return ErrFKViolation
		}
		if err != nil {
			return fmt.Errorf("user: insert role perm: %w", err)
		}
	}
	return nil
}

// CreateCustomRole 新建派生角色;名称必填唯一,权限码全集落库。返回新角色详情。
func (s *PGStore) CreateCustomRole(ctx context.Context, name string, permCodes []string) (*RoleDetail, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 64 {
		return nil, ErrInvalidInput
	}
	codes := dedupePerms(permCodes)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("user: create role begin: %w", err)
	}
	defer tx.Rollback(ctx)
	var id int64
	var code string
	err = tx.QueryRow(ctx,
		`INSERT INTO roles(code, name, is_builtin) VALUES($1, $2, false) RETURNING id, code`,
		randCode(), name).Scan(&id, &code)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("user: create role: %w", err)
	}
	if err := replaceRolePerms(ctx, tx, id, codes); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("user: create role commit: %w", err)
	}
	return &RoleDetail{ID: id, Code: code, Name: name, IsBuiltin: false, PermissionCodes: codes}, nil
}

// UpdateCustomRole 编辑派生角色(改名 + 权限集全量替换);内置角色拒改。
func (s *PGStore) UpdateCustomRole(ctx context.Context, id int64, name string, permCodes []string) error {
	name = strings.TrimSpace(name)
	if id <= 0 || name == "" || len([]rune(name)) > 64 {
		return ErrInvalidInput
	}
	builtin, err := s.roleIsBuiltin(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if builtin {
		return ErrRoleProtected
	}
	codes := dedupePerms(permCodes)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("user: update role begin: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE roles SET name = $2 WHERE id = $1 AND is_builtin = false`, id, name)
	if err != nil {
		return fmt.Errorf("user: update role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err := replaceRolePerms(ctx, tx, id, codes); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteCustomRole 删除派生角色;被账号/岗位引用时拒(ErrConflict)。
func (s *PGStore) DeleteCustomRole(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidInput
	}
	builtin, err := s.roleIsBuiltin(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if builtin {
		return ErrRoleProtected
	}
	var inUse bool
	err = s.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM accounts WHERE role_id = $1)
		    OR EXISTS(SELECT 1 FROM post_roles WHERE role_id = $1)`, id).Scan(&inUse)
	if err != nil {
		return fmt.Errorf("user: check role in use: %w", err)
	}
	if inUse {
		return ErrConflict
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("user: delete role begin: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, id); err != nil {
		return fmt.Errorf("user: delete role perms: %w", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM roles WHERE id = $1 AND is_builtin = false`, id)
	if err != nil {
		return fmt.Errorf("user: delete role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// roleIsBuiltin 角色是否内置;未命中返回 ErrNotFound。
func (s *PGStore) roleIsBuiltin(ctx context.Context, id int64) (bool, error) {
	var builtin bool
	err := s.db.QueryRow(ctx, `SELECT is_builtin FROM roles WHERE id = $1`, id).Scan(&builtin)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("user: role is builtin: %w", err)
	}
	return builtin, nil
}

// listAccountPermCodes 账号经角色持有的权限码全集(/auth/me 透出,前端动态菜单)。
func (s *PGStore) listAccountPermCodes(ctx context.Context, accountID int64) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT p.code FROM role_permissions rp
		JOIN accounts a ON a.role_id = rp.role_id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE a.id = $1 ORDER BY p.code`, accountID)
	if err != nil {
		return nil, fmt.Errorf("user: list account perms: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("user: scan account perm: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
