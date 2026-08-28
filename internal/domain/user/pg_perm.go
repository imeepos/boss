package user

// 权限判定(功能权限 RBAC 兜底 + 数据范围):自 pg.go 拆出(C 行数红线)。

import (
	"context"
	"fmt"
	"strings"
)

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

func parentOf(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[:i]
	}
	return ""
}
