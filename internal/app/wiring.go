package app

import (
	"errors"
	"time"

	"github.com/ymm-001/boss/internal/domain/user"
)

// Application 持有各域服务的装配结果,是模块化单体依赖绑定的唯一入口。
//
// 依赖倒置(见 docs/ADR-001):域之间只经接口依赖;本层把「接口 → 实现」绑定。
// 增量装配(见 docs/architecture-review.md 发现 3.4):
//   - 每个阶段只构造已实现的域服务;未实现的域,其字段保持 nil。
//   - 已声明的接口让编译期就能锁定域间契约,未来按域拆分只改这里的绑定与传输层。
type Application struct {
	User UserService
	// 以下域随阶段逐步实现接口并注入(阶段 2-9),当前以占位接口锁定边界。
	// Customer    CustomerService
	// Asset       AssetService
	// Resource    ResourceService
	// Order       OrderService
	// Billing     BillingService
	// QuadLink    QuadLinkService
	// AAA         AAAService
	// Device      DeviceService
	// Provision   ProvisionService
	// GIS         GISService
	// Analytics   AnalyticsService
}

// ErrNotImplemented 域尚未在该阶段接入装配时返回,便于调用方给出明确降级/提示。
var ErrNotImplemented = errors.New("app: domain service not wired yet")

// UserService 阶段1 账号/角色/权限/组织(子公司/部门/岗位/经营区域)/数据范围/区域地址。
// 与 internal/domain/user.Service 语义一致;后续把字段直接绑定到域的接口实现。
type UserService interface {
	// Login 校验账号口令,签发 token(权限校验走 RBAC 快照,不塞 token)。
	Login(username, password string) (token string, err error)
	// HasPermission 权限快照查询(权限变更即时生效)。
	HasPermission(accountID int64, permCode string) (bool, error)
	// HasDataScope 数据范围判定:资源属主组织是否落在账号数据范围内,越权拒并审计。
	HasDataScope(accountID int64, owner user.DataScope) (bool, error)

	// 组织实体只读查询(供管理后台「子公司/部门/岗位/经营区域」菜单)。
	ListRegions(parentPath string) ([]user.Region, error)
	ListLegalEntities() ([]user.LegalEntity, error)
	ListDepartments(legalEntityID int64) ([]user.Department, error)
	ListPosts(deptID int64) ([]user.Post, error)
	GetDataScope(accountID int64) (user.DataScope, error)
}

// New 装配依赖并返回 Application。
// 阶段1 尚无完整实现,返回空装配但结构可编译;各阶段在对应域实现后于此注入。
func New() (*Application, error) {
	_ = time.Now // 占位,阶段实现装配后移除
	return &Application{}, nil
}
