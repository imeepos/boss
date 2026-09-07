package odn

import (
	"context"
	"errors"
	"regexp"
)

// 领域错误。
var (
	// ErrNotFound 网格/设施不存在。
	ErrNotFound = errors.New("odn: not found")
	// ErrDuplicate 编码/网格冲突(唯一键)。
	ErrDuplicate = errors.New("odn: duplicate")
	// ErrInvalidCode 编码格式非法(红线 3:5 位数字网格分区模式)。
	ErrInvalidCode = errors.New("odn: invalid code")
	// ErrGridMissing 电杆/人井所属网格未备案(红线 2:网格分区方案须完整备案)。
	ErrGridMissing = errors.New("odn: grid not registered")
	// ErrGridFull 网格容量已满 999(规范 4.3/4.7)。
	ErrGridFull = errors.New("odn: grid full")
	// ErrRetired 编码已报废,永久锁定禁止复用(资产编码规范红线 2)。
	ErrRetired = errors.New("odn: code retired")
	// ErrSiteMissing 局点未备案或已退役(E16:site_no 无 FK,域层守护归属链)。
	ErrSiteMissing = errors.New("odn: site not registered")
)

// Kind 基础设施类型(规范 4.2)。
const (
	KindPole    = "P"   // 电杆:网格分区模式
	KindManhole = "MH"  // 人井:网格分区模式
	KindTower   = "TW"  // 铁塔:市域顺序
	KindClosure = "CLS" // 接头盒:市域顺序
	KindTermBox = "TBX" // 终端盒:市域顺序
)

// 网格状态(规范 4.6:20=预留扩展区)。
const (
	GridActive   = "ACTIVE"
	GridReserved = "RESERVED"
	GridRetired  = "RETIRED"
)

// 设施状态。
const (
	FacilityInUse  = "IN_USE"
	FacilityRetire = "RETIRED" // 报废:编码永久锁定,不可复用/删除
)

// 网格容量预警阈值(规范 4.7)。
const (
	GridWarnUsage = 800 // >=80% 容量:提交扩容预警
	GridMaxUsage  = 999 // 100%:启用预留网格码
	GridWarnCount = 90  // 城市网格数 >=90:启动细分评估
	MaxFacilities = 999 // 单网格容量
)

var (
	// gridKindCode 网格分区模式:P/MH + 2 位网格 + 3 位序号。
	gridKindCode = regexp.MustCompile(`^(P|MH)([0-9]{2})([0-9]{3})$`)
	// seqKindCode 顺序模式:TW/CLS/TBX + 5 位顺序号(00000 预留在代码层禁用)。
	seqKindCode = regexp.MustCompile(`^(TW|CLS|TBX)([0-9]{5})$`)
)

// Grid 网格分区(城市内 01~99)。
type Grid struct {
	PrvCode    string `json:"prvCode"`    // 所属 PRV,如 PHL001
	CityPrefix string `json:"cityPrefix"` // 城市前缀,如 MNL
	GridCode   int16  `json:"gridCode"`   // 01~99
	Name       string `json:"name"`       // 网格名称,如 马尼拉老城区
	Coverage   string `json:"coverage"`   // 覆盖区域描述
	Status     string `json:"status"`     // ACTIVE/RESERVED/RETIRED
	Facilities int64  `json:"facilities"` // 网格内设施占用数(P/MH)
}

// GridUsage 网格容量视图(规范 4.7 预警)。
type GridUsage struct {
	Grid
	Warn bool `json:"warn"` // 占用 >= 800 触发扩容预警
}

// Facility 基础设施(电杆/人井/铁塔/接头盒/终端盒)。
type Facility struct {
	Code            string  `json:"code"`       // P01001/MH01001/TW00001/CLS00001/TBX00001
	Kind            string  `json:"kind"`       // P/MH/TW/CLS/TBX
	PrvCode         string  `json:"prvCode"`    // 所属网格城市(仅 P/MH)
	CityPrefix      string  `json:"cityPrefix"` // 仅 P/MH
	GridCode        int16   `json:"gridCode"`   // 仅 P/MH
	Name            string  `json:"name"`
	Lat             float64 `json:"lat"`             // 纬度
	Lng             float64 `json:"lng"`             // 经度
	Status          string  `json:"status"`          // IN_USE/RETIRED
	LifecycleStatus string  `json:"lifecycleStatus"` // PLANNED/IN_BUILD/IN_SERVICE/RETIRED(000198)
}

