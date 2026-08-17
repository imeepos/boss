package app

// Application 持有各域服务的装配结果。
// 阶段1 起逐步注入:用户/权限 → 客户 → 资产 → 资源 → 订单/计费 → ...
// 各域只暴露接口,域间依赖在此绑定,禁止跨域直接 import 实现(见 docs/ADR-001)。
type Application struct {
	// UserSvc    user.Service
	// AssetSvc   asset.Service
	// OrderSvc   order.Service
}

// New 装配依赖并返回 Application。
func New() (*Application, error) {
	return &Application{}, nil
}
