package user

// 组织与数据权限模型(阶段1扩展)。
// 原则:角色=功能权限(HasPermission),组织=数据范围(HasDataScope),两者正交。
// 落地依据 docs/admin/multi-org-audit.md 五维审查结论,数据模型见 migrations/000002。

// Region 经营区域节点(集团/大区/省/城市,与地址地理树解耦)。
type Region struct {
	ID     int64  `json:"id"`
	Path   string `json:"path"`  // 物化路径,如 root.luzon.ncr.manila(唯一权威)
	Level  int8   `json:"level"` // 冗余列 = nlevel(path):1集团 2大区 3省 4城市
	Name   string `json:"name"`
	Parent string `json:"parent"` // 父路径 = subpath(path,0,-1)
}

// LegalEntity 子公司/法人(品牌隔离的最小隔离单元)。
type LegalEntity struct {
	ID   int64  `json:"id"`
	Code string `json:"code"` // LEG-A / LEG-B / LEG-C
	Name string `json:"name"`
}

// Department 部门(挂靠子公司)。
type Department struct {
	ID            int64  `json:"id"`
	LegalEntityID int64  `json:"legalEntityId"`
	LegalEntity   string `json:"legalEntity"` // 冗余展示:所属子公司名
	Name          string `json:"name"`
}

// Post 岗位(组织编制,与角色分离)。
type Post struct {
	ID       int64    `json:"id"`
	Code     string   `json:"code"` // dispatcher / cashier / agent / field_tech ...
	Name     string   `json:"name"`
	DeptID   int64    `json:"deptId"`
	DeptName string   `json:"deptName"` // 冗余展示
	Roles    []string `json:"roles"`    // 岗位绑定的功能角色码(如 technician/ops)
}

// PostRole 岗位→角色(功能权限)绑定,供 Post 内聚合展示。
type PostRole struct {
	RoleCode string `json:"roleCode"`
	RoleName string `json:"roleName"`
}

// DataScope 账号数据范围:资源属主归属组织,越权拒并审计。
type DataScope struct {
	AccountID     int64  `json:"accountId"`
	LegalEntityID int64  `json:"legalEntityId"` // 0=不限子公司(全集团)
	DeptID        int64  `json:"deptId"`        // 0=不限部门
	PostID        int64  `json:"postId"`        // 0=不限岗位
	RegionScope   string `json:"regionScope"`   // 空=全集团;非空=限该 path 子树
}