// ODNService ODN 无源物理层域服务口。
type ODNService interface {
	// 网格分区。
	ListGrids(ctx context.Context, prvCode, cityPrefix string) ([]GridUsage, error)
	CreateGrid(ctx context.Context, g Grid) error
	UpdateGrid(ctx context.Context, prvCode, cityPrefix string, gridCode int16, g Grid) error
	RetireGrid(ctx context.Context, prvCode, cityPrefix string, gridCode int16) error

	// 网格投资测算读模型(P-INFRA-1 W2;纯只读聚合,无写路径)。
	GridInvestment(ctx context.Context) ([]GridInvestmentRow, error)

	// 基础设施。
	ListFacilities(ctx context.Context, kind string, gridFilter GridRef) ([]Facility, error)
	GetFacility(ctx context.Context, code string) (*Facility, error)
	CreateFacility(ctx context.Context, f Facility) error
	RetireFacility(ctx context.Context, code string) error

	// 光缆段落/纤芯(规范第 5 章)。
	CreateSegment(ctx context.Context, code1, code2, name string) (*Segment, error)
	ListSegments(ctx context.Context, endpoint string) ([]Segment, error)
	AddFiber(ctx context.Context, segID int64, f Fiber) error
	ListFibers(ctx context.Context, segID int64) ([]Fiber, error)

	// 局点与核心链路设备(资产编码规范第 2/3 章)。
	ListSites(ctx context.Context, prvCode, cityPrefix string) ([]Site, error)
	CreateSite(ctx context.Context, st Site) error
	RetireSite(ctx context.Context, prvCode, cityPrefix string, siteNo int16) error
	CreateDevice(ctx context.Context, d Device) error
	ListDevices(ctx context.Context, kind, prvCode, cityPrefix string) ([]Device, error)
	RetireDevice(ctx context.Context, id int64) error

	// 生命周期状态机(P6,迁移 000198;PLANNED→IN_BUILD→IN_SERVICE→RETIRED,RETIRED 终态)。
	SetFacilityLifecycle(ctx context.Context, code, to string) error
	SetSiteLifecycle(ctx context.Context, prvCode, cityPrefix string, siteNo int16, to string) error
	SetDeviceLifecycle(ctx context.Context, id int64, to string) error

	// 施工项目与竣工回填(P6,迁移 000199;线性 PENDING→BUILDING→ACCEPTED)。
	// 000204 扩展:承包商指定、工程量清单(数量/单价/金额后端计算)、工程结算。
	CreateProject(ctx context.Context, p Construction) error
	GetProject(ctx context.Context, id int64) (*Construction, error)
	ListProjects(ctx context.Context, limit int) ([]Construction, error)
	AddProjectItem(ctx context.Context, projectID int64, facilityCode string, qty, unitPrice float64) error
	UpdateProjectItem(ctx context.Context, projectID, itemID int64, qty, unitPrice float64) error
	SetProjectContractor(ctx context.Context, projectID, contractorID int64, contractorName string) error
	StartProject(ctx context.Context, id int64) (int64, error)
	// AcceptProject 返回 (设施翻转数, 覆盖联动数, 错误);F6 竣工覆盖联动随事务(000211)。
	AcceptProject(ctx context.Context, id int64, acceptedBy int64, note string) (int64, int64, error)
	ListProjectItems(ctx context.Context, id int64) ([]ConstructionItem, error)
	// RecordProgress 幂等进度上报(P0-B,000213):created=false=重复消息不重复计量。
	RecordProgress(ctx context.Context, p ProgressEntry) (int64, bool, error)
	ListProgress(ctx context.Context, projectID int64, limit int) ([]ProgressEntry, error)
	ItemProgressSummary(ctx context.Context, projectID int64) ([]ItemProgress, error)
	CreateSettlement(ctx context.Context, projectID, createdBy int64) (*Settlement, error)
	ListSettlements(ctx context.Context, projectID int64) ([]Settlement, error)
	GetSettlement(ctx context.Context, id int64) (*Settlement, error)
	SettleSettlement(ctx context.Context, id, accountID int64) error
	VoidSettlement(ctx context.Context, id, accountID int64, reason string) error

	// 物理端口占用态(P2,迁移 000200;下单门控数据基础)。
	ListPorts(ctx context.Context, deviceID int64) ([]ODNPort, error)
	AllocatePort(ctx context.Context, deviceID, orderID int64) (*ODNPort, error)
	AllocateForAddress(ctx context.Context, addressID, orderID int64) (*ODNPort, error)
	ReleasePort(ctx context.Context, portID int64) error
	ActivatePort(ctx context.Context, portID int64) error

	// 下单覆盖门控(P2,T12;灰度 BOSS_ODN_COVERAGE_GATE)。
	CheckOrderCoverage(ctx context.Context, addressID int64) error

	// 逻辑-物理绑定(P3,迁移 000201;售后与 GIS 反查数据源)。
	BindPort(ctx context.Context, b ODNBinding) error
	UnbindPort(ctx context.Context, portID int64) error
	ListBindingsByPort(ctx context.Context, portID int64) ([]ODNBinding, error)
	ListBindingsByOrder(ctx context.Context, orderID int64) ([]ODNBinding, error)

	// 资源链批量导入(P-INFRA-1 W3;说明页六规则;导入域箱体设备展开)。
	ImportResourceChains(ctx context.Context, in ChainImportInput) (*ChainImportResult, error)
	ListResourceChains(ctx context.Context, batch string, limit int) ([]ResourceChainView, error)
	PatrolResourceChains(ctx context.Context) (*ChainPatrol, error)

	// 影响面分析(P7,只读聚合;运维侧影响谁)。
	ImpactByFacility(ctx context.Context, facilityCode string) (*ImpactReport, error)

	// 覆盖关联(P1,迁移 000197;odn↔业务首桥)。
	UpsertCoverage(ctx context.Context, c Coverage) error
	GetCoverageByAddress(ctx context.Context, addressID int64) (*Coverage, error)
	ListCoverage(ctx context.Context, limit int) ([]Coverage, error)
	ResolveLatLng(ctx context.Context, lat, lng float64) (*CoverageResolved, error)
	// 许可单(P-INFRA-1 W4,迁移 000211;ROW 路权/PECE 许可,状态机 terms.md 4)。
	CreatePermit(ctx context.Context, p Permit, createdBy int64) (*Permit, error)
	GetPermit(ctx context.Context, id int64) (*Permit, error)
	ListPermits(ctx context.Context, kind, status string, projectID int64, unlinked bool, limit int) ([]Permit, error)
	UpdatePermitArchive(ctx context.Context, id int64, p Permit) error
	TransitionPermit(ctx context.Context, id, accountID int64, to, reason, approvalNo, validFrom, validUntil string) (*Permit, error)
	LinkPermitProject(ctx context.Context, id, projectID int64) error
	UnlinkPermitProject(ctx context.Context, id int64) error
	ListProjectPermits(ctx context.Context, projectID int64) ([]Permit, error)
	CheckProjectPermits(ctx context.Context, projectID int64) (*PermitGateReport, error)
}

