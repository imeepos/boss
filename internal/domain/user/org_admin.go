// 组织实体写操作(SYS 域,阶段1):部门/岗位的受权创建与编辑(建号前的基础数据维护)。
package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrFKViolation = errors.New("user: foreign key violation")

// AssignRegionCoverage 区域挂/摘运营主体覆盖(migrations/000076)。
// legalEntityID=0 摘除覆盖(该区域回落到祖先覆盖/总公司兜底);未命中区域返回 ErrNotFound。
func (s *PGStore) AssignRegionCoverage(ctx context.Context, regionID, legalEntityID int64) (err error) {
	if regionID <= 0 || legalEntityID < 0 {
		return ErrInvalidInput
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE regions SET legal_entity_id = NULLIF($2, 0) WHERE id = $1`,
		regionID, legalEntityID)
	if isFKViolation(err) {
		return ErrFKViolation
	}
	if err != nil {
		return fmt.Errorf("user: assign region coverage: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateDepartment 新建部门(挂靠子公司,子公司内同名唯一)。
func (s *PGStore) CreateDepartment(ctx context.Context, legalEntityID int64, name string) (int64, error) {
	if legalEntityID <= 0 || strings.TrimSpace(name) == "" || len([]rune(name)) > 64 {
		return 0, ErrInvalidInput
	}
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO departments(legal_entity_id, name) VALUES($1,$2) RETURNING id`,
		legalEntityID, strings.TrimSpace(name)).Scan(&id)
	if isUniqueViolation(err) {
		return 0, ErrConflict
	}
	if isFKViolation(err) {
		return 0, ErrFKViolation
	}
	if err != nil {
		return 0, fmt.Errorf("user: create department: %w", err)
	}
	return id, nil
}

// UpdateDepartment 编辑部门(改挂靠/改名);未命中返回 ErrNotFound。
func (s *PGStore) UpdateDepartment(ctx context.Context, id, legalEntityID int64, name string) error {
	if legalEntityID <= 0 || strings.TrimSpace(name) == "" || len([]rune(name)) > 64 {
		return ErrInvalidInput
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE departments SET legal_entity_id=$2, name=$3 WHERE id=$1`,
		id, legalEntityID, strings.TrimSpace(name))
	if isUniqueViolation(err) {
		return ErrConflict
	}
	if isFKViolation(err) {
		return ErrFKViolation
	}
	if err != nil {
		return fmt.Errorf("user: update department: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreatePost 新建岗位(部门内 code 唯一;roles=绑定的功能角色码,可为空)。
func (s *PGStore) CreatePost(ctx context.Context, deptID int64, code, name string, roles []string) (int64, error) {
	if deptID <= 0 || !isValidPostCode(code) || strings.TrimSpace(name) == "" || len([]rune(name)) > 64 {
		return 0, ErrInvalidInput
	}
	if err := s.db.QueryRow(ctx, `SELECT 1 FROM departments WHERE id=$1`, deptID).Scan(new(int)); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("user: post dept check: %w", err)
	}
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO posts(dept_id, code, name) VALUES($1,$2,$3) RETURNING id`,
		deptID, code, strings.TrimSpace(name)).Scan(&id)
	if isUniqueViolation(err) {
		return 0, ErrConflict
	}
	if err != nil {
		return 0, fmt.Errorf("user: create post: %w", err)
	}
	if err := s.replacePostRoles(ctx, id, roles); err != nil {
		return id, err
	}
	return id, nil
}

// UpdatePost 编辑岗位(改名/换部门/重绑角色);未命中返回 ErrNotFound。
func (s *PGStore) UpdatePost(ctx context.Context, id, deptID int64, code, name string, roles []string) error {
	if deptID <= 0 || !isValidPostCode(code) || strings.TrimSpace(name) == "" || len([]rune(name)) > 64 {
		return ErrInvalidInput
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE posts SET dept_id=$2, code=$3, name=$4 WHERE id=$1`,
		id, deptID, code, strings.TrimSpace(name))
	if isUniqueViolation(err) {
		return ErrConflict
	}
	if isFKViolation(err) {
		return ErrFKViolation
	}
	if err != nil {
		return fmt.Errorf("user: update post: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return s.replacePostRoles(ctx, id, roles)
}

// DeleteDepartment 删除部门;仍有岗位或在职账号挂靠时拒(ErrConflict);未命中 ErrNotFound。
func (s *PGStore) DeleteDepartment(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidInput
	}
	var occupied int
	if err := s.db.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM posts WHERE dept_id=$1)
		     + (SELECT count(*) FROM accounts WHERE dept_id=$1)`, id).
		Scan(&occupied); err != nil {
		return fmt.Errorf("user: delete department check: %w", err)
	}
	if occupied > 0 {
		return ErrConflict
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM departments WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("user: delete department: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeletePost 删除岗位(事务内连同 post_roles);仍有在职账号挂岗时拒(ErrConflict)。
func (s *PGStore) DeletePost(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidInput
	}
	var occupied int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM accounts WHERE post_id=$1`, id).Scan(&occupied); err != nil {
		return fmt.Errorf("user: delete post check: %w", err)
	}
	if occupied > 0 {
		return ErrConflict
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("user: delete post tx: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `DELETE FROM posts WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("user: delete post: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM post_roles WHERE post_id=$1`, id); err != nil {
		return fmt.Errorf("user: delete post roles: %w", err)
	}
	return tx.Commit(ctx)
}

// replacePostRoles 全量替换岗位→角色绑定;未知角色码返回 ErrRoleNotFound。
func (s *PGStore) replacePostRoles(ctx context.Context, postID int64, roles []string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("user: post roles tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM post_roles WHERE post_id=$1`, postID); err != nil {
		return fmt.Errorf("user: post roles delete: %w", err)
	}
	for _, code := range roles {
		var roleID int64
		err := tx.QueryRow(ctx, `SELECT id FROM roles WHERE code=$1`, code).Scan(&roleID)
		if err != nil {
			return ErrRoleNotFound
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO post_roles(post_id, role_id) VALUES($1,$2) ON CONFLICT DO NOTHING`,
			postID, roleID); err != nil {
			return fmt.Errorf("user: post roles insert: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// isValidPostCode 岗位代码:2-64 位,小写字母开头,字母/数字/下划线。
func isValidPostCode(code string) bool {
	n := len(code)
	if n < 2 || n > 64 || code[0] < 'a' || code[0] > 'z' {
		return false
	}
	for _, r := range code {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
		default:
			return false
		}
	}
	return true
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return err != nil && errors.As(err, &pgErr) && pgErr.Code == "23503"
}
