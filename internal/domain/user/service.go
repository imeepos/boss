package user

import "context"

// Service 阶段1:账号/角色/权限/组织(子公司/部门/岗位/经营区域)/数据范围/区域地址层级。
// 权限变更即时生效:RBAC 快照写 Redis,校验走快照;数据范围走服务端快照,避免 token 膨胀。
type Service interface {
	Login(ctx context.Context, username, password string) (token string, err error)

	// HasPermission 功能权限判定(RBAC 快照)。
	HasPermission(ctx context.Context, accountID int64, permCode string) (bool, error)
	// HasDataScope 数据范围判定:资源属主组织是否落在账号数据范围内,越权拒并审计。
	HasDataScope(ctx context.Context, accountID int64, owner DataScope) (bool, error)

	ListAddresses(ctx context.Context, parentID int64) ([]Address, error)
	ImportAddresses(ctx context.Context, rows []AddressRow) (imported int, err error)

	// 组织实体(子公司/部门/岗位/经营区域)的只读查询与管理。
	ListRegions(ctx context.Context, parentPath string) ([]Region, error)
	ListLegalEntities(ctx context.Context) ([]LegalEntity, error)
	ListDepartments(ctx context.Context, legalEntityID int64) ([]Department, error)
	ListPosts(ctx context.Context, deptID int64) ([]Post, error)
	GetDataScope(ctx context.Context, accountID int64) (DataScope, error)
}

type Address struct {
	ID       int64
	ParentID int64
	Level    int8 // 1市 2区 3街道 4小区 5楼栋
	Name     string
}

type AddressRow struct {
	Path string // ltree 路径,如 bj.chaoyang.wangjing.xq1.ld2(唯一权威,见 docs/ADR-002)
	Name string
	// level 与 parent_id 为派生列,由服务侧按 nlevel(path)、subpath(path,0,-1) 反查计算,调用方无需提供。
}
