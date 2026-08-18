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

	// EnsureSuperAdmin 启动引导:幂等创建超级管理员(sysadmin),已存在则跳过不覆盖。
	EnsureSuperAdmin(ctx context.Context, username, password, realName string) (created bool, err error)

	// HasPermission 功能权限判定(RBAC 快照)。
	HasPermission(ctx context.Context, accountID int64, permCode string) (bool, error)
	// HasDataScope 数据范围判定:资源属主组织是否落在账号数据范围内,越权拒并审计。
	HasDataScope(ctx context.Context, accountID int64, owner DataScope) (bool, error)

	ListAddresses(ctx context.Context, parentID int64) ([]Address, error)
	ImportAddresses(ctx context.Context, rows []AddressRow) (imported int, err error)
	// SetAddressGeo 挂接国家/一级行政区锚点(仅 level=1 根节点,承接迁移 000040)。
	SetAddressGeo(ctx context.Context, id int64, countryCode, adminCode string) error
	// ListUnlinkedRoots 未挂国家的根节点清单(回填工作台)。
	ListUnlinkedRoots(ctx context.Context) ([]Address, error)
	// CreateAddress 新增节点:parentID=0 为根节点(可带锚点),label 为 path 段(小写字母数字)。
	CreateAddress(ctx context.Context, parentID int64, label, name, countryCode, adminCode string) (int64, error)
	// UpdateAddressName 改名(path 权威不可变,见 ADR-002)。
	UpdateAddressName(ctx context.Context, id int64, name string) error
	// DeleteAddress 删除叶节点;有子节点或被业务表引用则拒(ErrConflict)。
	DeleteAddress(ctx context.Context, id int64) error
	// SearchAddresses 关键字搜全树(名称/path/锚点),返回命中节点及其祖先链(前端自动展开用)。
	SearchAddresses(ctx context.Context, kw string) ([]AddressHit, error)

	// 组织实体(子公司/部门/岗位/经营区域)的只读查询与管理。
	ListRegions(ctx context.Context, parentPath string) ([]Region, error)
	ListLegalEntities(ctx context.Context) ([]LegalEntity, error)
	ListAccounts(ctx context.Context) ([]AccountRow, error)
	// ListDataScopes 账号数据范围清单(menu:datascope 保护;kw 过滤账号/姓名/角色)。
	ListDataScopes(ctx context.Context, kw string) ([]AccountRow, error)
	ListRoles(ctx context.Context) ([]Role, error)
	// CreateAccount/UpdateAccount 受权建号/改号(封闭模型,menu:account 保护)。
	CreateAccount(ctx context.Context, in AccountInput) (int64, error)
	UpdateAccount(ctx context.Context, id int64, in AccountInput) error
	// 业务参数热更(menu:params 保护)。
	ListParams(ctx context.Context) ([]Param, error)
	UpdateParam(ctx context.Context, key, value string, updatedBy int64) error
	// 导入任务记录(menu:importer)。
	RecordImportTask(ctx context.Context, kind string, operatorID int64, imported, failed int, detail map[string]any) error
	ListImportTasks(ctx context.Context) ([]ImportTask, error)
	CreateLegalEntity(ctx context.Context, e LegalEntity) (int64, error)
	UpdateLegalEntity(ctx context.Context, id int64, e LegalEntity) error
	ListMenuPermMatrix(ctx context.Context) (MenuPermMatrix, error)
	ListDepartments(ctx context.Context, legalEntityID int64) ([]Department, error)
	ListPosts(ctx context.Context, deptID int64) ([]Post, error)
	// 部门/岗位受权维护(建号前的基础数据写操作)。
	CreateDepartment(ctx context.Context, legalEntityID int64, name string) (int64, error)
	UpdateDepartment(ctx context.Context, id, legalEntityID int64, name string) error
	CreatePost(ctx context.Context, deptID int64, code, name string, roles []string) (int64, error)
	UpdatePost(ctx context.Context, id, deptID int64, code, name string, roles []string) error
	GetDataScope(ctx context.Context, accountID int64) (DataScope, error)
	GetProfile(ctx context.Context, accountID int64) (*Profile, error)
}

type Address struct {
	ID          int64  `json:"id"`
	ParentID    int64  `json:"parentId"`
	Level       int8   `json:"level"` // 1市 2区 3街道 4小区 5楼栋
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"` // 所在树根的国家锚点(alpha-2,空=未挂接)
	AdminCode   string `json:"adminCode"`   // 所在树根的一级行政区锚点(ISO 3166-2,可空)
	HasChildren bool   `json:"hasChildren"` // 是否存在子级
}

type AddressRow struct {
	Path string // ltree 路径,如 bj.chaoyang.wangjing.xq1.ld2(唯一权威,见 docs/ADR-002)
	Name string
	// level 与 parent_id 为派生列,由服务侧按 nlevel(path)、subpath(path,0,-1) 反查计算,调用方无需提供。
	CountryCode string // 可选,仅 level=1 根节点生效;非根行忽略
	AdminCode   string // 可选,同上;须与国家前缀一致(DB CHECK 兜底)
}

// AddressHit 搜索命中:节点 + 根到父的祖先链(按 level 升序,前端逐层自动展开)。
type AddressHit struct {
	Node      Address   `json:"node"`
	Ancestors []Address `json:"ancestors"`
}
