package app

// portReserver 包装 resource.PGStore 为 order.PortReserver 接口。
// *resource.PGStore 同时实现 ResourceChecker 和 PortReserver，
// 但 order.NewPGStore 的 type switch 只匹配第一个 case，
// 所以需要显式包装成不同类型以分别注入。

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/resource"
)

type portReserver struct{ svc *resource.PGStore }

func (r portReserver) ReserveFirstAvailable(ctx context.Context, addressID, orderID int64) (int64, error) {
	return r.svc.ReserveFirstAvailable(ctx, addressID, orderID)
}
