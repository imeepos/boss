package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("user: not found")

// ErrUnauthorized 账号不存在/口令错误/已停用。
var ErrUnauthorized = errors.New("user: unauthorized")

// dbtx 是 PGStore 依赖的最小数据库接口:*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PGStore 是 Service 接口的 PostgreSQL 实现(阶段1:组织/账号/数据范围)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
// JWT 签发/校验不在域内(D1:单事实源),由 app 层经 auth.Manager 处理。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// Login 校验账号口令,返回认证身份;账号不存在/口令错误/停用统一返回 ErrUnauthorized。
func (s *PGStore) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	var id int64
	var realName, roleCode, roleName, hash string
	var status int16
	err := s.db.QueryRow(ctx, `
		SELECT a.id, a.real_name, r.code, r.name, a.password_hash, a.status
		FROM accounts a JOIN roles r ON a.role_id = r.id
		WHERE a.username = $1`, username).
		Scan(&id, &realName, &roleCode, &roleName, &hash, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("user: login query: %w", err)
	}
	if status != 1 {
		return nil, ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return nil, ErrUnauthorized
	}
	return &LoginResult{AccountID: id, Username: username, RealName: realName, RoleCode: roleCode, RoleName: roleName}, nil
}

// ListLegalEntities 列出全部子公司/法人(含平台总公司标志)。
func (s *PGStore) ListLegalEntities(ctx context.Context) ([]LegalEntity, error) {
	rows, err := s.db.Query(ctx, `SELECT id, code, name, is_platform FROM legal_entities ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("user: list legal_entities: %w", err)
	}
	defer rows.Close()
	out := make([]LegalEntity, 0)
	for rows.Next() {
		var e LegalEntity
		if err := rows.Scan(&e.ID, &e.Code, &e.Name, &e.IsPlatform); err != nil {
			return nil, fmt.Errorf("user: scan legal_entity: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListRegions 列出经营区域(含覆盖主体);parentPath 为空返回全部,否则返回该子树(ltree 前缀)。
func (s *PGStore) ListRegions(ctx context.Context, parentPath string) ([]Region, error) {
	query := `SELECT r.id, r.path, r.level, r.name, COALESCE(r.legal_entity_id,0), COALESCE(le.name,'')
		FROM regions r LEFT JOIN legal_entities le ON le.id = r.legal_entity_id ORDER BY r.path`
	args := []any{}
	if parentPath != "" {
		query = `SELECT r.id, r.path, r.level, r.name, COALESCE(r.legal_entity_id,0), COALESCE(le.name,'')
			FROM regions r LEFT JOIN legal_entities le ON le.id = r.legal_entity_id
			WHERE r.path <@ $1::ltree ORDER BY r.path`
		args = append(args, parentPath)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("user: list regions: %w", err)
	}
	defer rows.Close()
	out := make([]Region, 0)
	for rows.Next() {
		var r Region
		if err := rows.Scan(&r.ID, &r.Path, &r.Level, &r.Name, &r.LegalEntityID, &r.LegalEntityName); err != nil {
			return nil, fmt.Errorf("user: scan region: %w", err)
		}
		r.Parent = parentOf(r.Path)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListDepartments 列出部门;legalEntityID=0 返回全部,否则按子公司过滤。
func (s *PGStore) ListDepartments(ctx context.Context, legalEntityID int64) ([]Department, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.id, d.legal_entity_id, COALESCE(le.name, ''), d.name
		FROM departments d
		LEFT JOIN legal_entities le ON le.id = d.legal_entity_id
		WHERE ($1 = 0 OR d.legal_entity_id = $1)
		ORDER BY d.id`, legalEntityID)
	if err != nil {
		return nil, fmt.Errorf("user: list departments: %w", err)
	}
	defer rows.Close()
	out := make([]Department, 0)
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.LegalEntityID, &d.LegalEntity, &d.Name); err != nil {
			return nil, fmt.Errorf("user: scan department: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListPosts 列出岗位;deptID=0 返回全部,否则按部门过滤;Roles 聚合岗位绑定的角色码(逗号串)。
func (s *PGStore) ListPosts(ctx context.Context, deptID int64) ([]Post, error) {
	rows, err := s.db.Query(ctx, `
		SELECT p.id, p.code, p.name, p.dept_id, COALESCE(d.name, ''),
		       COALESCE(array_to_string(array_agg(r.code) FILTER (WHERE r.code IS NOT NULL), ','), '')
		FROM posts p
		LEFT JOIN departments d ON d.id = p.dept_id
		LEFT JOIN post_roles pr ON pr.post_id = p.id
		LEFT JOIN roles r ON r.id = pr.role_id
		WHERE ($1 = 0 OR p.dept_id = $1)
		GROUP BY p.id, p.code, p.name, p.dept_id, d.name
		ORDER BY p.id`, deptID)
	if err != nil {
		return nil, fmt.Errorf("user: list posts: %w", err)
	}
	defer rows.Close()
	out := make([]Post, 0)
	for rows.Next() {
		var p Post
		var roles string
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.DeptID, &p.DeptName, &roles); err != nil {
			return nil, fmt.Errorf("user: scan post: %w", err)
		}
		if roles != "" {
			p.Roles = strings.Split(roles, ",")
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetDataScope 读取账号数据范围;未命中返回 ErrNotFound。
func (s *PGStore) GetDataScope(ctx context.Context, accountID int64) (DataScope, error) {
	var ds DataScope
	err := s.db.QueryRow(ctx, `
		SELECT id, COALESCE(legal_entity_id, 0), COALESCE(dept_id, 0),
		       COALESCE(post_id, 0), COALESCE(region_scope::text, '')
		FROM accounts WHERE id = $1`, accountID).
		Scan(&ds.AccountID, &ds.LegalEntityID, &ds.DeptID, &ds.PostID, &ds.RegionScope)
	if errors.Is(err, pgx.ErrNoRows) {
		return DataScope{}, ErrNotFound
	}
	if err != nil {
		return DataScope{}, fmt.Errorf("user: get data scope: %w", err)
	}
	return ds, nil
}

// GetProfile 读取当前用户信息(联表角色名/公司名,承接 /auth/me);未命中返回 ErrNotFound。
func (s *PGStore) GetProfile(ctx context.Context, accountID int64) (*Profile, error) {
	var p Profile
	err := s.db.QueryRow(ctx, `
		SELECT a.id, a.username, a.real_name, COALESCE(a.phone, ''),
		       r.code, r.name,
		       COALESCE(le.name, ''), COALESCE(a.region_scope::text, '')
		FROM accounts a
		JOIN roles r ON a.role_id = r.id
		LEFT JOIN legal_entities le ON a.legal_entity_id = le.id
		WHERE a.id = $1`, accountID).
		Scan(&p.AccountID, &p.Username, &p.RealName, &p.Phone, &p.RoleCode, &p.RoleName, &p.LegalEntityName, &p.RegionScope)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user: get profile: %w", err)
	}
	return &p, nil
}

// ListAddresses 按 parentID 列子节点(parentID=0 顶层);锚点经树根 join 继承(迁移 000040)。
func (s *PGStore) ListAddresses(ctx context.Context, parentID int64) ([]Address, error) {
	rows, err := s.db.Query(ctx, `
		SELECT a.id, COALESCE(a.parent_id, 0), a.level, a.name,
		       COALESCE(r.country_code, ''), COALESCE(r.admin_code, ''),
		       EXISTS(SELECT 1 FROM addresses c WHERE c.parent_id = a.id)
		FROM addresses a
		JOIN addresses r ON r.path = subpath(a.path, 0, 1)
		WHERE CASE WHEN $1 = 0 THEN a.parent_id IS NULL ELSE a.parent_id = $1 END
		ORDER BY a.path`, parentID)
	if err != nil {
		return nil, fmt.Errorf("user: list addresses: %w", err)
	}
	defer rows.Close()
	out := make([]Address, 0)
	for rows.Next() {
		var a Address
		if err := rows.Scan(&a.ID, &a.ParentID, &a.Level, &a.Name,
			&a.CountryCode, &a.AdminCode, &a.HasChildren); err != nil {
			return nil, fmt.Errorf("user: scan address: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// HasPermission 功能权限判定:账号角色是否绑定该权限码。
// 生产走 Redis RBAC 快照;此处为 PG 直查兜底(快照未命中时回源)。
func (s *PGStore) HasPermission(ctx context.Context, accountID int64, permCode string) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM role_permissions rp
			JOIN accounts a ON a.role_id = rp.role_id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE a.id = $1 AND p.code = $2
		)`, accountID, permCode).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("user: has permission: %w", err)
	}
	return ok, nil
}

// HasDataScope 数据范围判定:资源属主组织是否落在账号数据范围内,越权拒并审计。
func (s *PGStore) HasDataScope(ctx context.Context, accountID int64, owner DataScope) (bool, error) {
	scope, err := s.GetDataScope(ctx, accountID)
	if err != nil {
		return false, err
	}
	// 组织维度:账号限定子公司/部门/岗位时,资源属主必须一致。
	if scope.LegalEntityID != 0 && scope.LegalEntityID != owner.LegalEntityID {
		return false, nil
	}
	if scope.DeptID != 0 && scope.DeptID != owner.DeptID {
		return false, nil
	}
	if scope.PostID != 0 && scope.PostID != owner.PostID {
		return false, nil
	}
	// 区域维度:账号限定区域子树时,资源区域路径必须落在子树内。
	if scope.RegionScope != "" && !inSubtree(scope.RegionScope, owner.RegionScope) {
		return false, nil
	}
	return true, nil
}

// inSubtree 判定 ownerPath 是否等于 scopePath 或落在其子树(ltree 前缀)。
func inSubtree(scopePath, ownerPath string) bool {
	return ownerPath == scopePath || strings.HasPrefix(ownerPath, scopePath+".")
}

// parentOf 由 ltree 物化路径反查父路径(subpath(path,0,-1) 的应用层等价实现)。
func parentOf(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[:i]
	}
	return ""
}
