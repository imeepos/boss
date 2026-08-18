package user

import "context"

// Service 阶段1:账号/角色/权限/组织(子公司/部门/岗位/经营区域)/数据范围/区域地址层级。
// 权限变更即时生效:RBAC 快照写 Redis,校验走快照;数据范围走服务端快照,避免 token 膨胀。
// LoginResult 认证成功后的账号身份(不含 token;token 由 app 层经 auth.Manager 签发)。
type LoginResult struct {
	AccountID int64
	Username  string // 登录名
	RealName  string // 姓名快照(展示)
	RoleCode  string // 角色码
	RoleName  string // 角色名(展示)
}

// Profile 当前登录用户信息(承接 /auth/me header 展示)。
type Profile struct {
	AccountID       int64  `json:"accountId"`
	Username        string `json:"username"`
	RealName        string `json:"realName"`
	RoleCode        string `json:"roleCode"`
	RoleName        string `json:"roleName"`
	LegalEntityName string `json:"legalEntityName"`
	RegionScope     string `json:"regionScope"` // 空=全集团
}

type Service interface {
	Login(ctx context.Context, username, password string) (*LoginResult, error)

	// Register 自助注册(阶段1 基础功能):默认 ops 角色,成功后即可登录。
	Register(ctx context.Context, username, password, realName string) (*LoginResult, error)

	// HasPermission 功能权限判定(RBAC 快照)。
	HasPermission(ctx context.Context, accountID int64, permCode string) (bool, error)
	// HasDataScope 数据范围判定:资源属主组织是否落在账号数据范围内,越权拒并审计。
	HasDataScope(ctx context.Context, accountID int64, owner DataScope) (bool, error)

	ListAddresses(ctx context.Context, parentID int64) ([]Address, error)
	ImportAddresses(ctx context.Context, rows []AddressRow) (imported int, err error)

	// 组织实体(子公司/部门/岗位/经营区域)的只读查询与管理。
	ListRegions(ctx context.Context, parentPath string) ([]Region, error)
	ListLegalEntities(ctx context.Context) ([]LegalEntity, error)
	ListAccounts(ctx context.Context) ([]AccountRow, error)
	CreateLegalEntity(ctx context.Context, e LegalEntity) (int64, error)
	UpdateLegalEntity(ctx context.Context, id int64, e LegalEntity) error
	ListMenuPermMatrix(ctx context.Context) (MenuPermMatrix, error)
	ListDepartments(ctx context.Context, legalEntityID int64) ([]Department, error)
	ListPosts(ctx context.Context, deptID int64) ([]Post, error)
	GetDataScope(ctx context.Context, accountID int64) (DataScope, error)
	GetProfile(ctx context.Context, accountID int64) (*Profile, error)
}

type Address struct {
	ID       int64  `json:"id"`
	ParentID int64  `json:"parentId"`
	Level    int8   `json:"level"` // 1市 2区 3街道 4小区 5楼栋
	Name     string `json:"name"`
}

type AddressRow struct {
	Path string // ltree 路径,如 bj.chaoyang.wangjing.xq1.ld2(唯一权威,见 docs/ADR-002)
	Name string
	// level 与 parent_id 为派生列,由服务侧按 nlevel(path)、subpath(path,0,-1) 反查计算,调用方无需提供。
}