// GridRef 网格定位(城市 + 网格码)。
type GridRef struct {
	PrvCode    string
	CityPrefix string
	GridCode   int16
}

// ValidateFacilityCode 校验编码格式(红线 3)并解析 kind/网格段。
// 返回 (kind, 网格码, 序号);网格模式返回网格码,顺序模式返回 0。
// 序号 000/00000 为预留禁用(规范 4.1:000 预留)。
func ValidateFacilityCode(code string) (kind string, grid int16, seq int, err error) {
	if m := gridKindCode.FindStringSubmatch(code); m != nil {
		seq := int(m[3][0]-'0')*100 + int(m[3][1]-'0')*10 + int(m[3][2]-'0')
		if seq == 0 {
			return "", 0, 0, ErrInvalidCode
		}
		return m[1], int16(m[2][0]-'0')*10 + int16(m[2][1]-'0'), seq, nil
	}
	if m := seqKindCode.FindStringSubmatch(code); m != nil {
		seq := int(m[2][0]-'0')*10000 + int(m[2][1]-'0')*1000 + int(m[2][2]-'0')*100 +
			int(m[2][3]-'0')*10 + int(m[2][4]-'0')
		if seq == 0 {
			return "", 0, 0, ErrInvalidCode
		}
		return m[1], 0, seq, nil
	}
	return "", 0, 0, ErrInvalidCode
}
